package main

import (
	"context"
	cryptorand "crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dovgalb/project-rupor/config"
	"github.com/dovgalb/project-rupor/internal/auth/domain"
	bcryptadapter "github.com/dovgalb/project-rupor/internal/auth/repository/bcrypt"
	jwtadapter "github.com/dovgalb/project-rupor/internal/auth/repository/jwt"
	"github.com/dovgalb/project-rupor/internal/auth/repository/postgres"
	"github.com/dovgalb/project-rupor/internal/auth/repository/postgres/db"
	httpauth "github.com/dovgalb/project-rupor/internal/auth/transport/http"
	authmw "github.com/dovgalb/project-rupor/internal/auth/transport/http/middleware"
	"github.com/dovgalb/project-rupor/internal/auth/usecase"
	channelpg "github.com/dovgalb/project-rupor/internal/channel/repository/postgres"
	httpchannel "github.com/dovgalb/project-rupor/internal/channel/transport/http"
	channelusecase "github.com/dovgalb/project-rupor/internal/channel/usecase"
	chatpg "github.com/dovgalb/project-rupor/internal/chat/repository/postgres"
	httpchat "github.com/dovgalb/project-rupor/internal/chat/transport/http"
	wschat "github.com/dovgalb/project-rupor/internal/chat/transport/ws"
	chatusecase "github.com/dovgalb/project-rupor/internal/chat/usecase"
	roompg "github.com/dovgalb/project-rupor/internal/room/repository/postgres"
	httproom "github.com/dovgalb/project-rupor/internal/room/transport/http"
	roomusecase "github.com/dovgalb/project-rupor/internal/room/usecase"
	httpxmw "github.com/dovgalb/project-rupor/pkg/httpx/middleware"
	pws "github.com/dovgalb/project-rupor/pkg/websocket"
)

const (
	shutdownTimeout      = 5 * time.Second
	bcryptProductionCost = 10
	dummyTimingPassword  = "dummy_password_for_timing_safety"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg, err := config.Load(config.OsLookuper{})
	if err != nil {
		var verr config.ValidationError
		if errors.As(err, &verr) {
			logger.Error("config invalid",
				slog.String("code", verr.Code),
				slog.String("field", verr.Field),
				slog.String("reason", verr.Reason),
			)
		} else {
			logger.Error("config load failed", slog.Any("err", err))
		}
		os.Exit(1)
	}

	if err := run(cfg, logger); err != nil {
		logger.Error("server stopped with error", slog.Any("err", err))
		os.Exit(1)
	}
}

