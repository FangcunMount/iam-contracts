package assignment_test

import (
	"context"
	"testing"

	"github.com/FangcunMount/iam/v5/internal/apiserver/infra/authz/subjectresolver"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	assignment "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/assignment"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/role"
	"github.com/FangcunMount/iam/v5/internal/apiserver/testhelpers"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateGrantAndRevokeCommands_Invalids(t *testing.T) {
	v := assignment.NewValidator(&testhelpers.RoleRepoStub{}, subjectresolver.NewUserSubjectResolver(testhelpers.NewUserResolverStub()))

	err := v.ValidateGrantParameters("", 0, 0, "")
	require.Error(t, err)
	assert.True(t, perrors.IsCode(err, code.ErrInvalidArgument))

	err = v.ValidateRevokeParameters("", 0, 0)
	require.Error(t, err)
	assert.True(t, perrors.IsCode(err, code.ErrInvalidArgument))

	err = v.ValidateGrantParameters(assignment.SubjectTypeGroup, meta.FromUint64(100), meta.FromUint64(1), "1")
	require.Error(t, err)
	assert.True(t, perrors.IsCode(err, code.ErrInvalidArgument))

	err = v.ValidateRevokeParameters(assignment.SubjectTypeService, meta.FromUint64(1), meta.FromUint64(1))
	require.Error(t, err)
	assert.True(t, perrors.IsCode(err, code.ErrInvalidArgument))
}

func TestCheckRoleExists_NotFoundAndExisting(t *testing.T) {
	repoNotFound := &testhelpers.RoleRepoStub{R: nil, Err: perrors.WithCode(code.ErrRoleNotFound, "notfound")}
	v1 := assignment.NewValidator(repoNotFound, subjectresolver.NewUserSubjectResolver(testhelpers.NewUserResolverStub()))
	err := v1.CheckRoleExists(context.Background(), meta.FromUint64(100))
	require.Error(t, err)
	assert.True(t, perrors.IsCode(err, code.ErrRoleNotFound))

	repo := &testhelpers.RoleRepoStub{R: &role.Role{}, Err: nil}
	v2 := assignment.NewValidator(repo, subjectresolver.NewUserSubjectResolver(testhelpers.NewUserResolverStub()))
	err = v2.CheckRoleExists(context.Background(), meta.FromUint64(100))
	require.NoError(t, err)
}

func TestCheckSubjectExists_OnlySupportsExistingUsers(t *testing.T) {
	userResolver := testhelpers.NewUserResolverStub(meta.FromUint64(123))
	v := assignment.NewValidator(&testhelpers.RoleRepoStub{}, subjectresolver.NewUserSubjectResolver(userResolver))

	err := v.CheckSubjectExists(context.Background(), assignment.SubjectTypeGroup, meta.FromUint64(1))
	require.Error(t, err)
	assert.True(t, perrors.IsCode(err, code.ErrInvalidArgument))

	err = v.CheckSubjectExists(context.Background(), assignment.SubjectTypeUser, meta.FromUint64(999))
	require.Error(t, err)
	assert.True(t, perrors.IsCode(err, code.ErrUserNotFound))

	err = v.CheckSubjectExists(context.Background(), assignment.SubjectTypeUser, meta.FromUint64(123))
	require.NoError(t, err)
}
