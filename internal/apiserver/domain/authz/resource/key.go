package resource

import (
	"regexp"
	"strings"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
)

// keySegmentCount 资源键段数
const keySegmentCount = 4

// resourceSegmentPattern 资源键各段的具体名称格式。
var resourceSegmentPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)

// Key 表达授权资源键或通配资源范围；目录和请求须通过 ValidateTarget 校验。
type Key string

// NewKey 规范化四段资源键，允许各段使用完整通配符 *。
func NewKey(value string) (Key, error) {
	value = strings.TrimSpace(value)
	if _, err := parseFourSegmentResource(value, "resource key"); err != nil {
		return "", err
	}
	return Key(value), nil
}

// ValidateTarget 校验目录或请求的资源键：前三段必须具体，末段允许 *。
// 同时校验完整格式，避免直接类型转换绕过构造规则。
func (k Key) ValidateTarget() error {
	parts, err := parseFourSegmentResource(k.String(), "resource key")
	if err != nil {
		return err
	}
	for i := 0; i < keySegmentCount-1; i++ {
		if parts[i] == "*" {
			return perrors.WithCode(code.ErrInvalidArgument, "resource key wildcard is only allowed in name segment")
		}
	}
	return nil
}

// Covers 判断授予资源键是否覆盖目标资源；按四段逐段匹配，* 覆盖该段任意值。
func (k Key) Covers(candidate Key) bool {
	granted, err := parseFourSegmentResource(k.String(), "resource key")
	if err != nil {
		return false
	}
	target, err := parseFourSegmentResource(candidate.String(), "resource")
	if err != nil {
		return false
	}
	for i := range granted {
		if granted[i] != "*" && granted[i] != target[i] {
			return false
		}
	}
	return true
}

// parseFourSegmentResource 解析四段资源。
func parseFourSegmentResource(value, label string) ([]string, error) {
	if value == "" {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "%s is required", label)
	}
	parts := strings.Split(value, ":")
	if len(parts) != keySegmentCount {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "%s must use <app>:<domain>:<type>:<name-or-pattern>", label)
	}
	for index, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			return nil, perrors.WithCode(code.ErrInvalidArgument, "%s contains empty segment", label)
		}
		if trimmed != part {
			return nil, perrors.WithCode(code.ErrInvalidArgument, "%s contains untrimmed segment", label)
		}
		if trimmed != "*" && !resourceSegmentPattern.MatchString(trimmed) {
			return nil, perrors.WithCode(code.ErrInvalidArgument, "%s contains unsupported segment: %s", label, trimmed)
		}
		parts[index] = trimmed
	}
	return parts, nil
}

// String 返回资源键字符串。
func (k Key) String() string {
	return string(k)
}

// App 返回资源所属应用。
func (k Key) App() string {
	return appSegment(string(k))
}

// Domain 返回资源所属业务域。
func (k Key) Domain() string {
	return segment(string(k), 1)
}

// Type 返回资源类型。
func (k Key) Type() string {
	return segment(string(k), 2)
}

// AppNameFromKey 从资源键获取应用名称。
func AppNameFromKey(value string) (string, bool) {
	key, err := NewKey(value)
	if err != nil {
		return "", false
	}
	app := key.App()
	return app, app != "" && app != "*"
}

// appSegment 返回应用段。
func appSegment(value string) string {
	return segment(value, 0)
}

// segment 返回指定索引的段。
func segment(value string, index int) string {
	parts := strings.Split(value, ":")
	if len(parts) != keySegmentCount {
		return ""
	}
	return parts[index]
}
