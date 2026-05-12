package wschat_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestWS_Connect_NoToken_Returns401(t *testing.T) {
	t.Parallel()
	env := setupServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, resp := dialWS(t, env, ctx, "")
	if resp == nil {
		t.Fatal("ожидали HTTP-ответ (401)")
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestWS_Connect_InvalidToken_Returns401(t *testing.T) {
	t.Parallel()
	env := setupServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, resp := dialWS(t, env, ctx, "garbage.token")
	if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %v, want 401", resp)
	}
}

func TestWS_Connect_ExpiredToken_Returns401(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	tok := issueExpiredToken(t, env, uuid.New())

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, resp := dialWS(t, env, ctx, tok)
	if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %v, want 401", resp)
	}
}

func TestWS_Connect_ValidToken_Succeeds(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	uid := uuid.New()
	tok := issueTestToken(t, env, uid)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	c, _ := dialWS(t, env, ctx, tok)
	if c == nil {
		t.Fatal("dial вернул nil-conn")
	}
}

func TestWS_Subscribe_AsMember_SendsSubscribedFrame(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	uid := uuid.New()
	channel := uuid.New()
	env.messages.withChannel(channel, uuid.New(), "text")
	env.membership.allow(channel, uid)
	tok := issueTestToken(t, env, uid)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	c, _ := dialWS(t, env, ctx, tok)
	if c == nil {
		t.Fatal("dial failed")
	}

	writeJSON(t, ctx, c, map[string]any{
		"type":       "subscribe",
		"channel_id": channel.String(),
	})

	frame := readJSON(t, ctx, c)
	if frame["type"] != "subscribed" {
		t.Fatalf("type = %v, want subscribed", frame["type"])
	}
}

func TestWS_Subscribe_NotMember_SendsErrorFrame_CHAT004(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	uid := uuid.New()
	channel := uuid.New()
	env.messages.withChannel(channel, uuid.New(), "text")
	// membership.allow не вызван.
	tok := issueTestToken(t, env, uid)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	c, _ := dialWS(t, env, ctx, tok)
	if c == nil {
		t.Fatal("dial failed")
	}

	writeJSON(t, ctx, c, map[string]any{
		"type":       "subscribe",
		"channel_id": channel.String(),
	})

	frame := readJSON(t, ctx, c)
	if frame["type"] != "error" {
		t.Fatalf("type = %v, want error", frame["type"])
	}
	data, _ := frame["data"].(map[string]any)
	if data["code"] != "CHAT-004" {
		t.Fatalf("code = %v, want CHAT-004", data["code"])
	}
}

func TestWS_Subscribe_InvalidChannelUUID_SendsErrorFrame_CHAT005(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	uid := uuid.New()
	tok := issueTestToken(t, env, uid)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	c, _ := dialWS(t, env, ctx, tok)
	if c == nil {
		t.Fatal("dial failed")
	}

	writeJSON(t, ctx, c, map[string]any{
		"type":       "subscribe",
		"channel_id": "bad-uuid",
	})

	frame := readJSON(t, ctx, c)
	data, _ := frame["data"].(map[string]any)
	if data["code"] != "CHAT-005" {
		t.Fatalf("code = %v, want CHAT-005", data["code"])
	}
}

func TestWS_MessageSend_AsMember_PersistsAndBroadcasts(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	// sender + listener — два пользователя в одной комнате/канале.
	sender := uuid.New()
	listener := uuid.New()
	channel := uuid.New()
	room := uuid.New()
	env.messages.withChannel(channel, room, "text")
	env.membership.allow(channel, sender)
	env.membership.allow(channel, listener)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	listenerTok := issueTestToken(t, env, listener)
	listenerConn, _ := dialWS(t, env, ctx, listenerTok)
	if listenerConn == nil {
		t.Fatal("listener dial failed")
	}
	// listener подписывается на канал.
	writeJSON(t, ctx, listenerConn, map[string]any{
		"type":       "subscribe",
		"channel_id": channel.String(),
	})
	_ = readJSON(t, ctx, listenerConn) // съесть subscribed frame

	senderTok := issueTestToken(t, env, sender)
	senderConn, _ := dialWS(t, env, ctx, senderTok)
	if senderConn == nil {
		t.Fatal("sender dial failed")
	}

	writeJSON(t, ctx, senderConn, map[string]any{
		"type":       "message.send",
		"channel_id": channel.String(),
		"text":       "hello",
	})

	// sender получает message.sent ack.
	ack := readJSON(t, ctx, senderConn)
	if ack["type"] != "message.sent" {
		t.Fatalf("ack type = %v, want message.sent", ack["type"])
	}

	// listener должен получить message.new через broadcast.
	bcast := readJSON(t, ctx, listenerConn)
	if bcast["type"] != "message.new" {
		t.Fatalf("broadcast type = %v, want message.new", bcast["type"])
	}
	if len(env.messages.saved) != 1 {
		t.Fatalf("saved = %d, want 1", len(env.messages.saved))
	}
}

func TestWS_MessageSend_NotMember_SendsErrorFrame_CHAT004(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	uid := uuid.New()
	channel := uuid.New()
	env.messages.withChannel(channel, uuid.New(), "text")
	tok := issueTestToken(t, env, uid)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	c, _ := dialWS(t, env, ctx, tok)
	if c == nil {
		t.Fatal("dial failed")
	}

	writeJSON(t, ctx, c, map[string]any{
		"type":       "message.send",
		"channel_id": channel.String(),
		"text":       "hi",
	})

	frame := readJSON(t, ctx, c)
	data, _ := frame["data"].(map[string]any)
	if data["code"] != "CHAT-004" {
		t.Fatalf("code = %v, want CHAT-004", data["code"])
	}
}

func TestWS_MessageSend_VoiceChannel_SendsErrorFrame_CHAT003(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	uid := uuid.New()
	channel := uuid.New()
	env.messages.withChannel(channel, uuid.New(), "voice")
	env.membership.allow(channel, uid)
	tok := issueTestToken(t, env, uid)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	c, _ := dialWS(t, env, ctx, tok)
	if c == nil {
		t.Fatal("dial failed")
	}

	writeJSON(t, ctx, c, map[string]any{
		"type":       "message.send",
		"channel_id": channel.String(),
		"text":       "hello voice",
	})

	frame := readJSON(t, ctx, c)
	data, _ := frame["data"].(map[string]any)
	if data["code"] != "CHAT-003" {
		t.Fatalf("code = %v, want CHAT-003", data["code"])
	}
}

func TestWS_UnknownEventType_SendsErrorFrame_CHAT007(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	uid := uuid.New()
	tok := issueTestToken(t, env, uid)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	c, _ := dialWS(t, env, ctx, tok)
	if c == nil {
		t.Fatal("dial failed")
	}

	writeJSON(t, ctx, c, map[string]any{"type": "unknown.command"})

	frame := readJSON(t, ctx, c)
	data, _ := frame["data"].(map[string]any)
	if data["code"] != "CHAT-007" {
		t.Fatalf("code = %v, want CHAT-007", data["code"])
	}
}
