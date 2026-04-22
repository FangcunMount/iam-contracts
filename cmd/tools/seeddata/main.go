// Package main implements the IAM seed data tool.
//
// This tool populates the IAM database with initial data including:
// - Tenants (organizations)
// - Users and children profiles
// - Authentication accounts and credentials
// - Authorization resources and role assignments
// - Casbin policy rules
// - JWKS keys for JWT signing
//
// The tool is modularized into separate files:
// - seed_tenants.go: Tenant data seeding
// - seed_users.go: User center data seeding (users, children, guardianships)
// - seed_authn.go: Authentication account seeding
// - seed_authz.go: Authorization data seeding (resources, assignments, policies)
// - seed_jwks.go: JWKS key generation
//
// Usage:
//
//	go run ./cmd/tools/seeddata \
//	  --dsn "user:pass@tcp(host:port)/db" \
//	  --redis-cache "host:port" --redis-cache-password "pwd"
//
// See README.md for detailed documentation.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/FangcunMount/component-base/pkg/log"
	redis "github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/FangcunMount/iam-contracts/cmd/tools/internal/common"
)

// ==================== 核心类型定义 ====================

// seedStep represents a specific seeding step.
type seedStep string

// All available seed steps - 按职能划分的数据初始化步骤
const (
	// ===== 系统基础设施初始化 =====
	stepSystemInit seedStep = "system-init" // 系统基础设施初始化：租户 + JWKS密钥 + 微信应用

	// ===== 认证授权体系初始化 =====
	stepAuthnInit seedStep = "authn-init" // 认证授权体系初始化：角色 + 资源 + Casbin策略

	// ===== 认证账号回填 =====
	stepAuthnBackfill seedStep = "authn-backfill" // 认证账号回填：为数据库中已有用户补齐 account/credential

	// ===== 管理员账号初始化 =====
	stepAdminInit seedStep = "admin-init" // 管理员账号初始化：创建管理员用户 + 认证账号 + 分配角色权限 + QS员工

	// ===== 显式租户管理员自举 =====
	stepTenantBootstrapAdmin seedStep = "tenant-bootstrap-admin" // 显式租户 bootstrap：admin/operator 自举

	// ===== 家庭数据批量生成 =====
	stepFamilyInit seedStep = "family-init" // 家庭数据批量初始化：以家庭为单位批量生成测试数据
)

// defaultSteps defines the default execution order of all seed steps.
// 默认执行所有初始化步骤，按职能顺序：系统基础设施 → 认证授权体系 → 租户 bootstrap → 管理员账号
var defaultSteps = []seedStep{
	stepSystemInit,           // 系统基础设施：租户 + JWKS + 微信应用
	stepAuthnInit,            // 认证授权体系：完整的 RBAC 权限系统
	stepTenantBootstrapAdmin, // 显式租户 bootstrap：admin/operator 自举（无配置时自动跳过）
	stepAdminInit,            // 管理员账号：创建管理员并分配权限
	// stepFamilyInit, // 家庭数据批量初始化：以家庭为单位批量生成数据
}

// dependencies holds all external dependencies required by seed functions.
type dependencies struct {
	DB          *gorm.DB      // 数据库连接
	RedisCache  *redis.Client // Cache Redis客户端（可选，用于缓存、会话等）
	KeysDir     string        // JWKS密钥存储目录
	CasbinModel string        // Casbin模型文件路径
	Logger      log.Logger    // 日志记录器
	Config      *SeedConfig   // 种子数据配置
	OnConflict  string        // 数据已存在时的处理策略：skip/overwrite/fail
}

// seedContext holds the state and references created during seeding.
// This allows later steps to reference entities created by earlier steps.
type seedContext struct {
	Users     map[string]string // 用户别名 → 用户ID
	Children  map[string]string // 儿童别名 → 儿童ID
	Accounts  map[string]uint64 // 账号别名 → 账号ID
	Resources map[string]uint64 // 资源键 → 资源ID
	Roles     map[string]string // 角色别名 → 角色ID
}

// newSeedContext creates a new seed context with initialized maps.
func newSeedContext() *seedContext {
	return &seedContext{
		Users:     map[string]string{},
		Children:  map[string]string{},
		Accounts:  map[string]uint64{},
		Resources: map[string]uint64{},
		Roles:     map[string]string{},
	}
}

// ==================== 主程序入口 ====================

