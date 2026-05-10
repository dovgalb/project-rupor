package main

import (
	"context"
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
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dovgalb/project-rupor/config"
	"github.com/dovgalb/project-rupor/internal/auth/domain"
	bcryptadapter "github.com/dovgalb/project-rupor/internal/auth/repository/bcrypt"
	jwtadapter "github.com/dovgalb/project-rupor/internal/auth/repository/jwt"
	"github.com/dovgalb/project-rupor/internal/auth/repository/postgres"
	"github.com/dovgalb/project-rupor/internal/auth/repository/postgres/db"
	httpauth "github.com/dovgalb/project-rupor/internal/auth/transport/http"
	"github.com/dovgalb/project-rupor/internal/auth/usecase"
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

	mux := chi.NewRouter()
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

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}

	logger.Info("shutdown complete")
	return nil
}
