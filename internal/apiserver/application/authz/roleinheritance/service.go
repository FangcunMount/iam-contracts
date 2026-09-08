package roleinheritance

import (
	"context"
	"strings"

	"github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/management"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	policychange "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/policychange"
	authzuow "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/uow"
	domain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/roleinheritance"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
)

type CreateCommand struct {
	RoleID          meta.ID
	InheritedRoleID meta.ID
	GrantedBy       string
}

type RevokeCommand struct {
	ID        meta.ID
	RevokedBy string
	Reason    string
}

type Service struct {
	guard    management.Guard
	uow      authzuow.UnitOfWork
	repo     domain.Repository
	reloader policychange.RuntimePolicyReloader
}

func NewService(uow authzuow.UnitOfWork, repo domain.Repository, reloader policychange.RuntimePolicyReloader, guard management.Guard) *Service {
	return &Service{uow: uow, repo: repo, reloader: reloader, guard: guard}
}

func (s *Service) Create(ctx context.Context, cmd CreateCommand) (*domain.Inheritance, error) {
	if err := s.guard.RequireOperation(ctx, "iam:authz:collection:role_inheritances", "grant"); err != nil {
		return nil, err
	}
	if s == nil || s.uow == nil {
		return nil, perrors.WithCode(code.ErrInternalServerError, "role inheritance service is unavailable")
	}
	cmd.GrantedBy = strings.TrimSpace(cmd.GrantedBy)
	inheritance, err := domain.New(cmd.RoleID, cmd.InheritedRoleID, cmd.GrantedBy)
	if err != nil {
		return nil, err
	}
	err = s.uow.WithinTx(ctx, func(txCtx context.Context, tx authzuow.TxRepositories) error {
		if err := s.requireRoles(txCtx, tx, inheritance.RoleID, inheritance.InheritedRoleID); err != nil {
			return err
		}
		if err := tx.RoleInheritances.CreateChecked(txCtx, &inheritance); err != nil {
			return err
		}
		version, err := tx.PolicyVersions.Increment(txCtx, cmd.GrantedBy, "role inheritance created")
		if err != nil {
			return err
		}
		return policychange.StagePolicyVersionChanged(txCtx, tx.Events, version)
	})
	if err != nil {
		return nil, err
	}
	policychange.ReloadRuntimePolicy(ctx, s.reloader, "role_inheritance_created")
	return &inheritance, nil
}

func (s *Service) Revoke(ctx context.Context, cmd RevokeCommand) error {
	if err := s.guard.RequireOperation(ctx, "iam:authz:collection:role_inheritances", "revoke"); err != nil {
		return err
	}
	if s == nil || s.uow == nil {
		return perrors.WithCode(code.ErrInternalServerError, "role inheritance service is unavailable")
	}
	cmd.RevokedBy = strings.TrimSpace(cmd.RevokedBy)
	if cmd.ID.IsZero() || cmd.RevokedBy == "" {
		return perrors.WithCode(code.ErrInvalidArgument, "inheritance id and revoked by are required")
	}
	err := s.uow.WithinTx(ctx, func(txCtx context.Context, tx authzuow.TxRepositories) error {
		inheritance, err := tx.RoleInheritances.FindByID(txCtx, cmd.ID)
		if err != nil {
			return err
		}
		if err := s.requireRoles(txCtx, tx, inheritance.RoleID, inheritance.InheritedRoleID); err != nil {
			return err
		}
		outcome, err := tx.RoleInheritances.AtomicRevoke(txCtx, cmd.ID)
		if err != nil {
			return err
		}
		if outcome != domain.RevokeOutcomeRevoked {
			if outcome == domain.RevokeOutcomeAlreadyRevoked {
				return perrors.WithCode(code.ErrInvalidArgument, "role inheritance is not active")
			}
			return perrors.WithCode(code.ErrInvalidArgument, "role inheritance not found")
		}
		reason := strings.TrimSpace(cmd.Reason)
		if reason == "" {
			reason = "role inheritance revoked"
		}
		version, err := tx.PolicyVersions.Increment(txCtx, cmd.RevokedBy, reason)
		if err != nil {
			return err
		}
		return policychange.StagePolicyVersionChanged(txCtx, tx.Events, version)
	})
	if err != nil {
		return err
	}
	policychange.ReloadRuntimePolicy(ctx, s.reloader, "role_inheritance_revoked")
	return nil
}

func (s *Service) List(ctx context.Context, roleID meta.ID) ([]*domain.Inheritance, error) {
	if s == nil || s.repo == nil {
		return nil, perrors.WithCode(code.ErrInternalServerError, "role inheritance repository is unavailable")
	}

	if s.uow == nil {
		return nil, perrors.WithCode(code.ErrInternalServerError, "角色查询不可用")
	}
	var filtered []*domain.Inheritance
	err := s.uow.WithinTx(ctx, func(txCtx context.Context, tx authzuow.TxRepositories) error {
		if !roleID.IsZero() {
			target, err := tx.Roles.FindByID(txCtx, roleID)
			if err != nil {
				return err
			}
			if err = s.guard.RequireVisible(txCtx, target); err != nil {
				return err
			}
		}
		items, err := tx.RoleInheritances.ListActive(txCtx)
		if err != nil {
			return err
		}
		filtered = make([]*domain.Inheritance, 0, len(items))
		for _, item := range items {
			if item == nil || (!roleID.IsZero() && item.RoleID != roleID) {
				continue
			}
			visible := true
			for _, id := range []meta.ID{item.RoleID, item.InheritedRoleID} {
				target, err := tx.Roles.FindByID(txCtx, id)
				if err != nil {
					return err
				}
				ok, err := s.guard.Visible(txCtx, target)
				if err != nil {
					return err
				}
				visible = visible && ok
			}
			if visible {
				filtered = append(filtered, item)
			}
		}
		return nil
	})
	return filtered, err
}

func (s *Service) requireRoles(txCtx context.Context, tx authzuow.TxRepositories, child, parent meta.ID) error {
	if child > parent {
		child, parent = parent, child
	}
	for _, id := range []meta.ID{child, parent} {
		target, err := tx.Roles.FindByIDForUpdate(txCtx, id)
		if err != nil {
			return err
		}
		if err := s.guard.Require(txCtx, target); err != nil {
			return err
		}
	}
	return nil
}