func run(cfg *config.Config, logger *slog.Logger) error {
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL())
	if err != nil {
		return fmt.Errorf("pgxpool.New: %w", err)
	}
	defer pool.Close()

	queries := db.New(pool)
	userRepo := postgres.NewUserRepository(queries)
	refreshRepo := postgres.NewRefreshTokenRepository(pool)

	hasher := bcryptadapter.NewPasswordHasher(bcryptProductionCost)
	issuer := jwtadapter.NewTokenIssuer([]byte(cfg.JWTSecret()), cfg.JWTAccessTTL())

	clock := realClock{}
	uuids := realUUID{}
	randSrc := cryptoRand{}

	dummyPwd, err := domain.NewPassword(dummyTimingPassword)
	if err != nil {
		return fmt.Errorf("init dummy password: %w", err)
	}
	dummyHash, err := hasher.Hash(dummyPwd)
	if err != nil {
		return fmt.Errorf("init dummy hash: %w", err)
	}

	registerUC := usecase.NewRegisterUser(userRepo, hasher, clock, uuids)
	loginUC := usecase.NewLoginUser(
		userRepo, refreshRepo, hasher, issuer, clock, uuids, randSrc,
		cfg.JWTRefreshTTL(), dummyHash,
	)
	refreshUC := usecase.NewRefreshAccess(
		refreshRepo, issuer, clock, uuids, randSrc, cfg.JWTRefreshTTL(),
	)
	meUC := usecase.NewGetCurrentUser(userRepo)

	// === Room composition ===
	roomRepo := roompg.NewRoomRepository(pool)
	membershipRepo := roompg.NewMembershipRepository(pool)
	inviteRepo := roompg.NewInviteRepository(pool)
	inviteCodeGen := roompg.NewBase32CodeGen(cryptorand.Reader)

	createRoomUC := roomusecase.NewCreateRoom(roomRepo, clock, uuids)
	getRoomUC := roomusecase.NewGetRoom(roomRepo, membershipRepo)
	listRoomsUC := roomusecase.NewListUserRooms(roomRepo)
	deleteRoomUC := roomusecase.NewDeleteRoom(roomRepo, membershipRepo)
	listMembersUC := roomusecase.NewListMembers(membershipRepo)
	regenInviteUC := roomusecase.NewRegenerateInvite(inviteRepo, membershipRepo, inviteCodeGen, clock, uuids)

	// === Channel composition ===
	channelRepo := channelpg.NewChannelRepository(pool)
	membershipQuery := roompg.NewMembershipQueryAdapter(pool)

	createChannelUC := channelusecase.NewCreateChannel(channelRepo, membershipQuery, clock, uuids)
	listChannelsUC := channelusecase.NewListChannels(channelRepo, membershipQuery)
	deleteChannelUC := channelusecase.NewDeleteChannel(channelRepo, membershipQuery)

	// === WebSocket Hub ===
	hub := pws.NewHub(logger)

	// === Chat composition ===
	messageRepo := chatpg.NewMessageRepository(pool)
	membershipForChat := roompg.NewMembershipQueryChatAdapter(pool)
	chatBroadcaster := newHubChatBroadcaster(hub)
	roomEventsPublisher := newHubRoomEventsPublisher(hub)

	sendMessageUC := chatusecase.NewSendMessage(messageRepo, membershipForChat, chatBroadcaster, clock, uuids)
	listMessagesUC := chatusecase.NewListMessages(messageRepo, membershipForChat)

	// joinByCodeUC использует publisher; объявлен после hub.
	joinByCodeUC := roomusecase.NewJoinByCode(inviteRepo, membershipRepo, roomRepo, clock, roomEventsPublisher)

	mux := chi.NewRouter()

	uuidGen := func() string { return uuid.New().String() }
	userIDHook := func(ctx context.Context) []slog.Attr {
		uid, ok := authmw.UserIDFromContext(ctx)
		if !ok {
			return nil
		}
		return []slog.Attr{slog.String("user_id", uid.String())}
	}

	mux.Use(httpxmw.RequestID(uuidGen))
	mux.Use(httpxmw.Recover(logger))
	mux.Use(httpxmw.Logger(logger, userIDHook))
	mux.Use(httpxmw.CORS(cfg.CORSAllowedOrigins(), false))

	mux.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", healthHandler)
		httpauth.RegisterRoutes(r, httpauth.Deps{
			Register:    registerUC,
			Login:       loginUC,
			Refresh:     refreshUC,
			Me:          meUC,
			TokenIssuer: issuer,
			Clock:       clock,
		})
		httproom.RegisterRoutes(r, httproom.Deps{
			CreateRoom:       createRoomUC,
			GetRoom:          getRoomUC,
			ListUserRooms:    listRoomsUC,
			DeleteRoom:       deleteRoomUC,
			ListMembers:      listMembersUC,
			RegenerateInvite: regenInviteUC,
			JoinByCode:       joinByCodeUC,
			TokenIssuer:      issuer,
			Clock:            clock,
		})
		httpchannel.RegisterRoutes(r, httpchannel.Deps{
			CreateChannel: createChannelUC,
			ListChannels:  listChannelsUC,
			DeleteChannel: deleteChannelUC,
			TokenIssuer:   issuer,
			Clock:         clock,
		})
		httpchat.RegisterRoutes(r, httpchat.Deps{
			ListMessages: listMessagesUC,
			TokenIssuer:  issuer,
			Clock:        clock,
		})
		wschat.RegisterWSRoute(r, wschat.WSDeps{
			Hub:                hub,
			SendMessage:        sendMessageUC,
			MembershipForChat:  membershipForChat,
			MembershipForRooms: roomIDsAdapter{repo: roomRepo},
			TokenIssuer:        issuer,
			Clock:              clock,
			OriginPatterns:     stripScheme(cfg.CORSAllowedOrigins()),
		})
	})

	addr := ":" + strconv.Itoa(cfg.ServerPort())
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("server starting", slog.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
		close(serverErr)
	}()

	select {
	case err := <-serverErr:
		if err != nil {
			return err
		}
	case <-signalCtx.Done():
		logger.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	// Hub.Shutdown ДО srv.Shutdown — клиенты должны получить close-frame 1001
	// до того, как HTTP-сервер прекратит ответ.
	hub.Shutdown(shutdownCtx)

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}

	logger.Info("shutdown complete")
	return nil
}
