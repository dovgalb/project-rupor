//go:build integration

package postgres_test

import (
	"testing"
)

// Интеграционные тесты chat-репозитория. Запускаются с тегом `integration`
// против реального Postgres (TEST_DATABASE_URL). Реализуются в рамках issue 1.5
// (см. internal/channel/repository/postgres/channel_repository_integration_test.go).

func TestMessageRepositoryIntegration_Save_Persists(t *testing.T) {
	t.Skip("integration: requires Postgres, see issue 1.5")
	_ = dbConn(t)
}

func TestMessageRepositoryIntegration_Save_ThenListByChannel_RoundTrip(t *testing.T) {
	t.Skip("integration: requires Postgres, see issue 1.5")
	_ = dbConn(t)
}

func TestMessageRepositoryIntegration_ListByChannel_Empty_ReturnsEmpty(t *testing.T) {
	t.Skip("integration: requires Postgres, see issue 1.5")
	_ = dbConn(t)
}

func TestMessageRepositoryIntegration_ListByChannel_BeforeCursor_Filters(t *testing.T) {
	t.Skip("integration: requires Postgres, see issue 1.5")
	_ = dbConn(t)
}

func TestMessageRepositoryIntegration_ListByChannel_LimitRespected(t *testing.T) {
	t.Skip("integration: requires Postgres, see issue 1.5")
	_ = dbConn(t)
}

func TestMessageRepositoryIntegration_ListByChannel_OrderDescByCreatedAt(t *testing.T) {
	t.Skip("integration: requires Postgres, see issue 1.5")
	_ = dbConn(t)
}

func TestMessageRepositoryIntegration_ChannelOf_Text_ReturnsKindText(t *testing.T) {
	t.Skip("integration: requires Postgres, see issue 1.5")
	_ = dbConn(t)
}

func TestMessageRepositoryIntegration_ChannelOf_Voice_ReturnsKindVoice(t *testing.T) {
	t.Skip("integration: requires Postgres, see issue 1.5")
	_ = dbConn(t)
}

func TestMessageRepositoryIntegration_ChannelOf_Unknown_ReturnsErrChannelNotFound(t *testing.T) {
	t.Skip("integration: requires Postgres, see issue 1.5")
	_ = dbConn(t)
}

func TestMessageRepositoryIntegration_ChannelDelete_CascadesMessages(t *testing.T) {
	t.Skip("integration: requires Postgres, see issue 1.5")
	_ = dbConn(t)
}
