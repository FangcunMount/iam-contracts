package authorization

import (
	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/resource"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/subject"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
)

// Request 一次访问的授权判定请求。
type Request struct {
	Subject subject.Ref // 接受本次鉴权的主体引用，不是身份证明

	ResourceKey resource.Key    // 资源键
	Action      resource.Action // 本次请求执行的动作
}

// NewRequest 创建请求
func NewRequest(sub subject.Ref, resourceKey, action string) (Request, error) {
	if sub.IsZero() {
		return Request{}, perrors.WithCode(code.ErrInvalidArgument, "subject is required")
	}
	resourceKeyValue, err := resource.NewKey(resourceKey)
	if err != nil {
		return Request{}, err
	}
	if err := resourceKeyValue.ValidateTarget(); err != nil {
		return Request{}, err
	}
	actionValue, err := resource.NewAction(action)
	if err != nil {
		return Request{}, err
	}
	if err := actionValue.ValidateConcrete(); err != nil {
		return Request{}, err
	}
	return Request{Subject: sub, ResourceKey: resourceKeyValue, Action: actionValue}, nil
}
