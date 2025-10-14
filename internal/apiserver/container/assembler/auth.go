package assembler

import (
	"gorm.io/gorm"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/application/account"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/application/adapter"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/application/uow"
	mysqlacct "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/infra/mysql/account"
	authhandler "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authn/interface/restful/handler"
	mysqluser "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/infra/mysql/user"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
	"github.com/fangcun-mount/iam-contracts/pkg/errors"
)

// AuthModule 认证模块
// 负责组装认证相关的所有组件
type AuthModule struct {
	// 这里可以添加认证相关的组件
	RegisterService *account.RegisterService
	EditorService   *account.EditorService
	QueryService    *account.QueryService
	StatusService   *account.StatusService
	AccountHandler  *authhandler.AccountHandler
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

	// 初始化用户仓储(用于防腐层)
	userRepo := mysqluser.NewRepository(db)

	// 创建用户适配器(防腐层)
	userAdapter := adapter.NewUserAdapter(userRepo)

	// 事务 UnitOfWork
	unitOfWork := uow.NewUnitOfWork(db)

	// 应用服务 - 注意注入 UserAdapter
	m.RegisterService = account.NewRegisterService(accountRepo, wechatRepo, operationRepo, unitOfWork, userAdapter)
	m.EditorService = account.NewEditorService(wechatRepo, operationRepo, unitOfWork)
	m.QueryService = account.NewQueryService(accountRepo, wechatRepo, operationRepo)
	m.StatusService = account.NewStatusService(accountRepo)

	m.AccountHandler = authhandler.NewAccountHandler(
		m.RegisterService,
		m.EditorService,
		m.StatusService,
		m.QueryService,
	)

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
