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
}

// TenantConfig 租户配置
type TenantConfig struct {
	Code        string `yaml:"code"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// UserConfig 用户配置
type UserConfig struct {
	Alias       string `yaml:"alias"`       // 用于引用的别名
	TenantCode  string `yaml:"tenant_code"` // 租户代码
	Username    string `yaml:"username"`
	DisplayName string `yaml:"display_name"`
	Phone       string `yaml:"phone"`
	Email       string `yaml:"email"`
}

// ChildConfig 儿童档案配置
type ChildConfig struct {
	Alias      string `yaml:"alias"`       // 用于引用的别名
	TenantCode string `yaml:"tenant_code"` // 租户代码
	Name       string `yaml:"name"`
	Nickname   string `yaml:"nickname"`
	Gender     string `yaml:"gender"` // male/female
	Birthday   string `yaml:"birthday"`
}

// GuardianshipConfig 监护关系配置
type GuardianshipConfig struct {
	UserAlias     string `yaml:"user_alias"`     // 用户别名
	ChildAlias    string `yaml:"child_alias"`    // 儿童别名
	Relationship  string `yaml:"relationship"`   // father/mother/other
	IsPrimary     bool   `yaml:"is_primary"`     // 是否主监护人
	ContactPhone  string `yaml:"contact_phone"`  // 联系电话
	EmergencyCall bool   `yaml:"emergency_call"` // 紧急联系人
}

// AccountConfig 认证账号配置
type AccountConfig struct {
	Alias       string `yaml:"alias"`       // 用于引用的别名
	UserAlias   string `yaml:"user_alias"`  // 关联的用户别名
	TenantCode  string `yaml:"tenant_code"` // 租户代码
	Username    string `yaml:"username"`
	Password    string `yaml:"password"`
	AccountType string `yaml:"account_type"` // operation/parent/teacher
}

// ResourceConfig 授权资源配置
type ResourceConfig struct {
	Alias       string `yaml:"alias"`       // 用于引用的别名
	TenantCode  string `yaml:"tenant_code"` // 租户代码
	Code        string `yaml:"code"`
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`         // api/menu/button
	ParentAlias string `yaml:"parent_alias"` // 父资源别名
	Path        string `yaml:"path"`
	Method      string `yaml:"method"`
	Description string `yaml:"description"`
}

// AssignmentConfig 角色分配配置
type AssignmentConfig struct {
	AccountAlias    string   `yaml:"account_alias"`    // 账号别名
	TenantCode      string   `yaml:"tenant_code"`      // 租户代码
	Role            string   `yaml:"role"`             // admin/teacher/parent
	ResourceAliases []string `yaml:"resource_aliases"` // 资源别名列表
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
