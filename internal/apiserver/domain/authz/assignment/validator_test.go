package assignment_test

import (
	"context"
	"testing"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authz/assignment"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authz/role"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/testhelpers"
	"github.com/FangcunMount/iam-contracts/internal/pkg/code"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Use shared testhelpers stubs to avoid duplication. Tests run as external package to avoid import cycles.

func TestValidateGrantAndRevokeCommands_Invalids(t *testing.T) {
	v := assignment.NewValidator(&testhelpers.AssignmentRepoStub{}, &testhelpers.RoleRepoStub{})

	// empty grant command
	err := v.ValidateGrantCommand(assignment.GrantCommand{})
	require.Error(t, err)
	assert.True(t, perrors.IsCode(err, code.ErrInvalidArgument))

	// empty revoke command
	err = v.ValidateRevokeCommand(assignment.RevokeCommand{})
	require.Error(t, err)
	assert.True(t, perrors.IsCode(err, code.ErrInvalidArgument))

	// validate list queries
	err = v.ValidateListBySubjectQuery("", "")
	require.Error(t, err)
	assert.True(t, perrors.IsCode(err, code.ErrInvalidArgument))

	err = v.ValidateListByRoleQuery(0, "")
	require.Error(t, err)
	assert.True(t, perrors.IsCode(err, code.ErrInvalidArgument))
}

func TestValidateRevokeByIDParameters_Invalid(t *testing.T) {
	v := assignment.NewValidator(&testhelpers.AssignmentRepoStub{}, &testhelpers.RoleRepoStub{})
	// zero assignment id
	err := v.ValidateRevokeByIDParameters(assignment.NewAssignmentID(0), "")
	require.Error(t, err)
	assert.True(t, perrors.IsCode(err, code.ErrInvalidArgument))
}

func TestCheckRoleExists_NotFoundAndTenantMismatch(t *testing.T) {
	// role not found -> should map to ErrRoleNotFound
	repoNotFound := &testhelpers.RoleRepoStub{R: nil, Err: perrors.WithCode(code.ErrRoleNotFound, "notfound")}
	v1 := assignment.NewValidator(&testhelpers.AssignmentRepoStub{}, repoNotFound)
	err := v1.CheckRoleExists(context.Background(), 100, "t1")
	require.Error(t, err)
	assert.True(t, perrors.IsCode(err, code.ErrRoleNotFound))

	// tenant mismatch
	repo := &testhelpers.RoleRepoStub{R: &role.Role{TenantID: "other"}, Err: nil}
	v2 := assignment.NewValidator(&testhelpers.AssignmentRepoStub{}, repo)
	err = v2.CheckRoleExists(context.Background(), 100, "tenant-a")
	require.Error(t, err)
	assert.True(t, perrors.IsCode(err, code.ErrPermissionDenied))
}

func TestFindAssignmentBySubjectAndRole_FoundAndNotFound(t *testing.T) {
	a1 := &assignment.Assignment{SubjectType: assignment.SubjectTypeUser, SubjectID: "u1", RoleID: 11, TenantID: "t"}
	repo := &testhelpers.AssignmentRepoStub{Assignments: []*assignment.Assignment{a1}, Err: nil}
	v := assignment.NewValidator(repo, &testhelpers.RoleRepoStub{})

	asg, err := v.FindAssignmentBySubjectAndRole(context.Background(), assignment.SubjectTypeUser, "u1", 11, "t")
	require.NoError(t, err)
	require.NotNil(t, asg)
	assert.Equal(t, uint64(11), asg.RoleID)

	// not found
	repoEmpty := &testhelpers.AssignmentRepoStub{Assignments: []*assignment.Assignment{}, Err: nil}
	v2 := assignment.NewValidator(repoEmpty, &testhelpers.RoleRepoStub{})
	asg2, err2 := v2.FindAssignmentBySubjectAndRole(context.Background(), assignment.SubjectTypeUser, "u1", 99, "t")
	require.Error(t, err2)
	assert.Nil(t, asg2)
	assert.True(t, perrors.IsCode(err2, code.ErrAssignmentNotFound))
}
