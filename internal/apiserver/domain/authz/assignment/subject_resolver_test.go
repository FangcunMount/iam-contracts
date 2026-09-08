package assignment_test

import (
	"context"
	"testing"

	"github.com/FangcunMount/iam/v5/internal/apiserver/infra/authz/subjectresolver"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/assignment"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/subject"
	"github.com/FangcunMount/iam/v5/internal/apiserver/testhelpers"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	"github.com/stretchr/testify/require"
)

func TestSubjectResolverRegistryResolvesUsersAndRejectsUnsupportedSubjects(t *testing.T) {
	t.Parallel()

	userResolver := testhelpers.NewUserResolverStub(meta.FromUint64(123))
	registry := assignment.NewSubjectResolverRegistry(subjectresolver.NewUserSubjectResolver(userResolver))

	userRef, err := subject.NewUserRef(meta.FromUint64(123))
	require.NoError(t, err)
	require.NoError(t, registry.Resolve(context.Background(), userRef))

	groupRef, err := subject.NewRef(subject.TypeGroup, meta.FromUint64(99))
	require.NoError(t, err)
	err = registry.Resolve(context.Background(), groupRef)
	require.True(t, perrors.IsCode(err, code.ErrInvalidArgument))
	var unsupported assignment.UnsupportedSubjectTypeError
	require.ErrorAs(t, err, &unsupported)
}

func TestUserSubjectResolverReportsMissingUsers(t *testing.T) {
	t.Parallel()

	registry := assignment.NewSubjectResolverRegistry(subjectresolver.NewUserSubjectResolver(testhelpers.NewUserResolverStub()))
	userRef, err := subject.NewUserRef(meta.FromUint64(404))
	require.NoError(t, err)

	err = registry.Resolve(context.Background(), userRef)
	require.True(t, perrors.IsCode(err, code.ErrUserNotFound))
}
