package dto

import (
	"github.com/FangcunMount/iam/v5/internal/pkg/authzcompat"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
)

type CreatePermissionGrantRequest struct {
	RoleID        meta.ID                 `json:"role_id" binding:"required" swaggertype:"string"`
	ResourceID    meta.ID                 `json:"resource_id" binding:"required" swaggertype:"string"`
	Action        string                  `json:"action" binding:"required"`
	ConstraintSet authzcompat.Constraints `json:"constraint_set"`
}

type RevokePermissionGrantRequest struct {
	Reason string `json:"reason"`
}

type PermissionGrantResponse struct {
	ID meta.ID `json:"id" swaggertype:"string"`

	RoleID          meta.ID                 `json:"role_id" swaggertype:"string"`
	ResourceID      meta.ID                 `json:"resource_id" swaggertype:"string"`
	ResourcePattern string                  `json:"resource_pattern"`
	Action          string                  `json:"action"`
	ConstraintSet   authzcompat.Constraints `json:"constraint_set"`
	GrantKey        string                  `json:"grant_key"`
	GrantedBy       string                  `json:"granted_by"`
	Active          bool                    `json:"active"`
}