func main() {
	// 解析命令行参数
	dsnFlag := flag.String("dsn", "", "MySQL DSN, e.g. user:pass@tcp(host:3306)/iam?parseTime=true&loc=Local")
	redisCacheFlag := flag.String("redis-cache", "", "Cache Redis address host:port (optional, for caching, sessions, rate limiting)")
	redisCacheUsernameFlag := flag.String("redis-cache-username", "", "Cache Redis username (optional, for Redis 6.0+ ACL)")
	redisCachePasswordFlag := flag.String("redis-cache-password", "", "Cache Redis password (optional)")
	keysDirFlag := flag.String("keys-dir", "./tmp/keys", "Directory to store generated JWKS private keys")
	casbinModelFlag := flag.String("casbin-model", "configs/casbin_model.conf", "Path to casbin model configuration file")
	configFileFlag := flag.String("config", "configs/seeddata.yaml", "Path to seed data configuration file")
	stepsFlag := flag.String("steps", strings.Join(stepListToStrings(defaultSteps), ","), "Comma separated seed steps (system-init,authn-init,authn-backfill,tenant-bootstrap-admin,admin-init,family-init)")
	familyCountFlag := flag.Int("family-count", defaultFamilyCount, "Number of families to generate in family seed step")
	workerCountFlag := flag.Int("worker-count", defaultWorkerCount, "Number of concurrent workers for family seed step")
	mockFlag := flag.Bool("mock", false, "Append family-init after base seed steps to generate mock family/testee data")
	verboseFlag := flag.Bool("verbose", false, "Enable verbose output including SQL logs")
	devFlag := flag.Bool("dev", false, "Enable development mode: print full runtime logs (verbose) vs stats-only mode")
	onConflictFlag := flag.String("on-conflict", "skip", "Behavior when data already exists: skip|overwrite|fail")
	skipJWKSFlag := flag.Bool("skip-jwks", false, "Skip JWKS seeding (use existing keys generated by IAM)")
	flag.Parse()

	// 初始化日志
	logger := log.New(log.NewOptions())

	onConflict := strings.ToLower(*onConflictFlag)
	if onConflict != "skip" && onConflict != "overwrite" && onConflict != "fail" {
		logger.Fatalw("❌ 无效的冲突处理策略", "on_conflict", *onConflictFlag)
	}

	// 加载种子数据配置
	logger.Infow("📄 加载种子数据配置...", "config_file", *configFileFlag)
	seedConfig, err := LoadSeedConfig(*configFileFlag)
	if err != nil {
		logger.Fatalw("❌ 加载配置文件失败", "error", err, "file", *configFileFlag)
	}
	logger.Infow("✅ 配置文件加载成功", "tenants", len(seedConfig.Tenants), "users", len(seedConfig.Users))

	// 确保密钥目录存在
	if err = ensureDir(*keysDirFlag); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create keys directory: %v\n", err)
		os.Exit(1)
	}

	// 连接数据库
	dsn := common.ResolveDSN(*dsnFlag)
	db := common.MustOpenGORM(dsn, *verboseFlag)
	defer common.CloseGORM(db)

	// 连接 Cache Redis（可选）
	redisCacheAddr := common.ResolveRedisAddr(*redisCacheFlag)
	var redisCacheClient *redis.Client
	if redisCacheAddr != "" {
		redisCacheClient = common.MustOpenRedisWithAuth(redisCacheAddr, *redisCacheUsernameFlag, *redisCachePasswordFlag)
		defer func() {
			_ = redisCacheClient.Close()
		}()
	}

	// 创建依赖对象
	deps := &dependencies{
		DB:          db,
		RedisCache:  redisCacheClient,
		KeysDir:     *keysDirFlag,
		CasbinModel: *casbinModelFlag,
		Logger:      logger,
		Config:      seedConfig,
		OnConflict:  onConflict,
	}

	// 将 --dev 标志传递给 family seeder 的全局变量
	familyDevMode = *devFlag

	// 解析要执行的步骤
	stepOrder := parseSteps(*stepsFlag)
	stepOrder = applyMockMode(stepOrder, *mockFlag)
	ctx := context.Background()
	state := newSeedContext()

	if *mockFlag {
		logger.Infow("🧪 mock 模式已启用", "family_count", *familyCountFlag, "worker_count", *workerCountFlag)
	}
	logger.Infow("🚀 开始执行 seed 数据脚本", "steps", stepOrder)

	// 按顺序执行各个步骤
	for _, step := range stepOrder {
		switch step {
		case stepSystemInit:
			// 【系统基础设施初始化】租户 + JWKS密钥 + 微信应用
			logger.Infow("🏗️  开始系统基础设施初始化...")

			// 1. 创建租户
			if err := seedTenants(ctx, deps); err != nil {
				logger.Fatalw("❌ 租户创建失败", "error", err)
			}
			// 2. 生成 JWKS 密钥（可跳过，使用已有密钥）
			if *skipJWKSFlag {
				logger.Infow("⏭️  跳过 JWKS 密钥生成（skip-jwks=true）")
			} else {
				if err := seedJWKS(ctx, deps); err != nil {
					logger.Fatalw("❌ JWKS密钥生成失败", "error", err)
				}
			}
			// 3. 创建微信应用
			if err := seedWechatApps(ctx, deps); err != nil {
				logger.Fatalw("❌ 微信应用创建失败", "error", err)
			}

			logger.Infow("✅ 系统基础设施初始化完成")

		case stepAuthnInit:
			// 【认证授权体系初始化】完整的 RBAC 权限系统
			logger.Infow("🔐 开始认证授权体系初始化...")

			// 1. 创建角色
			if err := seedRoles(ctx, deps, state); err != nil {
				logger.Fatalw("❌ 角色创建失败", "error", err)
			}
			// 2. 创建资源
			if err := seedAuthzResources(ctx, deps, state); err != nil {
				logger.Fatalw("❌ 资源创建失败", "error", err)
			}
			// 3. Casbin 策略规则
			if err := seedCasbinPolicies(ctx, deps); err != nil {
				logger.Fatalw("❌ Casbin策略创建失败", "error", err)
			}

			logger.Infow("✅ 认证授权体系初始化完成")

		case stepAuthnBackfill:
			logger.Infow("🔁 开始认证账号回填...")
			if err := seedAuthnBackfill(ctx, deps, state, *workerCountFlag); err != nil {
				logger.Fatalw("❌ 认证账号回填失败", "error", err)
			}
			logger.Infow("✅ 认证账号回填完成")

		case stepAdminInit:
			// 【管理员账号初始化】创建管理员并分配权限
			logger.Infow("👤 开始管理员账号初始化...")

			// 1. 创建管理员用户
			if err := seedAdmin(ctx, deps, state); err != nil {
				logger.Fatalw("❌ 管理员用户创建失败", "error", err)
			}
			// 2. 创建认证账号
			if err := seedAuthn(ctx, deps, state); err != nil {
				logger.Fatalw("❌ 认证账号创建失败", "error", err)
			}
			// 3. 分配角色权限
			if err := seedRoleAssignments(ctx, deps, state); err != nil {
				logger.Warnw("⚠️  角色分配失败（非致命错误）", "error", err)
			}
			// 4. 创建 QS 员工
			if err := seedStaff(ctx, deps, state); err != nil {
				logger.Fatalw("❌ QS员工创建失败", "error", err)
			}

			logger.Infow("✅ 管理员账号初始化完成")

		case stepTenantBootstrapAdmin:
			logger.Infow("🏢 开始显式租户管理员自举...")
			if err := seedTenantBootstrapAdmins(ctx, deps, state); err != nil {
				logger.Fatalw("❌ 租户管理员自举失败", "error", err)
			}
			logger.Infow("✅ 显式租户管理员自举完成")

		case stepFamilyInit:
			// 【家庭数据批量初始化】以家庭为单位批量生成测试数据
			logger.Infow("👨‍👩‍👧‍👦 开始家庭数据批量生成...")
			if err := seedFamilyCenter(ctx, deps, *familyCountFlag, *workerCountFlag); err != nil {
				logger.Fatalw("❌ 家庭数据批量创建失败", "error", err)
			}
			logger.Infow("✅ 家庭数据批量生成完成")

		default:
			logger.Warnw("⚠️  未知的 seed 步骤，跳过", "step", step)
		}
	}

	logger.Infow("🎉 所有 seed 步骤执行完成", "total_steps", len(stepOrder))
}

// ==================== 通用辅助函数 ====================

// parseSteps 解析步骤字符串为步骤列表
func parseSteps(raw string) []seedStep {
	if strings.TrimSpace(raw) == "" {
		return defaultSteps
	}
	items := strings.Split(raw, ",")
	var steps []seedStep
	for _, item := range items {
		item = strings.TrimSpace(strings.ToLower(item))
		if item == "" {
			continue
		}
		steps = append(steps, seedStep(item))
	}
	return steps
}

func applyMockMode(steps []seedStep, enabled bool) []seedStep {
	if !enabled {
		return steps
	}
	for _, step := range steps {
		if step == stepFamilyInit {
			return steps
		}
	}
	return append(append([]seedStep{}, steps...), stepFamilyInit)
}

// stepListToStrings 将步骤列表转换为字符串列表
func stepListToStrings(steps []seedStep) []string {
	out := make([]string, 0, len(steps))
	for _, s := range steps {
		out = append(out, string(s))
	}
	return out
}

// ensureDir 确保目录存在
func ensureDir(path string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("path is empty")
	}
	return os.MkdirAll(filepath.Clean(path), 0o755)
}
