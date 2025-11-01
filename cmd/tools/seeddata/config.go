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
	Resources     []ResourceConfig     `yaml:"resources"`
	Assignments   []AssignmentConfig   `yaml:"assignments"`
	Policies      []PolicyConfig       `yaml:"policies"`
	JWKS          JWKSConfig           `yaml:"jwks"`
	WechatApps    []WechatAppConfig    `yaml:"wechat_apps"`
	EncryptionKey string               `yaml:"encryption_key"` // IDP 模块加密密钥（32字节）
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

// AssignmentConfig 角色分配配置
type AssignmentConfig struct {
	SubjectType string `yaml:"subject_type"` // user/group
	SubjectID   string `yaml:"subject_id"`   // 主体ID (可以是别名引用)
	RoleID      uint64 `yaml:"role_id"`      // 角色ID
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
