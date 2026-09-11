package constraint

import (
	"encoding/json"
	"sort"
	"strings"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam/v5/internal/apiserver/maintenance/legacycondition/attribute"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
)

const Version1 uint32 = 1

type Operator string

const OperatorEQ Operator = "eq"

// Value 带类型的属性值，必须且只能设置一个与 Type 对应的值字段。
type Value struct {
	Type   attribute.Type `json:"type"`
	String *string        `json:"string,omitempty"`
	Int64  *int64         `json:"int64,omitempty"`
	Bool   *bool          `json:"bool,omitempty"`
}

func StringValue(value string) Value {
	return Value{Type: attribute.TypeString, String: &value}
}

func Int64Value(value int64) Value {
	return Value{Type: attribute.TypeInt64, Int64: &value}
}

func BoolValue(value bool) Value {
	return Value{Type: attribute.TypeBool, Bool: &value}
}

func (v Value) Validate() error { return validateValue(v) }

// Predicate 针对一个对象属性的条件，当前只支持等值比较。
type Predicate struct {
	Key      string   `json:"key"`
	Operator Operator `json:"operator"`
	Value    Value    `json:"value"`
}

func Equal(key string, value Value) Predicate {
	return Predicate{Key: key, Operator: OperatorEQ, Value: value}
}

// Set 授权条件集合；所有条件均满足时匹配，空集合表示无条件。
type Set struct {
	Version uint32      `json:"version"` // 条件表达格式版本
	AllOf   []Predicate `json:"all_of"`  // 必须同时满足的条件
}

func Empty() Set {
	return Set{Version: Version1, AllOf: []Predicate{}}
}

func New(predicates ...Predicate) (Set, error) {
	return Set{Version: Version1, AllOf: predicates}.Normalize()
}

func (s Set) Normalize() (Set, error) { return s.normalize(true) }

// Read-only evaluation may borrow scalar values; exported normalized values own them.
func (s Set) normalize(ownValues bool) (Set, error) {
	if s.Version != 0 && s.Version != Version1 {
		return Set{}, perrors.WithCode(code.ErrInvalidArgument, "unsupported constraint set version: %d", s.Version)
	}
	if len(s.AllOf) > 8 {
		return Set{}, perrors.WithCode(code.ErrInvalidArgument, "constraint set supports at most 8 predicates")
	}
	normalized := make([]Predicate, len(s.AllOf))
	copy(normalized, s.AllOf)
	seen := make(map[string]struct{}, len(normalized))
	for index := range normalized {
		predicate, err := normalizePredicate(normalized[index])
		if err != nil {
			return Set{}, err
		}
		if _, exists := seen[predicate.Key]; exists {
			return Set{}, perrors.WithCode(code.ErrInvalidArgument, "duplicate constraint attribute: %s", predicate.Key)
		}
		seen[predicate.Key] = struct{}{}
		if ownValues {
			predicate.Value = predicate.Value.Clone()
		}
		normalized[index] = predicate
	}
	sort.Slice(normalized, func(i, j int) bool { return normalized[i].Key < normalized[j].Key })
	return Set{Version: Version1, AllOf: normalized}, nil
}

func (s Set) IsUnconditional() bool {
	normalized, err := s.normalize(false)
	return err == nil && len(normalized.AllOf) == 0
}

func (s Set) ValidateAgainst(schema attribute.Schema) error {
	normalized, err := s.normalize(false)
	if err != nil {
		return err
	}
	normalizedSchema, err := schema.Normalize()
	if err != nil {
		return err
	}
	for _, predicate := range normalized.AllOf {
		definition, ok := normalizedSchema.Find(predicate.Key)
		if !ok {
			return perrors.WithCode(code.ErrInvalidArgument, "unsupported constraint attribute: %s", predicate.Key)
		}
		if definition.Type != predicate.Value.Type {
			return perrors.WithCode(code.ErrInvalidArgument, "constraint attribute type mismatch: %s", predicate.Key)
		}
		if predicate.Value.Type == attribute.TypeString && len(definition.AllowedStringValues) > 0 {
			allowed := false
			for _, candidate := range definition.AllowedStringValues {
				if predicate.Value.String != nil && candidate == *predicate.Value.String {
					allowed = true
					break
				}
			}
			if !allowed {
				return perrors.WithCode(code.ErrInvalidArgument, "constraint attribute value is not allowed: %s", predicate.Key)
			}
		}
	}
	return nil
}

func (s Set) CanonicalJSON() ([]byte, error) {
	normalized, err := s.normalize(false)
	if err != nil {
		return nil, err
	}
	return json.Marshal(normalized)
}

func ParseJSON(data []byte) (Set, error) {
	if len(strings.TrimSpace(string(data))) == 0 {
		return Empty(), nil
	}
	var set Set
	if err := json.Unmarshal(data, &set); err != nil {
		return Set{}, perrors.WithCode(code.ErrInvalidArgument, "invalid constraint set JSON: %v", err)
	}
	return set.Normalize()
}

func normalizePredicate(predicate Predicate) (Predicate, error) {
	predicate.Key = strings.TrimSpace(predicate.Key)
	if !strings.HasPrefix(predicate.Key, "object.") || len(predicate.Key) == len("object.") {
		return Predicate{}, perrors.WithCode(code.ErrInvalidArgument, "constraint key must use object.<name>: %s", predicate.Key)
	}
	if predicate.Operator != OperatorEQ {
		return Predicate{}, perrors.WithCode(code.ErrInvalidArgument, "unsupported constraint operator: %s", predicate.Operator)
	}
	if err := validateValue(predicate.Value); err != nil {
		return Predicate{}, err
	}
	return predicate, nil
}

func validateValue(value Value) error {
	setValues := 0
	if value.String != nil {
		setValues++
	}
	if value.Int64 != nil {
		setValues++
	}
	if value.Bool != nil {
		setValues++
	}
	if setValues != 1 {
		return perrors.WithCode(code.ErrInvalidArgument, "constraint value must contain exactly one typed value")
	}
	switch value.Type {
	case attribute.TypeString:
		if value.String == nil {
			return perrors.WithCode(code.ErrInvalidArgument, "constraint string value is required")
		}
	case attribute.TypeInt64:
		if value.Int64 == nil {
			return perrors.WithCode(code.ErrInvalidArgument, "constraint int64 value is required")
		}
	case attribute.TypeBool:
		if value.Bool == nil {
			return perrors.WithCode(code.ErrInvalidArgument, "constraint bool value is required")
		}
	default:
		return perrors.WithCode(code.ErrInvalidArgument, "unsupported constraint value type: %s", value.Type)
	}
	return nil
}

func (v Value) Clone() Value {
	out := v
	if v.String != nil {
		value := *v.String
		out.String = &value
	}
	if v.Int64 != nil {
		value := *v.Int64
		out.Int64 = &value
	}
	if v.Bool != nil {
		value := *v.Bool
		out.Bool = &value
	}
	return out
}
func (s Set) Clone() Set {
	out := s
	if s.AllOf != nil {
		out.AllOf = make([]Predicate, len(s.AllOf))
		copy(out.AllOf, s.AllOf)
	}
	for i := range out.AllOf {
		out.AllOf[i].Value = out.AllOf[i].Value.Clone()
	}
	return out
}
