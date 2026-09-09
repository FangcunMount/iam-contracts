package resource

import (
	"strings"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/component-base/pkg/util/idutil"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/attribute"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
)

// Resource 授权资源定义（聚合根），声明资源支持的动作和对象属性契约。
// Key 使用 <app>:<domain>:<type>:<name-or-pattern> 格式。
type Resource struct {
	ID ResourceID // 资源实体标识

	// ---- 资源标识与归属 ----
	Key     Key    // 授权资源键
	AppName string // 所属应用，与 Key 的应用段一致
	Domain  string // 所属业务域，与 Key 的业务域段一致
	Type    string // 资源类型，与 Key 的类型段一致

	// ---- 展示信息 ----
	DisplayName string // 显示名称
	Description string // 描述信息

	// ---- 授权契约 ----
	Actions         []Action         // 资源支持的动作
	AttributeSchema attribute.Schema // 对象属性定义，用于校验授权条件与鉴权输入
}

// NewResource 创建资源
func NewResource(key string, actions []string, opts ...ResourceOption) (Resource, error) {
	resourceKey, err := NewKey(key)
	if err != nil {
		return Resource{}, err
	}
	if err := resourceKey.ValidateTarget(); err != nil {
		return Resource{}, err
	}
	normalizedActions, err := NormalizeActions(actions)
	if err != nil {
		return Resource{}, err
	}
	r := Resource{
		Key:     resourceKey,
		Actions: normalizedActions,
	}
	for _, opt := range opts {
		opt(&r)
	}
	r.AppName = strings.TrimSpace(r.AppName)
	r.Domain = strings.TrimSpace(r.Domain)
	r.Type = strings.TrimSpace(r.Type)
	if r.AppName == "" {
		r.AppName = resourceKey.App()
	}
	if r.Domain == "" {
		r.Domain = resourceKey.Domain()
	}
	if r.Type == "" {
		r.Type = resourceKey.Type()
	}
	if r.AppName != resourceKey.App() {
		return Resource{}, perrors.WithCode(code.ErrInvalidArgument, "resource app does not match key")
	}
	if r.Domain != resourceKey.Domain() || r.Type != resourceKey.Type() {
		return Resource{}, perrors.WithCode(code.ErrInvalidArgument, "resource domain/type does not match key")
	}
	attributeSchema, err := r.AttributeSchema.Normalize()
	if err != nil {
		return Resource{}, err
	}
	r.AttributeSchema = attributeSchema
	if err := r.Rename(r.DisplayName); err != nil {
		return Resource{}, err
	}
	return r, nil
}

// RestoreResource 从持久化数据恢复资源，校验资源键、动作与属性契约，
// 但不强制要求创建时的非空显示名称。
func RestoreResource(key string, actions []string, opts ...ResourceOption) (Resource, error) {
	resourceKey, err := NewKey(key)
	if err != nil {
		return Resource{}, err
	}
	if err := resourceKey.ValidateTarget(); err != nil {
		return Resource{}, err
	}
	normalizedActions, err := NormalizeActions(actions)
	if err != nil {
		return Resource{}, err
	}
	r := Resource{
		Key:     resourceKey,
		Actions: normalizedActions,
	}
	for _, opt := range opts {
		opt(&r)
	}
	r.AppName = strings.TrimSpace(r.AppName)
	r.Domain = strings.TrimSpace(r.Domain)
	r.Type = strings.TrimSpace(r.Type)
	r.DisplayName = strings.TrimSpace(r.DisplayName)
	r.Description = strings.TrimSpace(r.Description)
	if r.AppName == "" {
		r.AppName = resourceKey.App()
	}
	if r.Domain == "" {
		r.Domain = resourceKey.Domain()
	}
	if r.Type == "" {
		r.Type = resourceKey.Type()
	}
	if r.AppName != resourceKey.App() {
		return Resource{}, perrors.WithCode(code.ErrInvalidArgument, "resource app does not match key")
	}
	if r.Domain != resourceKey.Domain() || r.Type != resourceKey.Type() {
		return Resource{}, perrors.WithCode(code.ErrInvalidArgument, "resource domain/type does not match key")
	}
	attributeSchema, err := r.AttributeSchema.Normalize()
	if err != nil {
		return Resource{}, err
	}
	r.AttributeSchema = attributeSchema
	return r, nil
}

