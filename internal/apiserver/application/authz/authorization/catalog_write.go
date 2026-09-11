package authorization

import (
	"context"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	domain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/authorization"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/subject"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
)

// RequireCatalogWrite 检查当前操作人是否具有资源目录写入权限。
func (s *DecisionService) RequireCatalogWrite(ctx context.Context, actor subject.Ref, action string) error {
	request, err := domain.NewRequest(actor, ResourceResources, action)
	if err != nil {
		return err
	}
	decision, err := s.Check(ctx, request)
	if err != nil {
		return err
	}
	if !decision.Allowed {
		return perrors.WithCode(code.ErrPermissionDenied, "resource catalog write permission required")
	}
	return nil
}
