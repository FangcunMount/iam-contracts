package assembler

import (
	"gorm.io/gorm"

	appchild "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/application/child"
	appguard "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/application/guardianship"
	appuow "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/application/uow"
	appuser "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/application/user"
	mysqlchild "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/infra/mysql/child"
	mysqlguard "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/infra/mysql/guardianship"
	mysqluser "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/infra/mysql/user"
	identitygrpc "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/interface/grpc/identity"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/interface/restful/handler"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
	"github.com/fangcun-mount/iam-contracts/pkg/errors"
)

// UserModule 用户模块
// 负责组装用户相关的所有组件
type UserModule struct {
	// handler 层
	UserHandler         *handler.UserHandler
	ChildHandler        *handler.ChildHandler
	GuardianshipHandler *handler.GuardianshipHandler
	IdentityGRPCService *identitygrpc.Service
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

	// 初始化仓储
	userRepo := mysqluser.NewRepository(db)
	childRepo := mysqlchild.NewRepository(db)
	guardRepo := mysqlguard.NewRepository(db)

	// 事务
	uow := appuow.NewUnitOfWork(db)

	// 用户服务
	userRegisterSrv := appuser.NewRegisterService(userRepo)
	userProfileSrv := appuser.NewProfileService(userRepo)
	userQuerySrv := appuser.NewQueryService(userRepo)

	// 儿童服务
	childRegisterSrv := appchild.NewRegisterService(childRepo)
	childProfileSrv := appchild.NewProfileService(childRepo)
	childQuerySrv := appchild.NewQueryService(childRepo)

	// 监护服务
	guardManagerSrv := appguard.NewManagerService(guardRepo, childRepo, userRepo, uow)
	guardQuerySrv := appguard.NewQueryService(guardRepo, childRepo)

	// 初始化 handler 层
	m.UserHandler = handler.NewUserHandler(
		userRegisterSrv,
		userProfileSrv,
		userQuerySrv,
	)

	m.ChildHandler = handler.NewChildHandler(
		childRegisterSrv,
		childProfileSrv,
		childQuerySrv,
		guardManagerSrv,
		guardQuerySrv,
	)

	m.GuardianshipHandler = handler.NewGuardianshipHandler(
		guardManagerSrv,
		guardQuerySrv,
	)

	m.IdentityGRPCService = identitygrpc.NewService(
		userQuerySrv,
		childQuerySrv,
		guardQuerySrv,
	)

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
