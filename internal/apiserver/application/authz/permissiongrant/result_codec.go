package permissiongrant

import (
	"github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/operation"
	domain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/permissiongrant"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/resource"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	"time"
)

type grantResult struct {
	ID          string    `json:"id"`
	RoleID      string    `json:"role_id"`
	ResourceID  string    `json:"resource_id"`
	ResourceKey string    `json:"resource_key"`
	Action      string    `json:"action"`
	GrantedBy   string    `json:"granted_by"`
	GrantedAt   time.Time `json:"granted_at"`
	Version     uint32    `json:"version"`
}

func grantResultCodec(current *domain.Grant) *operation.ResultCodec {
	return operation.ProjectResult("iam.grant-result/v1", func() (grantResult, error) {
		if current == nil || current.ID.IsZero() || current.GrantedAt.IsZero() {
			return grantResult{}, operation.ErrInvalid
		}
		return grantResult{ID: current.ID.String(), RoleID: current.RoleID.String(), ResourceID: current.ResourceID.String(), ResourceKey: current.ResourceKeyString(), Action: current.ActionString(), GrantedBy: current.GrantedBy, GrantedAt: current.GrantedAt, Version: current.Version}, nil
	}, func(value grantResult) error {
		id, err := meta.ParseID(value.ID)
		if err != nil || id.IsZero() || id.String() != value.ID {
			return operation.ErrInvalid
		}
		roleID, err := meta.ParseID(value.RoleID)
		if err != nil || roleID.IsZero() || roleID.String() != value.RoleID {
			return operation.ErrInvalid
		}
		resourceID, err := meta.ParseID(value.ResourceID)
		if err != nil || resourceID.IsZero() || resourceID.String() != value.ResourceID {
			return operation.ErrInvalid
		}
		if value.GrantedAt.IsZero() || value.Version == 0 {
			return operation.ErrInvalid
		}
		grant, err := domain.New(roleID, resource.NewResourceID(resourceID.Uint64()), value.ResourceKey, value.Action, value.GrantedBy)
		if err != nil {
			return err
		}
		grant.ID = id
		grant.GrantedAt = value.GrantedAt
		grant.Version = value.Version
		*current = grant
		return nil
	})
}
