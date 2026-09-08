package roleinheritance

import (
	"strings"
	"time"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
)

type Inheritance struct {
	ID meta.ID

	RoleID          meta.ID
	InheritedRoleID meta.ID
	GrantedBy       string
	GrantedAt       time.Time
	RevokedAt       *time.Time
	Version         uint32
}

func New(roleID, inheritedRoleID meta.ID, grantedBy string) (Inheritance, error) {
	if roleID.IsZero() || inheritedRoleID.IsZero() {
		return Inheritance{}, perrors.WithCode(code.ErrInvalidArgument, "role ids are required")
	}
	if roleID == inheritedRoleID {
		return Inheritance{}, perrors.WithCode(code.ErrInvalidArgument, "role cannot inherit itself")
	}
	grantedBy = strings.TrimSpace(grantedBy)
	if grantedBy == "" {
		return Inheritance{}, perrors.WithCode(code.ErrInvalidArgument, "granted by is required")
	}
	return Inheritance{

		RoleID:          roleID,
		InheritedRoleID: inheritedRoleID,
		GrantedBy:       grantedBy,
		Version:         1,
	}, nil
}

func (i *Inheritance) Revoke(at time.Time) error {
	if i == nil {
		return perrors.WithCode(code.ErrInvalidArgument, "role inheritance is required")
	}
	if i.RevokedAt != nil {
		return nil
	}
	if at.IsZero() {
		at = time.Now()
	}
	i.RevokedAt = &at
	return nil
}

type RestoreOptions struct {
	ID        meta.ID
	GrantedAt time.Time
	RevokedAt *time.Time
	Version   uint32
}

func Restore(roleID, inheritedRoleID meta.ID, grantedBy string, options RestoreOptions) (Inheritance, error) {
	inheritance, err := New(roleID, inheritedRoleID, grantedBy)
	if err != nil {
		return Inheritance{}, err
	}
	inheritance.ID = options.ID
	inheritance.GrantedAt = options.GrantedAt
	inheritance.RevokedAt = options.RevokedAt
	if options.Version > 0 {
		inheritance.Version = options.Version
	}
	return inheritance, nil
}

func (i Inheritance) IsActive() bool { return i.RevokedAt == nil }
