package policy_test

import (
	"testing"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/permissiongrant"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/policy"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/resource"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	"github.com/stretchr/testify/require"
)

func TestResourceChangePolicyRejectsInvalidCandidate(t *testing.T) {
	grant := assessmentRetryGrant(t)
	candidate := documentResource(t, []string{"read"})

	err := policy.ResourceChangePolicy{}.ValidateDependencies(candidate, []*permissiongrant.Grant{&grant})

	require.Error(t, err)
	require.True(t, perrors.IsCode(err, code.ErrResourceInUse))
}

func TestResourceChangePolicyAcceptsCompatibleCandidate(t *testing.T) {
	grant := assessmentRetryGrant(t)
	candidate := documentResource(t, []string{"retry", "read"})

	require.NoError(t, policy.ResourceChangePolicy{}.ValidateDependencies(candidate, []*permissiongrant.Grant{&grant}))
}

func documentResource(t *testing.T, actions []string) resource.Resource {
	t.Helper()
	value, err := resource.NewResource(
		"example:catalog:collection:documents",
		actions,
		resource.WithID(resource.NewResourceID(101)),
		resource.WithDisplayName("Assessments"),
	)
	require.NoError(t, err)
	return value
}

func assessmentRetryGrant(t *testing.T) permissiongrant.Grant {
	t.Helper()
	grant, err := permissiongrant.New(
		meta.FromUint64(7),

		resource.NewResourceID(101),
		"example:catalog:collection:documents",
		"retry",
		"operator",
	)
	require.NoError(t, err)
	grant.ID = meta.FromUint64(501)
	return grant
}
