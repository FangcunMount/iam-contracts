package assembler

import (
	"gorm.io/gorm"

	appacc "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/application/account"
	appuow "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/application/uow"
	mysqlacct "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/infra/mysql/account"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
	"github.com/fangcun-mount/iam-contracts/pkg/errors"
)

// AuthModule 认证模块
// 负责组装认证相关的所有组件
type AuthModule struct {
	// 这里可以添加认证相关的组件
	RegisterService *appacc.RegisterService
	EditorService   *appacc.EditorService
	QueryService    *appacc.QueryService
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

	// 初始化仓储
	accountRepo := mysqlacct.NewAccountRepository(db)
	operationRepo := mysqlacct.NewOperationRepository(db)
	wechatRepo := mysqlacct.NewWeChatRepository(db)

	// 事务 UnitOfWork
	u := appuow.NewUnitOfWork(db)

	// 应用服务
	m.RegisterService = appacc.NewRegisterService(accountRepo, wechatRepo, operationRepo, u)
	m.EditorService = appacc.NewEditorService(wechatRepo, operationRepo, u)
	m.QueryService = appacc.NewQueryService(accountRepo, wechatRepo, operationRepo)

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
