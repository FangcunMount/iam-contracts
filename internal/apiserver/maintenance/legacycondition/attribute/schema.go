package attribute

import (
	"sort"
	"strings"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
)

// Type 属性类型
type Type string

const (
	TypeString Type = "string" // 字符串
	TypeInt64  Type = "int64"  // 整数
	TypeBool   Type = "bool"   // 布尔值
)

// Definition 属性定义
type Definition struct {
	Key                 string   `json:"key"`                             // 对象属性键，使用 object.<name>
	Type                Type     `json:"type"`                            // 属性值类型
	AllowedStringValues []string `json:"allowed_string_values,omitempty"` // 字符串允许值；空列表表示不限制枚举值
}

// Schema 对象属性定义契约，声明鉴权属性的类型及允许值。
type Schema struct {
	Version    uint32       `json:"version"`    // 属性定义格式版本
	Attributes []Definition `json:"attributes"` // 属性定义列表
}

// EmptySchema 空属性模式
func EmptySchema() Schema {
	return Schema{Version: 1, Attributes: []Definition{}}
}

// NewSchema 创建属性模式
func NewSchema(definitions []Definition) (Schema, error) {
	if len(definitions) > 32 {
		return Schema{}, perrors.WithCode(code.ErrInvalidArgument, "attribute schema supports at most 32 attributes")
	}
	normalized := make([]Definition, len(definitions))
	copy(normalized, definitions)
	seen := make(map[string]struct{}, len(normalized))
	for index := range normalized {
		definition, err := normalizeDefinition(normalized[index])
		if err != nil {
			return Schema{}, err
		}
		if _, exists := seen[definition.Key]; exists {
			return Schema{}, perrors.WithCode(code.ErrInvalidArgument, "duplicate attribute definition: %s", definition.Key)
		}
		seen[definition.Key] = struct{}{}
		normalized[index] = definition
	}
	sort.Slice(normalized, func(i, j int) bool { return normalized[i].Key < normalized[j].Key })
	return Schema{Version: 1, Attributes: normalized}, nil
}

// Normalize 规范化属性模式
func (s Schema) Normalize() (Schema, error) {
	if s.Version != 0 && s.Version != 1 {
		return Schema{}, perrors.WithCode(code.ErrInvalidArgument, "unsupported attribute schema version: %d", s.Version)
	}
	return NewSchema(s.Attributes)
}

// Find 查找属性定义
func (s Schema) Find(key string) (Definition, bool) {
	key = strings.TrimSpace(key)
	for _, definition := range s.Attributes {
		if definition.Key == key {
			return definition, true
		}
	}
	return Definition{}, false
}

// normalizeDefinition 规范化属性定义
func normalizeDefinition(definition Definition) (Definition, error) {
	definition.Key = strings.TrimSpace(definition.Key)
	if !strings.HasPrefix(definition.Key, "object.") || len(definition.Key) == len("object.") {
		return Definition{}, perrors.WithCode(code.ErrInvalidArgument, "attribute key must use object.<name>: %s", definition.Key)
	}
	switch definition.Type {
	case TypeString:
	case TypeInt64, TypeBool:
		if len(definition.AllowedStringValues) > 0 {
			return Definition{}, perrors.WithCode(code.ErrInvalidArgument, "allowed string values require string attribute: %s", definition.Key)
		}
	default:
		return Definition{}, perrors.WithCode(code.ErrInvalidArgument, "unsupported attribute type: %s", definition.Type)
	}
	seen := make(map[string]struct{}, len(definition.AllowedStringValues))
	values := make([]string, 0, len(definition.AllowedStringValues))
	for _, value := range definition.AllowedStringValues {
		value = strings.TrimSpace(value)
		if value == "" {
			return Definition{}, perrors.WithCode(code.ErrInvalidArgument, "allowed attribute value cannot be empty: %s", definition.Key)
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	sort.Strings(values)
	definition.AllowedStringValues = values
	return definition, nil
}

// Clone 深复制属性定义及允许值列表，返回独立的属性契约。
func (s Schema) Clone() Schema {
	out := s
	if s.Attributes != nil {
		out.Attributes = make([]Definition, len(s.Attributes))
		copy(out.Attributes, s.Attributes)
	}
	for i := range out.Attributes {
		if s.Attributes[i].AllowedStringValues != nil {
			out.Attributes[i].AllowedStringValues = append([]string{}, s.Attributes[i].AllowedStringValues...)
		}
	}
	return out
}
