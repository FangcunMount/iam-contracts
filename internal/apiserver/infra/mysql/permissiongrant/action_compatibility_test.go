package permissiongrant_test

import (
	"context"
	"testing"

	domain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/permissiongrant"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/resource"
	repo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/permissiongrant"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUnifiedValuesPreserveStoredGrants(t *testing.T) {
	for _, tc := range []struct {
		name, action, pattern, key string
		resourceID                 uint64
	}{
		{"managed", "retry", "example:catalog:collection:documents", "5f65aa3ccd079ce7d4020b3ba003446fa9c25fcd020c124f5e87700283dabe17", 20},
		{"system", "*", "*:*:*:*", "b77f7e845def79b2608b19a7b4484a2337cea61576d0e01d4dd33d7a5862d384", 0},
		{"application", "*", "example:*:*:*", "e72d7f4d3734b095851008a0c45e28af65f836347ba70df84332312be9178738", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Fixed keys capture the pre-refactor canonical wire representation.
			grant, err := domain.Restore(meta.FromUint64(10), resource.NewResourceID(tc.resourceID), tc.pattern, tc.action, "bootstrap", domain.RestoreOptions{GrantKey: tc.key, Version: 1})
			require.NoError(t, err)
			var created domain.Grant
			if tc.resourceID == 0 {
				created, err = domain.NewSystem(grant.RoleID, grant.ResourceID, tc.pattern, tc.action, "bootstrap")
			} else {
				created, err = domain.New(grant.RoleID, grant.ResourceID, tc.pattern, tc.action, "bootstrap")
			}
			require.NoError(t, err)
			require.Equal(t, tc.key, created.GrantKey)
			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
			require.NoError(t, err)
			sqlDB, err := db.DB()
			require.NoError(t, err)
			sqlDB.SetMaxOpenConns(1)
			t.Cleanup(func() { _ = sqlDB.Close() })
			require.NoError(t, db.AutoMigrate(&repo.GrantPO{}))
			repository := repo.NewRepository(db)
			require.NoError(t, repository.Create(context.Background(), &grant))
			restored, err := repository.FindByID(context.Background(), grant.ID)
			require.NoError(t, err)
			require.Equal(t, tc.key, restored.GrantKey)
			require.Equal(t, tc.action, restored.ActionString())
			require.Equal(t, tc.pattern, restored.ResourceKeyString())
			po, err := (repo.Mapper{}).ToPO(restored)
			require.NoError(t, err)
			require.Equal(t, tc.pattern, po.ResourcePattern)
			require.Equal(t, tc.resourceID, restored.ResourceID.Uint64())
			require.True(t, restored.MatchesAction(resource.Action("retry")))
			outcome, err := repository.AtomicRevoke(context.Background(), grant.ID)
			require.NoError(t, err)
			require.Equal(t, domain.RevokeOutcomeRevoked, outcome)
			revoked, err := repository.FindByID(context.Background(), grant.ID)
			require.NoError(t, err)
			require.False(t, revoked.IsActive())
			require.Equal(t, tc.key, revoked.GrantKey)
		})
	}
}
