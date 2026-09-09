package permissiongrant_test

import (
	"testing"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/constraint"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/permissiongrant"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/resource"
	authzfixture "github.com/FangcunMount/iam/v5/internal/apiserver/testfixtures/authzschema"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	"github.com/stretchr/testify/require"
)

func TestManagedGrantUsesStableCanonicalKey(t *testing.T) {
	constraints, err := constraint.New(constraint.Equal(authzfixture.AttributeKey, constraint.StringValue("active")))
	require.NoError(t, err)
	grant, err := permissiongrant.New(
		meta.FromUint64(10),

		resource.NewResourceID(20),
		"example:catalog:collection:documents",
		"retry",
		constraints,
		"operator-1",
	)
	require.NoError(t, err)
	require.True(t, grant.IsConditional())
	require.Len(t, grant.GrantKey, 64)

	second, err := permissiongrant.New(
		meta.FromUint64(10), resource.NewResourceID(20),
		"example:catalog:collection:documents", "retry", constraints, "operator-2",
	)
	require.NoError(t, err)
	require.Equal(t, grant.GrantKey, second.GrantKey)
}

func TestManagedGrantRequiresCatalogResourceAndConcreteAction(t *testing.T) {
	_, err := permissiongrant.New(
		meta.FromUint64(10), resource.ResourceID{},
		"example:*:*:*", "retry", constraint.Empty(), "operator-1",
	)
	require.True(t, perrors.IsCode(err, code.ErrInvalidArgument))

	_, err = permissiongrant.New(
		meta.FromUint64(10), resource.NewResourceID(20),
		"example:catalog:collection:documents", "*", constraint.Empty(), "operator-1",
	)
	require.True(t, perrors.IsCode(err, code.ErrInvalidArgument))
}

func TestSystemWildcardGrantMustBeUnconditional(t *testing.T) {
	conditional, err := constraint.New(constraint.Equal(authzfixture.AttributeKey, constraint.StringValue("active")))
	require.NoError(t, err)
	_, err = permissiongrant.NewSystem(
		meta.FromUint64(10), resource.ResourceID{},
		"*:*:*:*", resource.WildcardAction.String(), conditional, "bootstrap",
	)
	require.True(t, perrors.IsCode(err, code.ErrInvalidArgument))

	grant, err := permissiongrant.NewSystem(
		meta.FromUint64(10), resource.ResourceID{},
		"*:*:*:*", resource.WildcardAction.String(), constraint.Empty(), "bootstrap",
	)
	require.NoError(t, err)
	action, err := resource.NewAction("retry")
	require.NoError(t, err)
	require.True(t, grant.MatchesAction(action))
}

func TestManagedGrantValidatesAgainstResourceSchema(t *testing.T) {
	constraints, err := constraint.New(constraint.Equal(authzfixture.AttributeKey, constraint.StringValue("active")))
	require.NoError(t, err)
	grant, err := permissiongrant.New(
		meta.FromUint64(10), resource.NewResourceID(20),
		"example:catalog:collection:documents", "retry", constraints, "operator-1",
	)
	require.NoError(t, err)
	catalogResource, err := resource.NewResource(
		"example:catalog:collection:documents",
		[]string{"retry", "batch_evaluate"},
		resource.WithID(resource.NewResourceID(20)),
		resource.WithDisplayName("Assessments"),
		resource.WithAttributeSchema(authzfixture.Schema()),
	)
	require.NoError(t, err)
	require.NoError(t, grant.ValidateAgainst(catalogResource))

	invalid, err := permissiongrant.New(
		meta.FromUint64(10), resource.NewResourceID(20),
		"example:catalog:collection:documents", "retry",
		constraint.Empty(), "operator-1",
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

func TestConditionalGrantCannotAuthorizeCollectionOrBatchActions(t *testing.T) {
	constraints, err := constraint.New(constraint.Equal(authzfixture.AttributeKey, constraint.StringValue("active")))
	require.NoError(t, err)
	for _, action := range []string{"list", "search", "batch_evaluate"} {
		_, err := permissiongrant.New(
			meta.FromUint64(10), resource.NewResourceID(20),
			"example:catalog:collection:documents", action, constraints, "operator-1",
		)
		require.True(t, perrors.IsCode(err, code.ErrInvalidArgument), action)
	}
}

func TestGrantResourceKeyKeepsCatalogAndSystemBoundaries(t *testing.T) {
	catalog, err := resource.NewResource("example:catalog:collection:documents", []string{"read"}, resource.WithID(resource.NewResourceID(20)), resource.WithDisplayName("Documents"))
	require.NoError(t, err)
	grant, err := permissiongrant.New(meta.FromUint64(10), catalog.ID, "example:*:*:*", "read", constraint.Empty(), "operator")
	require.NoError(t, err) // Creation and catalog binding remain separate checks.
	require.Error(t, grant.ValidateAgainst(catalog))
	grant, err = permissiongrant.NewSystem(meta.FromUint64(10), resource.ResourceID{}, "example:*:*:*", "*", constraint.Empty(), "bootstrap")
	require.NoError(t, err)
	require.True(t, grant.CoversResource(catalog.Key))
	require.False(t, grant.CoversResource(resource.Key("other:catalog:collection:documents")))
}
