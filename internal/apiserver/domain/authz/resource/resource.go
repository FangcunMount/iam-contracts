package resource

import (
	"github.com/FangcunMount/component-base/pkg/util/idutil"
)

// Resource 域对象资源目录（聚合根）
// V1：仅域对象类型，格式：<app>:<domain>:<type>:* 例如 scale:form:*
type Resource struct {
	ID          ResourceID
	Key         string   // 资源键，如 scale:form:*
	DisplayName string   // 显示名称
	AppName     string   // 应用名称
	Domain      string   // 业务域
	Type        string   // 对象类型
	Actions     []string // 允许的动作列表
	Description string   // 描述
}

// NewResource 创建新资源
func NewResource(key string, actions []string, opts ...ResourceOption) Resource {
	r := Resource{
		Key:     key,
		Actions: actions,
	}
	for _, opt := range opts {
		opt(&r)
	}
	return r
}

// ResourceOption 资源选项
type ResourceOption func(*Resource)

func WithID(id ResourceID) ResourceOption        { return func(r *Resource) { r.ID = id } }
func WithDisplayName(name string) ResourceOption { return func(r *Resource) { r.DisplayName = name } }
func WithAppName(app string) ResourceOption      { return func(r *Resource) { r.AppName = app } }
func WithDomain(domain string) ResourceOption    { return func(r *Resource) { r.Domain = domain } }
func WithType(typ string) ResourceOption         { return func(r *Resource) { r.Type = typ } }
func WithDescription(desc string) ResourceOption { return func(r *Resource) { r.Description = desc } }

// HasAction 检查资源是否包含指定动作
func (r *Resource) HasAction(action string) bool {
	for _, a := range r.Actions {
		if a == action {
			return true
		}
	}
	return false
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
