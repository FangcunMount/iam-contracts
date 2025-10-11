package assembler

import (
	"gorm.io/gorm"

	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
	"github.com/fangcun-mount/iam-contracts/pkg/errors"
)

// AuthModule 认证模块
// 负责组装认证相关的所有组件
type AuthModule struct {
	// 这里可以添加认证相关的组件
}

// NewAuthModule 创建认证模块
func NewAuthModule() *AuthModule {
	return &AuthModule{}
}

// Initialize 初始化模块
func (m *AuthModule) Initialize(params ...interface{}) error {
	db := params[0].(*gorm.DB)
	if db == nil {
		return errors.WithCode(code.ErrModuleInitializationFailed, "database connection is nil")
	}

	// 这里可以初始化认证相关的组件
	// 目前简化处理，不依赖具体的业务逻辑

	return nil
}

// CheckHealth 检查模块健康状态
func (m *AuthModule) CheckHealth() error {
	return nil
}

// Cleanup 清理模块资源
func (m *AuthModule) Cleanup() error {
	return nil
}

// ModuleInfo 返回模块信息
func (m *AuthModule) ModuleInfo() ModuleInfo {
	return ModuleInfo{
		Name:        "auth",
		Version:     "1.0.0",
		Description: "认证模块",
	}
}
