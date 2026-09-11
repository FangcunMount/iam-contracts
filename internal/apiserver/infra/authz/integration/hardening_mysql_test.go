package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/management"

	grantapp "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/permissiongrant"
	resourceapp "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/resource"
	roleapp "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/role"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/resource"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/role"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/subject"
	authzruntime "github.com/FangcunMount/iam/v5/internal/apiserver/infra/authz/runtime"
	grantrepo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/permissiongrant"
	resourcerepo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/resource"
	rolerepo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/role"
	authzuow "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/uow/authz"
	"github.com/FangcunMount/iam/v5/internal/apiserver/testfixtures/authzdb"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func seedRole(t *testing.T, db *gorm.DB, name string) role.Role {
	t.Helper()
	r, err := role.NewRole(name, name)
	require.NoError(t, err)
	require.NoError(t, rolerepo.NewRoleRepository(db).Create(management.WithAuthenticatedService(context.Background(), "admin"), &r))
	return r
}
func concurrent(t *testing.T, a, b func() error) {
	t.Helper()
	start := make(chan struct{})
	results := make(chan error, 2)
	go func() { <-start; results <- a() }()
	go func() { <-start; results <- b() }()
	close(start)
	first, second := <-results, <-results
	require.True(t, (first == nil) != (second == nil), "exactly one conflicting operation must succeed: %v, %v", first, second)
}
func TestMySQLRoleDeletionAndGrantSerialize(t *testing.T) {
	db := authzdb.Open(t, true)
	ctx := management.WithAuthenticatedService(context.Background(), "admin")
	r := seedRole(t, db, "reader")
	res, err := resource.NewResource("example:catalog:collection:documents", []string{"read"}, resource.WithDisplayName("Documents"))
	require.NoError(t, err)
	require.NoError(t, resourcerepo.NewResourceRepository(db).Create(ctx, &res))
	uow := authzuow.NewUnitOfWork(db, nil, authzdb.Stager(t, db))
	grants := grantapp.NewService(uow, grantrepo.NewRepository(db), nil, management.NewGuard(nil))
	roles := roleapp.NewRoleCatalog(uow, nil, management.NewGuard(nil))
	concurrent(t, func() error {
		_, err := grants.Create(ctx, grantapp.CreateCommand{RoleID: r.ID, ResourceID: res.ID, Action: "read", GrantedBy: "seed"})
		return err
	}, func() error {
		return roles.DeleteRole(ctx, roleapp.DeleteRoleCommand{ID: r.ID, ChangedBy: "seed"})
	})
	data, err := authzruntime.NewMySQLSource(db).Load(ctx)
	require.NoError(t, err)
	_, err = authzruntime.BuildSnapshot(data, time.Time{})
	require.NoError(t, err)
}

type platformAdmission struct{}

func (platformAdmission) RequireCatalogWrite(context.Context, subject.Ref, string) error { return nil }
func TestMySQLResourceUpdateAndGrantSerialize(t *testing.T) {
	db := authzdb.Open(t, true)
	ctx := management.WithAuthenticatedService(context.Background(), "admin")
	r := seedRole(t, db, "reader")
	res, err := resource.NewResource("example:catalog:collection:documents", []string{"read", "use"}, resource.WithDisplayName("Documents"))
	require.NoError(t, err)
	require.NoError(t, resourcerepo.NewResourceRepository(db).Create(ctx, &res))
	uow := authzuow.NewUnitOfWork(db, nil, authzdb.Stager(t, db))
	actor, err := subject.NewUserRef(meta.ID(1))
	require.NoError(t, err)
	grants := grantapp.NewService(uow, grantrepo.NewRepository(db), nil, management.NewGuard(nil))
	resources := resourceapp.NewResourceCatalog(uow, nil, platformAdmission{})
	concurrent(t, func() error {
		_, err := grants.Create(ctx, grantapp.CreateCommand{RoleID: r.ID, ResourceID: res.ID, Action: "use", GrantedBy: "seed"})
		return err
	}, func() error {
		_, err := resources.UpdateResource(ctx, resourceapp.UpdateResourceCommand{ID: res.ID, ChangedBy: "seed", Actor: actor, Actions: []string{"read"}})
		return err
	})
	data, err := authzruntime.NewMySQLSource(db).Load(ctx)
	require.NoError(t, err)
	_, err = authzruntime.BuildSnapshot(data, time.Time{})
	require.NoError(t, err)
}