// normalizeDisplayName 规范化显示名称
func normalizeDisplayName(displayName string) (string, error) {
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		return "", perrors.WithCode(code.ErrInvalidArgument, "显示名称不能为空")
	}
	return displayName, nil
}

// Rename 修改资源显示名称，不改变资源键。
func (r *Resource) Rename(displayName string) error {
	normalized, err := normalizeDisplayName(displayName)
	if err != nil {
		return err
	}
	r.DisplayName = normalized
	return nil
}

// ChangeDescription 更新资源描述
func (r *Resource) ChangeDescription(description string) {
	r.Description = strings.TrimSpace(description)
}

// ResourceOption 资源配置/创建选项
type ResourceOption func(*Resource)

func WithID(id ResourceID) ResourceOption        { return func(r *Resource) { r.ID = id } }
func WithDisplayName(name string) ResourceOption { return func(r *Resource) { r.DisplayName = name } }
func WithAppName(app string) ResourceOption      { return func(r *Resource) { r.AppName = app } }
func WithDomain(domain string) ResourceOption    { return func(r *Resource) { r.Domain = domain } }
func WithType(typ string) ResourceOption         { return func(r *Resource) { r.Type = typ } }
func WithDescription(desc string) ResourceOption { return func(r *Resource) { r.Description = desc } }
func WithAttributeSchema(schema attribute.Schema) ResourceOption {
	return func(r *Resource) { r.AttributeSchema = schema }
}

// KeyString 返回资源键字符串
func (r Resource) KeyString() string {
	return r.Key.String()
}

// ActionStrings 返回资源支持的动作列表
func (r Resource) ActionStrings() []string {
	if len(r.Actions) == 0 {
		return nil
	}
	actions := make([]string, 0, len(r.Actions))
	for _, action := range r.Actions {
		actions = append(actions, action.String())
	}
	return actions
}

// HasAction 检查资源是否包含指定动作
func (r *Resource) HasAction(action string) bool {
	target, err := NewAction(action)
	if err != nil || target.ValidateConcrete() != nil {
		return false
	}
	for _, a := range r.Actions {
		if a == target {
			return true
		}
	}
	return false
}

// ChangeAttributeSchema 更新资源属性模式
func (r *Resource) ChangeAttributeSchema(schema attribute.Schema) error {
	normalized, err := schema.Normalize()
	if err != nil {
		return err
	}
	r.AttributeSchema = normalized
	return nil
}

// ChangeCatalog 更新资源支持的动作列表。
// 本方法不检查已有授权依赖，也不保存或发布变更；这些操作由应用用例协调。
func (r *Resource) ChangeCatalog(actions []string) error {
	normalizedActions, err := NormalizeActions(actions)
	if err != nil {
		return err
	}
	r.Actions = normalizedActions
	return nil
}

// NormalizeActions 规范化动作列表
func NormalizeActions(actions []string) ([]Action, error) {
	seen := make(map[string]struct{}, len(actions))
	normalized := make([]Action, 0, len(actions))
	for _, action := range actions {
		actionValue, err := NewAction(action)
		if err != nil {
			if strings.TrimSpace(action) == "" {
				continue
			}
			return nil, err
		}
		if err := actionValue.ValidateConcrete(); err != nil {
			return nil, err
		}
		actionKey := actionValue.String()
		if _, exists := seen[actionKey]; exists {
			continue
		}
		seen[actionKey] = struct{}{}
		normalized = append(normalized, actionValue)
	}
	if len(normalized) == 0 {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "动作列表不能为空")
	}
	return normalized, nil
}

// ResourceID 资源ID值对象
type ResourceID idutil.ID

func NewResourceID(value uint64) ResourceID {
	return ResourceID(idutil.NewID(value)) // 从 uint64 构造
}

func (id ResourceID) Uint64() uint64 {
	return idutil.ID(id).Uint64()
}

func (id ResourceID) String() string {
	return idutil.ID(id).String()
}

func (r Resource) Clone() Resource {
	out := r
	if r.Actions != nil {
		out.Actions = append([]Action{}, r.Actions...)
	}
	out.AttributeSchema = r.AttributeSchema.Clone()
	return out
}
