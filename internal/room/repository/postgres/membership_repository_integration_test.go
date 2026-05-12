//go:build integration

package postgres_test

import (
	"testing"
)

func TestMembershipRepositoryIntegration_OneOwnerPartialUnique(t *testing.T) {
	t.Skip("integration: requires Postgres, see issue 1.5")
	_ = dbConn(t)
}

func TestMembershipRepositoryIntegration_AddDuplicate_ReturnsErrAlreadyMember(t *testing.T) {
	t.Skip("integration: requires Postgres, see issue 1.5")
	_ = dbConn(t)
}
