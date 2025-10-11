package assembler

import (
	"gorm.io/gorm"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/interface/restful/handler"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
	"github.com/fangcun-mount/iam-contracts/pkg/errors"
)

// UserModule 用户模块
// 负责组装用户相关的所有组件
type UserModule struct {
	// handler 层
	UserHandler *handler.UserHandler
}

// NewUserModule 创建用户模块
func NewUserModule() *UserModule {
	return &UserModule{}
}

// Initialize 初始化模块
func (m *UserModule) Initialize(params ...interface{}) error {
	db := params[0].(*gorm.DB)
	if db == nil {
		return errors.WithCode(code.ErrModuleInitializationFailed, "database connection is nil")
	}

	// 初始化 handler 层
	m.UserHandler = handler.NewUserHandler()

	return nil
}

// Cleanup 清理模块资源
func (m *UserModule) Cleanup() error {
	// 如果有需要清理的资源，在这里进行清理
	// 比如关闭数据库连接、释放缓存等
	return nil
}

// CheckHealth 检查模块健康状态
func (m *UserModule) CheckHealth() error {
	return nil
}

// ModuleInfo 返回模块信息
func (m *UserModule) ModuleInfo() ModuleInfo {
	return ModuleInfo{
		Name:        "user",
		Version:     "1.0.0",
		Description: "用户管理模块",
	}
}
