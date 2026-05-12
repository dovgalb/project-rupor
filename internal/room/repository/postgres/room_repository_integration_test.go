//go:build integration

package postgres_test

import (
	"testing"
)

func TestRoomRepositoryIntegration_SaveWithOwner_Atomic(t *testing.T) {
	t.Skip("integration: requires Postgres, see issue 1.5")
	_ = dbConn(t)
}

func TestRoomRepositoryIntegration_Delete_Cascades(t *testing.T) {
	t.Skip("integration: requires Postgres, see issue 1.5")
	_ = dbConn(t)
}

func TestRoomRepositoryIntegration_ListByMember_OrderedDesc(t *testing.T) {
	t.Skip("integration: requires Postgres, see issue 1.5")
	_ = dbConn(t)
}
