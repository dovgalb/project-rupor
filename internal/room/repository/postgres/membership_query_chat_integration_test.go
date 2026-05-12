//go:build integration

package postgres_test

import (
	"testing"
)

// Интеграционные тесты MembershipQueryChatAdapter. Запускаются с тегом
// `integration` против реального Postgres (TEST_DATABASE_URL). Реализация
// в рамках issue 1.5.

func TestMembershipQueryChatAdapter_Member_AnyMember_ReturnsNil(t *testing.T) {
	t.Skip("integration: requires Postgres, see issue 1.5")
	_ = dbConn(t)
}

func TestMembershipQueryChatAdapter_NotMember_ChannelExists_ReturnsErrAccessDenied(t *testing.T) {
	t.Skip("integration: requires Postgres, see issue 1.5")
	_ = dbConn(t)
}

func TestMembershipQueryChatAdapter_NotMember_ChannelMissing_ReturnsErrChannelNotFound(t *testing.T) {
	t.Skip("integration: requires Postgres, see issue 1.5")
	_ = dbConn(t)
}
