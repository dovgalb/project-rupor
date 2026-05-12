//go:build integration

package postgres_test

import (
	"testing"
)

func TestChannelRepositoryIntegration_Save_Persists(t *testing.T) {
	t.Skip("integration: requires Postgres, see issue 1.5")
	_ = dbConn(t)
}

func TestChannelRepositoryIntegration_UniqueNamePerRoom(t *testing.T) {
	t.Skip("integration: requires Postgres, see issue 1.5")
	_ = dbConn(t)
}

func TestChannelRepositoryIntegration_DeleteInRoom_NotFound_ReturnsErr(t *testing.T) {
	t.Skip("integration: requires Postgres, see issue 1.5")
	_ = dbConn(t)
}

func TestChannelRepositoryIntegration_DeleteInRoom_WrongRoom_ReturnsErr(t *testing.T) {
	t.Skip("integration: requires Postgres, see issue 1.5")
	_ = dbConn(t)
}

func TestChannelRepositoryIntegration_RoomDelete_CascadesChannels(t *testing.T) {
	t.Skip("integration: requires Postgres, see issue 1.5")
	_ = dbConn(t)
}
