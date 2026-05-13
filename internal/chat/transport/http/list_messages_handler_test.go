package httpchat_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

func doGet(t *testing.T, env *testEnv, path, token string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, env.server.URL+path, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	return resp
}

func TestListMessages_AsMember_Returns200(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	uid := uuid.New()
	tok := issueTestToken(t, env, uid)
	channel := uuid.New()
	room := uuid.New()
	env.messages.withChannel(channel, room, "text")
	env.membership.allow(channel, uid)

	now := env.clock.now
	env.messages.put(mustMessage(t, uuid.New(), channel, uid, "first", now))
	env.messages.put(mustMessage(t, uuid.New(), channel, uid, "second", now.Add(time.Second)))

	resp := doGet(t, env, "/channels/"+channel.String()+"/messages", tok)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body struct {
		Items []struct {
			ID        string `json:"id"`
			ChannelID string `json:"channelId"`
			AuthorID  string `json:"authorId"`
			Text      string `json:"text"`
			CreatedAt string `json:"createdAt"`
		} `json:"items"`
		NextBefore *string `json:"nextBefore"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Items) != 2 {
		t.Fatalf("items len = %d, want 2", len(body.Items))
	}
	if body.Items[0].Text != "first" {
		t.Fatalf("Items[0].Text = %q, want first", body.Items[0].Text)
	}
	if body.Items[0].ChannelID != channel.String() {
		t.Fatalf("channelId mismatch")
	}
	if body.NextBefore != nil {
		t.Fatalf("NextBefore = %v, want nil", *body.NextBefore)
	}
}

func TestListMessages_NoToken_Returns401(t *testing.T) {
	t.Parallel()
	env := setupServer(t)

	resp := doGet(t, env, "/channels/"+uuid.New().String()+"/messages", "")
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestListMessages_ExpiredToken_Returns401(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	tok := issueExpiredToken(t, env, uuid.New())

	resp := doGet(t, env, "/channels/"+uuid.New().String()+"/messages", tok)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestListMessages_InvalidChannelUUID_Returns400_CHAT005(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	tok := issueTestToken(t, env, uuid.New())

	resp := doGet(t, env, "/channels/bad-uuid/messages", tok)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	var er errResp
	_ = json.NewDecoder(resp.Body).Decode(&er)
	if er.Error.Code != "CHAT-005" {
		t.Fatalf("code = %q, want CHAT-005", er.Error.Code)
	}
}

func TestListMessages_InvalidBeforeUUID_Returns400_CHAT005(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	tok := issueTestToken(t, env, uuid.New())

	resp := doGet(t, env, "/channels/"+uuid.New().String()+"/messages?before=bad", tok)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	var er errResp
	_ = json.NewDecoder(resp.Body).Decode(&er)
	if er.Error.Code != "CHAT-005" {
		t.Fatalf("code = %q, want CHAT-005", er.Error.Code)
	}
}

func TestListMessages_LimitZero_Returns400_CHAT006(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	tok := issueTestToken(t, env, uuid.New())

	resp := doGet(t, env, "/channels/"+uuid.New().String()+"/messages?limit=0", tok)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	var er errResp
	_ = json.NewDecoder(resp.Body).Decode(&er)
	if er.Error.Code != "CHAT-006" {
		t.Fatalf("code = %q, want CHAT-006", er.Error.Code)
	}
}

func TestListMessages_LimitNegative_Returns400_CHAT006(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	tok := issueTestToken(t, env, uuid.New())

	resp := doGet(t, env, "/channels/"+uuid.New().String()+"/messages?limit=-1", tok)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	var er errResp
	_ = json.NewDecoder(resp.Body).Decode(&er)
	if er.Error.Code != "CHAT-006" {
		t.Fatalf("code = %q, want CHAT-006", er.Error.Code)
	}
}

func TestListMessages_LimitTooBig_Returns400_CHAT006(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	tok := issueTestToken(t, env, uuid.New())

	resp := doGet(t, env, "/channels/"+uuid.New().String()+"/messages?limit=101", tok)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	var er errResp
	_ = json.NewDecoder(resp.Body).Decode(&er)
	if er.Error.Code != "CHAT-006" {
		t.Fatalf("code = %q, want CHAT-006", er.Error.Code)
	}
}

func TestListMessages_NotMember_Returns403_CHAT004(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	tok := issueTestToken(t, env, uuid.New())
	channel := uuid.New()
	env.messages.withChannel(channel, uuid.New(), "text")
	// membership.allow не вызван — fake вернёт ErrChatAccessDenied.

	resp := doGet(t, env, "/channels/"+channel.String()+"/messages", tok)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
	var er errResp
	_ = json.NewDecoder(resp.Body).Decode(&er)
	if er.Error.Code != "CHAT-004" {
		t.Fatalf("code = %q, want CHAT-004", er.Error.Code)
	}
}

func TestListMessages_UnknownChannel_Returns404_CHAT002(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	tok := issueTestToken(t, env, uuid.New())
	channel := uuid.New()
	env.membership.markChannelUnknown(channel)

	resp := doGet(t, env, "/channels/"+channel.String()+"/messages", tok)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
	var er errResp
	_ = json.NewDecoder(resp.Body).Decode(&er)
	if er.Error.Code != "CHAT-002" {
		t.Fatalf("code = %q, want CHAT-002", er.Error.Code)
	}
}

func TestListMessages_FullPage_ReturnsNextBefore(t *testing.T) {
	t.Parallel()
	env := setupServer(t)
	uid := uuid.New()
	tok := issueTestToken(t, env, uid)
	channel := uuid.New()
	env.messages.withChannel(channel, uuid.New(), "text")
	env.membership.allow(channel, uid)

	now := env.clock.now
	var lastID uuid.UUID
	for i := 0; i < 3; i++ {
		id := uuid.New()
		env.messages.put(mustMessage(t, id, channel, uid, "msg", now.Add(time.Duration(i)*time.Second)))
		lastID = id
	}

	resp := doGet(t, env, "/channels/"+channel.String()+"/messages?limit=3", tok)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body struct {
		Items      []any   `json:"items"`
		NextBefore *string `json:"nextBefore"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Items) != 3 {
		t.Fatalf("items len = %d, want 3", len(body.Items))
	}
	if body.NextBefore == nil {
		t.Fatal("NextBefore == nil при заполненной странице")
	}
	if *body.NextBefore != lastID.String() {
		t.Fatalf("NextBefore = %q, want %q", *body.NextBefore, lastID.String())
	}
}
