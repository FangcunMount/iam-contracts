package role

import (
	"github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/operation"
	domain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/role"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
)

type roleResult struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	DisplayName          string `json:"display_name"`
	Description          string `json:"description"`
	ManagementProtection string `json:"management_protection"`
}

func roleResultCodec(current func() *domain.Role, restore func(domain.Role)) *operation.ResultCodec {
	return operation.ProjectResult("iam.role-result/v1", func() (roleResult, error) {
		value := current()
		if value == nil || value.ID.IsZero() {
			return roleResult{}, operation.ErrInvalid
		}
		return roleResult{ID: value.ID.String(), Name: value.NameString(), DisplayName: value.DisplayName, Description: value.Description, ManagementProtection: string(value.ManagementProtection)}, nil
	}, func(value roleResult) error {
		id, err := meta.ParseID(value.ID)
		if err != nil || id.IsZero() || id.String() != value.ID {
			return operation.ErrInvalid
		}
		role, err := domain.NewRole(value.Name, value.DisplayName, domain.WithID(id), domain.WithDescription(value.Description), domain.WithManagementProtection(domain.ManagementProtection(value.ManagementProtection)))
		if err != nil {
			return err
		}
		restore(role)
		return nil
	})
}
