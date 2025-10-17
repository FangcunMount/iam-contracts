package assembler

import (
	"gorm.io/gorm"

	appchild "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/application/child"
	appguard "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/application/guardianship"
	appuow "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/application/uow"
	appuser "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/application/user"
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
	// IdentityGRPCService 暂时注释，待实现
	// IdentityGRPCService *identitygrpc.Service
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

	// 事务
	uow := appuow.NewUnitOfWork(db)

	// 用户应用服务
	userAppSrv := appuser.NewUserApplicationService(uow)
	userProfileAppSrv := appuser.NewUserProfileApplicationService(uow)

	// 儿童应用服务
	childAppSrv := appchild.NewChildApplicationService(uow)
	childProfileAppSrv := appchild.NewChildProfileApplicationService(uow)

	// 监护关系应用服务
	guardAppSrv := appguard.NewGuardianshipApplicationService(uow)

	// 初始化 handler 层
	m.UserHandler = handler.NewUserHandler(
		userAppSrv,
		userProfileAppSrv,
	)

	m.ChildHandler = handler.NewChildHandler(
		childAppSrv,
		childProfileAppSrv,
		guardAppSrv,
	)

	m.GuardianshipHandler = handler.NewGuardianshipHandler(
		guardAppSrv,
	)

	// TODO: IdentityGRPCService 需要查询服务，暂时跳过
	// m.IdentityGRPCService = identitygrpc.NewService(...)

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
