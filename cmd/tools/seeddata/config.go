package main

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// SeedConfig 定义整个种子数据配置结构
type SeedConfig struct {
	Tenants       []TenantConfig       `yaml:"tenants"`
	Users         []UserConfig         `yaml:"users"`
	Children      []ChildConfig        `yaml:"children"`
	Guardianships []GuardianshipConfig `yaml:"guardianships"`
	Accounts      []AccountConfig      `yaml:"accounts"`
	Roles         []RoleConfig         `yaml:"roles"` // 角色配置
	Resources     []ResourceConfig     `yaml:"resources"`
	Assignments   []AssignmentConfig   `yaml:"assignments"`
	Policies      []PolicyConfig       `yaml:"policies"`
	JWKS          JWKSConfig           `yaml:"jwks"`
	WechatApps    []WechatAppConfig    `yaml:"wechat_apps"`
	EncryptionKey string               `yaml:"encryption_key"`  // IDP 模块加密密钥（32字节）
	CollectionURL string               `yaml:"collection_url"`  // Collection 服务 URL
	QSServiceURL  string               `yaml:"qs_service_url"`  // QS 服务 URL
	IAMServiceURL string               `yaml:"iam_service_url"` // IAM 服务 URL（用于登录获取 token）
}

// TenantConfig 租户配置
type TenantConfig struct {
	Code         string `yaml:"code"`
	Name         string `yaml:"name"`
	ContactName  string `yaml:"contact_name"`
	ContactPhone string `yaml:"contact_phone"`
	ContactEmail string `yaml:"contact_email"`
	Status       string `yaml:"status"`
	MaxUsers     int    `yaml:"max_users"`
	MaxRoles     int    `yaml:"max_roles"`
	Description  string `yaml:"description"`
}

// UserConfig 用户配置
type UserConfig struct {
	Alias  string `yaml:"alias"`   // 用于引用的别名
	Name   string `yaml:"name"`    // 用户姓名
	Phone  string `yaml:"phone"`   // 手机号
	Email  string `yaml:"email"`   // 邮箱
	IDCard string `yaml:"id_card"` // 身份证号
	Status int    `yaml:"status"`  // 用户状态
	// 员工相关配置（可选，用于 QS 服务创建员工）
	OrgID    int      `yaml:"org_id"`    // 机构ID
	Roles    []string `yaml:"roles"`     // 角色列表
	IsActive bool     `yaml:"is_active"` // 是否激活
}

// ChildConfig 儿童档案配置
type ChildConfig struct {
	Alias    string `yaml:"alias"`    // 用于引用的别名
	Name     string `yaml:"name"`     // 儿童姓名
	IDCard   string `yaml:"id_card"`  // 身份证号
	Gender   int    `yaml:"gender"`   // 性别: 1-男, 2-女
	Birthday string `yaml:"birthday"` // 出生日期
	Height   int    `yaml:"height"`   // 身高（十分之一厘米）
	Weight   int    `yaml:"weight"`   // 体重（十分之一公斤）
}

// GuardianshipConfig 监护关系配置
type GuardianshipConfig struct {
	UserAlias  string `yaml:"user_alias"`  // 用户别名
	ChildAlias string `yaml:"child_alias"` // 儿童别名
	Relation   string `yaml:"relation"`    // 监护关系类型
}

// AccountConfig 认证账号配置
type AccountConfig struct {
	Alias       string `yaml:"alias"`        // 用于引用的别名
	UserAlias   string `yaml:"user_alias"`   // 关联的用户别名
	Provider    string `yaml:"provider"`     // operation/wechat/parent/teacher
	ExternalID  string `yaml:"external_id"`  // 外部ID
	Username    string `yaml:"username"`     // 用户名
	Password    string `yaml:"password"`     // 密码
	AppID       string `yaml:"app_id"`       // 应用ID
	Status      int    `yaml:"status"`       // 状态
	AccountType string `yaml:"account_type"` // 账号类型(兼容旧配置)
}

// ResourceConfig 授权资源配置
type ResourceConfig struct {
	Alias       string   `yaml:"alias"`        // 用于引用的别名
	Key         string   `yaml:"key"`          // 资源键
	DisplayName string   `yaml:"display_name"` // 显示名称
	AppName     string   `yaml:"app_name"`     // 应用名称
	Domain      string   `yaml:"domain"`       // 域
	Type        string   `yaml:"type"`         // 资源类型: collection/api/menu/button
	Actions     []string `yaml:"actions"`      // 允许的操作列表
	Description string   `yaml:"description"`  // 描述
}

// RoleConfig 角色配置
type RoleConfig struct {
	Alias       string `yaml:"alias"`        // 用于引用的别名
	Name        string `yaml:"name"`         // 角色名称（租户内唯一）
	DisplayName string `yaml:"display_name"` // 显示名称
	TenantID    string `yaml:"tenant_id"`    // 租户ID
	IsSystem    bool   `yaml:"is_system"`    // 是否系统角色
	Description string `yaml:"description"`  // 描述
}

// AssignmentConfig 角色分配配置
type AssignmentConfig struct {
	SubjectType string `yaml:"subject_type"` // user/group
	SubjectID   string `yaml:"subject_id"`   // 主体ID（支持 @alias 引用用户别名）
	RoleID      uint64 `yaml:"role_id"`      // 角色ID（与 role_alias 二选一）
	RoleAlias   string `yaml:"role_alias"`   // 角色别名（支持 @alias 引用角色）
	TenantID    string `yaml:"tenant_id"`    // 租户ID
	GrantedBy   string `yaml:"granted_by"`   // 授予者
}

// PolicyConfig Casbin策略配置
type PolicyConfig struct {
	Type    string   `yaml:"type"`    // p/g
	Subject string   `yaml:"subject"` // 主体
	Values  []string `yaml:"values"`  // 策略值
}

// JWKSConfig JWKS密钥配置
type JWKSConfig struct {
	KeyID      string `yaml:"key_id"`
	Algorithm  string `yaml:"algorithm"`
	KeySize    int    `yaml:"key_size"`
	ValidYears int    `yaml:"valid_years"`
}

// WechatAppConfig 微信应用配置
type WechatAppConfig struct {
	Alias     string `yaml:"alias"`      // 用于引用的别名
	AppID     string `yaml:"app_id"`     // 微信应用 ID
	Name      string `yaml:"name"`       // 应用名称
	Type      string `yaml:"type"`       // 应用类型：MiniProgram/MP
	Status    string `yaml:"status"`     // 应用状态：Enabled/Disabled/Archived
	AppSecret string `yaml:"app_secret"` // AppSecret（可选，创建时设置）
}

// LoadSeedConfig 从 YAML 文件加载种子数据配置
func LoadSeedConfig(filepath string) (*SeedConfig, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config SeedConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// ParseDate 解析日期字符串
func ParseDate(dateStr string) (time.Time, error) {
	return time.Parse("2006-01-02", dateStr)
}
