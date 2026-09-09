package authorization

import (
	"strings"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/attribute"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/constraint"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/resource"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/subject"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
)

// ObjectContext 本次鉴权的对象标识及对象事实；属性由拥有对象的业务服务可信提供。
type ObjectContext struct {
	// ---- 对象事实 ----
	ObjectID   string                // 业务对象标识；提供属性时必填
	Attributes constraint.Attributes // 可信的带类型对象属性
}

// NewObjectContext 创建对象上下文
func NewObjectContext(objectID string, attributes constraint.Attributes) (ObjectContext, error) {
	objectID = strings.TrimSpace(objectID)
	if len(attributes) > 0 && objectID == "" {
		return ObjectContext{}, perrors.WithCode(code.ErrInvalidArgument, "object id is required when object attributes are supplied")
	}
	copyAttributes := make(constraint.Attributes, len(attributes))
	for key, value := range attributes {
		key = strings.TrimSpace(key)
		if !strings.HasPrefix(key, "object.") || len(key) == len("object.") {
			return ObjectContext{}, perrors.WithCode(code.ErrInvalidArgument, "object attribute key must use object.<name>: %s", key)
		}
		if err := value.Validate(); err != nil {
			return ObjectContext{}, err
		}
		copyAttributes[key] = value.Clone()
	}
	return ObjectContext{ObjectID: objectID, Attributes: copyAttributes}, nil
}

// Request 一次访问的授权判定请求。
type Request struct {
	Subject subject.Ref // 接受本次鉴权的主体引用，不是身份证明

	ResourceKey resource.Key    // 资源键
	Action      resource.Action // 本次请求执行的动作
	Object      ObjectContext   // 对象上下文
}

// NewRequest 创建请求
func NewRequest(sub subject.Ref, resourceKey, action string, object ObjectContext) (Request, error) {
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
	object, err = NewObjectContext(object.ObjectID, object.Attributes)
	if err != nil {
		return Request{}, err
	}
	return Request{Subject: sub, ResourceKey: resourceKeyValue, Action: actionValue, Object: object}, nil
}

// ValidateAttributes 验证属性
func ValidateAttributes(schema attribute.Schema, attributes constraint.Attributes) error {
	normalized, err := schema.Normalize()
	if err != nil {
		return err
	}
	for key, value := range attributes {
		definition, ok := normalized.Find(key)
		if !ok {
			return perrors.WithCode(code.ErrInvalidArgument, "unsupported object attribute: %s", key)
		}
		if err := value.Validate(); err != nil {
			return err
		}
		if definition.Type != value.Type {
			return perrors.WithCode(code.ErrInvalidArgument, "object attribute type mismatch: %s", key)
		}
		if definition.Type == attribute.TypeString && len(definition.AllowedStringValues) > 0 {
			allowed := false
			for _, candidate := range definition.AllowedStringValues {
				if value.String != nil && candidate == *value.String {
					allowed = true
					break
				}
			}
			if !allowed {
				return perrors.WithCode(code.ErrInvalidArgument, "object attribute value is not allowed: %s", key)
			}
		}
	}
	return nil
}
