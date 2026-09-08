package handler

import (
	roleDomain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/role"
	"github.com/FangcunMount/iam/v5/internal/apiserver/transport/rest/authz/dto"
)

// toRoleResponse 转换为响应对象
func (h *RoleHandler) toRoleResponse(r *roleDomain.Role) dto.RoleResponse {
	return dto.RoleResponse{
		ID:                   r.ID,
		ManagementProtection: string(r.ManagementProtection),
		Name:                 r.NameString(),
		DisplayName:          r.DisplayName,

		Description: r.Description,
	}
}
