package httpx_test

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dovgalb/project-rupor/pkg/httpx"
)

func TestWriteJSONError_BasicEnvelope(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	httpx.WriteJSONError(rec, 401, "AUTH-010", "access token invalid")

	if rec.Code != 401 {
		t.Fatalf("Code = %d, want 401", rec.Code)
	}

	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if body.Error.Code != "AUTH-010" {
		t.Fatalf("Code = %q, want AUTH-010", body.Error.Code)
	}
	if body.Error.Message != "access token invalid" {
		t.Fatalf("Message = %q", body.Error.Message)
	}
}

func TestWriteJSONError_SetsContentTypeAndStatus(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	httpx.WriteJSONError(rec, 500, "INTERNAL", "internal")

	if got := rec.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	if !strings.HasSuffix(rec.Body.String(), "\n") {
		t.Fatalf("body must end with newline, got %q", rec.Body.String())
	}
}
