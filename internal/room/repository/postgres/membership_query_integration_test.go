//go:build integration

package postgres_test

import (
	"testing"
)

func TestMembershipQueryAdapterIntegration_RequireAdminOrOwner(t *testing.T) {
	t.Skip("integration: requires Postgres, see issue 1.5")
	_ = dbConn(t)
}
