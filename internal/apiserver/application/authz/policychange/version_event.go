package policychange

import (
	"context"
	"fmt"

	policyDomain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/policy"
	"github.com/FangcunMount/iam/v5/pkg/event"
)

func StagePolicyVersionChanged(ctx context.Context, stager event.Stager, version *policyDomain.PolicyVersion) error {
	if version == nil {
		return nil
	}
	if stager == nil {
		return fmt.Errorf("authz policy version event stager is required")
	}
	return stager.Stage(ctx, policyDomain.NewVersionChangedEvent(version.Version))
}
