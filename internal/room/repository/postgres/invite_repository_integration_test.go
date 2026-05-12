//go:build integration

package postgres_test

import (
	"testing"
)

func TestInviteRepositoryIntegration_RegenerateActive_RevokesAndInserts(t *testing.T) {
	t.Skip("integration: requires Postgres, see issue 1.5")
	_ = dbConn(t)
}

func TestInviteRepositoryIntegration_OnlyOneActivePerRoom(t *testing.T) {
	t.Skip("integration: requires Postgres, see issue 1.5")
	_ = dbConn(t)
}

func TestInviteRepositoryIntegration_FindActiveByCode_IgnoresRevoked(t *testing.T) {
	t.Skip("integration: requires Postgres, see issue 1.5")
	_ = dbConn(t)
}
