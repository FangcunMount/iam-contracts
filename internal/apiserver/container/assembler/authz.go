package assembler

import (
	"fmt"

	casbin2 "github.com/casbin/casbin/v2"
	redis2 "github.com/go-redis/redis/v7"
	"gorm.io/gorm"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/application/assignment"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/application/policy"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/application/resource"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/authz/application/role"
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
	// Application Services
	RoleService       *role.Service
	AssignmentService *assignment.Service
	PolicyService     *policy.Service
	ResourceService   *resource.Service

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

	// 4. 初始化应用服务
	m.RoleService = role.NewService(roleRepository)
	m.AssignmentService = assignment.NewService(assignmentRepository, roleRepository, casbinAdapter)
	m.PolicyService = policy.NewService(policyVersionRepository, roleRepository, resourceRepository, casbinAdapter, versionNotifier)
	m.ResourceService = resource.NewService(resourceRepository)

	// 5. 初始化 HTTP 处理器
	m.RoleHandler = handler.NewRoleHandler(m.RoleService)
	m.AssignmentHandler = handler.NewAssignmentHandler(m.AssignmentService)
	m.PolicyHandler = handler.NewPolicyHandler(m.PolicyService)
	m.ResourceHandler = handler.NewResourceHandler(m.ResourceService)

	fmt.Printf("✅ Authz module initialized\n")
	return nil
}
