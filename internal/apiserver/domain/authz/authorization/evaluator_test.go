package authorization_test

import (
	"testing"
	"time"

	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/authorization"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/permissiongrant"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/resource"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/role"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/subject"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	"github.com/stretchr/testify/require"
)

const documentResource = "example:catalog:collection:documents"

func TestEvaluatorAllowsMatchingCandidateAndPreservesEvidence(t *testing.T) {
	t.Parallel()

	request, catalogResource, roles, grantsByRole := evaluationFixture(t, "active")
	at := time.Date(2026, time.September, 1, 8, 0, 0, 0, time.UTC)

	decision, err := authorization.NewEvaluator().Evaluate(request, authorization.EvaluationContext{
		EffectiveRoles: roles, RoleNames: map[meta.ID]role.Name{12: "example:evaluator"}, GrantsByRole: grantsByRole, Resource: catalogResource, PolicyVersion: 9,
	}, at)

	require.NoError(t, err)
	require.True(t, decision.Allowed)
	require.Equal(t, meta.FromUint64(102), decision.MatchedGrantID)
	require.Equal(t, "example:evaluator", decision.MatchedRole)
	require.EqualValues(t, 9, decision.PolicyVersion)
	require.Equal(t, at, decision.EvaluatedAt)
}

func TestEvaluatorUsesCandidateOrderForMatchedEvidence(t *testing.T) {
	t.Parallel()

	request, catalogResource, roles, grantsByRole := evaluationFixture(t, "active")
	second := *grantsByRole[roles[0]][0]
	second.ID = meta.FromUint64(103)
	grantsByRole[roles[0]] = append(grantsByRole[roles[0]], &second)

	decision, err := authorization.NewEvaluator().Evaluate(request, authorization.EvaluationContext{
		EffectiveRoles: roles, RoleNames: map[meta.ID]role.Name{12: "example:evaluator"}, GrantsByRole: grantsByRole, Resource: catalogResource, PolicyVersion: 9,
	}, time.Time{})

	require.NoError(t, err)
	require.Equal(t, meta.FromUint64(102), decision.MatchedGrantID)
	require.Equal(t, "example:evaluator", decision.MatchedRole)
}

func evaluationFixture(
	t testing.TB,
	statusValue string,
) (authorization.Request, *resource.Resource, []meta.ID, map[meta.ID][]*permissiongrant.Grant) {
	t.Helper()

	catalogResource, err := resource.NewResource(
		documentResource,
		[]string{"retry"},
		resource.WithID(resource.NewResourceID(20)),
		resource.WithDisplayName("Assessments"),
	)
	require.NoError(t, err)
	grant, err := permissiongrant.New(
		meta.FromUint64(12), catalogResource.ID,
		catalogResource.KeyString(), "retry", "bootstrap",
	)
	require.NoError(t, err)
	grant.ID = meta.FromUint64(102)

	sub, err := subject.NewUserRef(meta.FromUint64(2))
	require.NoError(t, err)
	request, err := authorization.NewRequest(sub, documentResource, "retry")
	require.NoError(t, err)

	return request, &catalogResource, []meta.ID{12}, map[meta.ID][]*permissiongrant.Grant{
		12: {&grant},
	}
}
