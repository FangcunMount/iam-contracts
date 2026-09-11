package permissiongrant_test

import (
	"testing"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/permissiongrant"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/resource"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	"github.com/stretchr/testify/require"
)

func TestManagedGrantUsesStableCanonicalKey(t *testing.T) {
	grant, err := permissiongrant.New(
		meta.FromUint64(10),

		resource.NewResourceID(20),
		"example:catalog:collection:documents",
		"retry",
		"operator-1",
	)
	require.NoError(t, err)

	require.Len(t, grant.GrantKey, 64)

	second, err := permissiongrant.New(
		meta.FromUint64(10), resource.NewResourceID(20),
		"example:catalog:collection:documents", "retry", "operator-2",
	)
	require.NoError(t, err)
	require.Equal(t, grant.GrantKey, second.GrantKey)
}

func TestManagedGrantRequiresCatalogResourceAndConcreteAction(t *testing.T) {
	_, err := permissiongrant.New(
		meta.FromUint64(10), resource.ResourceID{},
		"example:*:*:*", "retry", "operator-1",
	)
	require.True(t, perrors.IsCode(err, code.ErrInvalidArgument))

	_, err = permissiongrant.New(
		meta.FromUint64(10), resource.NewResourceID(20),
		"example:catalog:collection:documents", "*", "operator-1",
	)
	require.True(t, perrors.IsCode(err, code.ErrInvalidArgument))
}

func TestManagedGrantValidatesAgainstResourceSchema(t *testing.T) {
	grant, err := permissiongrant.New(
		meta.FromUint64(10), resource.NewResourceID(20),
		"example:catalog:collection:documents", "retry", "operator-1",
	)
	require.NoError(t, err)
	catalogResource, err := resource.NewResource(
		"example:catalog:collection:documents",
		[]string{"retry", "batch_evaluate"},
		resource.WithID(resource.NewResourceID(20)),
		resource.WithDisplayName("Assessments"),
	)
	require.NoError(t, err)
	require.NoError(t, grant.ValidateAgainst(catalogResource))

	invalid, err := permissiongrant.New(
		meta.FromUint64(10), resource.NewResourceID(20),
		"example:catalog:collection:documents", "retry",
		"operator-1",
	)
	require.NoError(t, err)
	otherResource, err := resource.NewResource(
		"example:evaluation:collection:other", []string{"retry"},
		resource.WithID(resource.NewResourceID(20)),
		resource.WithDisplayName("Other"),
	)
	require.NoError(t, err)
	err = invalid.ValidateAgainst(otherResource)
	require.True(t, perrors.IsCode(err, code.ErrInvalidArgument))
}

func TestGrantResourceKeyKeepsCatalogAndSystemBoundaries(t *testing.T) {
	catalog, err := resource.NewResource("example:catalog:collection:documents", []string{"read"}, resource.WithID(resource.NewResourceID(20)), resource.WithDisplayName("Documents"))
	require.NoError(t, err)
	grant, err := permissiongrant.New(meta.FromUint64(10), catalog.ID, "example:*:*:*", "read", "operator")
	require.NoError(t, err) // Creation and catalog binding remain separate checks.
	require.Error(t, grant.ValidateAgainst(catalog))
	grant, err = permissiongrant.NewSystem(meta.FromUint64(10), resource.ResourceID{}, "example:*:*:*", "*", "bootstrap")
	require.NoError(t, err)
	require.True(t, grant.CoversResource(catalog.Key))
	require.False(t, grant.CoversResource(resource.Key("other:catalog:collection:documents")))
}
