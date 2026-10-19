package accountprovision

import (
	"context"
	"fmt"

	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authn/authentication"
	"github.com/FangcunMount/iam/v5/internal/apiserver/maintenance/policywrite"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	"gorm.io/gorm"
)

func reviewerRequest(input Input, hasher authentication.PasswordHasher, org meta.ID, fp string, result *ReviewerScopeReport) policywrite.Request {
	// Credential verification stays in the original maintenance preconditions;
	// only the reviewed public identity and authorization fingerprint are frozen.
	return policywrite.Request{Operation: "maintenance.reviewer_scope.apply", ChangedBy: "user:" + input.ActorID, Input: map[string]any{"user_id": input.UserID, "actor_id": input.ActorID, "request_id": input.RequestID, "org_id": org.String(), "fingerprint": fp}, Preview: func(ctx context.Context, tx *gorm.DB) (bool, error) {
		report, err := reviewerScopeInspect(ctx, tx, input, hasher, org)
		if err != nil {
			return false, err
		}
		if report.Fingerprint != fp {
			return false, fmt.Errorf("authorization facts changed; repeat preflight")
		}
		if len(report.Plan.Changes) == 0 {
			*result = report
			return false, nil
		}
		return true, nil
	}}
}
