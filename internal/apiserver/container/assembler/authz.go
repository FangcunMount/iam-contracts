package assembler

import (
	"fmt"

	casbin2 "github.com/casbin/casbin/v2"
	redis2 "github.com/go-redis/redis/v7"
	"gorm.io/gorm"

	assignmentApp "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/application/assignment"
	policyApp "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/application/policy"
	resourceApp "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/application/resource"
	roleApp "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/application/role"
	assignmentService "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/assignment/service"
	policyService "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/policy/service"
	resourceService "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/resource/service"
	roleService "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/domain/role/service"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/infra/casbin"
	assignmentInfra "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/infra/mysql/assignment"
	policyInfra "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/infra/mysql/policy"
	resourceInfra "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/infra/mysql/resource"
	roleInfra "github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/infra/mysql/role"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/infra/redis"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/interface/restful/handler"
)

// AuthzModule 授权模块
type AuthzModule struct {
	// HTTP Handlers
	RoleHandler       *handler.RoleHandler
	AssignmentHandler *handler.AssignmentHandler
	PolicyHandler     *handler.PolicyHandler
	ResourceHandler   *handler.ResourceHandler

	// Infrastructure
	Enforcer *casbin2.Enforcer
}

// NewAuthzModule 创建授权模块
func NewAuthzModule() *AuthzModule {
	return &AuthzModule{}
}

// Initialize 初始化授权模块
func (m *AuthzModule) Initialize(db *gorm.DB, redisClient *redis2.Client) error {
	if db == nil {
		return fmt.Errorf("mysql db is required")
	}
	if redisClient == nil {
		return fmt.Errorf("redis client is required")
	}

	// 1. 初始化 Casbin Enforcer
	// TODO: 配置 Casbin 模型文件路径
	modelPath := "configs/casbin_model.conf"
	casbinAdapter, err := casbin.NewCasbinAdapter(db, modelPath)
	if err != nil {
		return fmt.Errorf("failed to create casbin adapter: %w", err)
	}

	// 2. 初始化仓储层
	roleRepository := roleInfra.NewRoleRepository(db)
	assignmentRepository := assignmentInfra.NewAssignmentRepository(db)
	resourceRepository := resourceInfra.NewResourceRepository(db)
	policyVersionRepository := policyInfra.NewPolicyVersionRepository(db)

	// 3. 初始化版本通知器
	versionNotifier := redis.NewVersionNotifier(
		// TODO: 需要转换 redis v7 到 v9，或者暂时使用 nil
		nil,
		"authz:policy_changed",
	)

	// 4. 初始化领域服务
	// Resource 模块
	resourceManager := resourceService.NewResourceManager(resourceRepository)
	// Role 模块
	roleManager := roleService.NewRoleManager(roleRepository)
	// Policy 模块
	policyManager := policyService.NewPolicyManager(roleRepository, resourceRepository)
	// Assignment 模块
	assignmentManager := assignmentService.NewAssignmentManager(assignmentRepository, roleRepository)

	// 5. 初始化应用服务 - CQRS 分离
	// Resource 模块
	resourceCommander := resourceApp.NewResourceCommandService(resourceManager, resourceRepository)
	resourceQueryer := resourceApp.NewResourceQueryService(resourceRepository)
	// Role 模块
	roleCommander := roleApp.NewRoleCommandService(roleManager, roleRepository)
	roleQueryer := roleApp.NewRoleQueryService(roleRepository)
	// Policy 模块
	policyCommander := policyApp.NewPolicyCommandService(policyManager, policyVersionRepository, casbinAdapter, versionNotifier)
	policyQueryer := policyApp.NewPolicyQueryService(policyManager, policyVersionRepository, casbinAdapter)
	// Assignment 模块
	assignmentCommander := assignmentApp.NewAssignmentCommandService(assignmentManager, assignmentRepository, casbinAdapter)
	assignmentQueryer := assignmentApp.NewAssignmentQueryService(assignmentManager, assignmentRepository)

	// 6. 初始化 HTTP 处理器 - 依赖 driving 接口（CQRS）
	// Resource Handler
	m.ResourceHandler = handler.NewResourceHandler(resourceCommander, resourceQueryer)
	// Role Handler
	m.RoleHandler = handler.NewRoleHandler(roleCommander, roleQueryer)
	// Policy Handler
	m.PolicyHandler = handler.NewPolicyHandler(policyCommander, policyQueryer)
	// Assignment Handler
	m.AssignmentHandler = handler.NewAssignmentHandler(assignmentCommander, assignmentQueryer)
	return nil
}
