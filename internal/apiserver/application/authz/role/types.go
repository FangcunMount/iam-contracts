package role

import (
	"context"
	"strings"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	roleDomain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/role"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
)

type Catalog interface {
	CreateRole(ctx context.Context, cmd CreateRoleCommand) (*roleDomain.Role, error)
	UpdateRole(ctx context.Context, cmd UpdateRoleCommand) (*roleDomain.Role, error)
	DeleteRole(ctx context.Context, cmd DeleteRoleCommand) error
}

type Directory interface {
	GetRoleByID(ctx context.Context, roleID meta.ID) (*roleDomain.Role, error)
	GetRoleByName(ctx context.Context, name string) (*roleDomain.Role, error)
	ListRoles(ctx context.Context, query ListRolesQuery) (*ListRolesResult, error)
	ListAllRoles(ctx context.Context) ([]*roleDomain.Role, error)
}

type CreateRoleCommand struct {
	ManagementProtection roleDomain.ManagementProtection
	Name                 roleDomain.Name
	DisplayName          string

	Description string
	ChangedBy   string
}

func NewCreateRoleCommand(name, displayName, description string) (CreateRoleCommand, error) {
	roleName, err := roleDomain.NewName(name)
	if err != nil {
		return CreateRoleCommand{}, err
	}
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		return CreateRoleCommand{}, perrors.WithCode(code.ErrInvalidArgument, "显示名称不能为空")
	}
	return CreateRoleCommand{
		Name:                 roleName,
		ManagementProtection: roleDomain.ManagementStandard,
		DisplayName:          displayName,

		Description: description,
	}, nil
}

func (cmd CreateRoleCommand) NameString() string {
	return cmd.Name.String()
}

type UpdateRoleCommand struct {
	ID meta.ID

	ChangedBy   string
	DisplayName *string
	Description *string
}

type DeleteRoleCommand struct {
	ID meta.ID

	ChangedBy string
}

func NewUpdateRoleCommand(id meta.ID, displayName, description *string) (UpdateRoleCommand, error) {
	if id.IsZero() {
		return UpdateRoleCommand{}, perrors.WithCode(code.ErrInvalidArgument, "角色ID不能为空")
	}
	var displayNameValue *string
	if displayName != nil {
		trimmed := strings.TrimSpace(*displayName)
		if trimmed == "" {
			return UpdateRoleCommand{}, perrors.WithCode(code.ErrInvalidArgument, "显示名称不能为空")
		}
		displayNameValue = &trimmed
	}
	var descriptionValue *string
	if description != nil {
		value := *description
		descriptionValue = &value
	}
	return UpdateRoleCommand{
		ID:          id,
		DisplayName: displayNameValue,
		Description: descriptionValue,
	}, nil
}

type ListRolesQuery struct {
	Offset int
	Limit  int
}

func NewListRolesQuery(offset, limit int) (ListRolesQuery, error) {
	return ListRolesQuery{

		Offset: offset,
		Limit:  limit,
	}, nil
}

type ListRolesResult struct {
	Roles []*roleDomain.Role
	Total int64
}
