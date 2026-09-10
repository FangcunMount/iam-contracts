package uow

import (
	"context"

	assignmentDomain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/assignment"
	permissionGrantDomain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/permissiongrant"
	policyDomain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/policy"
	resourceDomain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/resource"
	roleDomain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/role"
	"github.com/FangcunMount/iam/v5/pkg/event"
)

type TxRepositories struct {
	Assignments      assignmentDomain.Repository
	Roles            roleDomain.Repository
	Resources        resourceDomain.Repository
	PolicyVersions   policyDomain.Repository
	SubjectResolver  assignmentDomain.SubjectResolver
	PermissionGrants permissionGrantDomain.Repository
	Events           event.Stager
}

type UnitOfWork interface {
	WithinTx(ctx context.Context, fn func(txCtx context.Context, tx TxRepositories) error) error
}
