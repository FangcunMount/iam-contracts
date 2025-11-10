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
//	  --redis-cache "host:port" --redis-cache-password "pwd" \
//	  --redis-store "host:port" --redis-store-password "pwd"
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

// All available seed steps.
const (
	stepTenants     seedStep = "tenants"     // 创建租户数据
	stepUserCenter  seedStep = "user"        // 创建用户、儿童、监护关系
	stepAuthn       seedStep = "authn"       // 创建认证账号和凭证
	stepRoles       seedStep = "roles"       // 创建基础角色
	stepResources   seedStep = "resources"   // 创建授权资源
	stepAssignments seedStep = "assignments" // 创建角色分配
	stepCasbin      seedStep = "casbin"      // 创建Casbin策略规则
	stepJWKS        seedStep = "jwks"        // 生成JWKS密钥
	stepWechatApp   seedStep = "wechatapp"   // 创建微信应用
)

// defaultSteps defines the default execution order of all seed steps.
var defaultSteps = []seedStep{
	stepTenants,
	stepUserCenter,
	stepAuthn,
	stepRoles,
	stepResources,
	stepAssignments,
	stepCasbin,
	stepJWKS,
	stepWechatApp,
}

// dependencies holds all external dependencies required by seed functions.
type dependencies struct {
	DB          *gorm.DB      // 数据库连接
	RedisCache  *redis.Client // Cache Redis客户端（可选，用于缓存、会话等）
	RedisStore  *redis.Client // Store Redis客户端（可选，用于持久化存储）
	KeysDir     string        // JWKS密钥存储目录
	CasbinModel string        // Casbin模型文件路径
	Logger      log.Logger    // 日志记录器
	Config      *SeedConfig   // 种子数据配置
}

// seedContext holds the state and references created during seeding.
// This allows later steps to reference entities created by earlier steps.
type seedContext struct {
	Users     map[string]string // 用户别名 → 用户ID
	Children  map[string]string // 儿童别名 → 儿童ID
	Accounts  map[string]uint64 // 账号别名 → 账号ID
	Resources map[string]uint64 // 资源键 → 资源ID
}

// newSeedContext creates a new seed context with initialized maps.
func newSeedContext() *seedContext {
	return &seedContext{
		Users:     map[string]string{},
		Children:  map[string]string{},
		Accounts:  map[string]uint64{},
		Resources: map[string]uint64{},
	}
}

// ==================== 主程序入口 ====================

func main() {
	// 解析命令行参数
	dsnFlag := flag.String("dsn", "", "MySQL DSN, e.g. user:pass@tcp(host:3306)/iam_contracts?parseTime=true&loc=Local")
	redisCacheFlag := flag.String("redis-cache", "", "Cache Redis address host:port (optional, for caching, sessions, rate limiting)")
	redisCacheUsernameFlag := flag.String("redis-cache-username", "", "Cache Redis username (optional, for Redis 6.0+ ACL)")
	redisCachePasswordFlag := flag.String("redis-cache-password", "", "Cache Redis password (optional)")
	redisStoreFlag := flag.String("redis-store", "", "Store Redis address host:port (optional, for persistent storage, queues)")
	redisStoreUsernameFlag := flag.String("redis-store-username", "", "Store Redis username (optional, for Redis 6.0+ ACL)")
	redisStorePasswordFlag := flag.String("redis-store-password", "", "Store Redis password (optional)")
	keysDirFlag := flag.String("keys-dir", "./tmp/keys", "Directory to store generated JWKS private keys")
	casbinModelFlag := flag.String("casbin-model", "configs/casbin_model.conf", "Path to casbin model configuration file")
	configFileFlag := flag.String("config", "configs/seeddata.yaml", "Path to seed data configuration file")
	stepsFlag := flag.String("steps", strings.Join(stepListToStrings(defaultSteps), ","), "Comma separated seed steps (tenants,user,authn,resources,assignments,casbin,jwks)")
	flag.Parse()

	// 初始化日志
	logger := log.New(log.NewOptions())

	// 加载种子数据配置
	logger.Infow("📄 加载种子数据配置...", "config_file", *configFileFlag)
	seedConfig, err := LoadSeedConfig(*configFileFlag)
	if err != nil {
		logger.Fatalw("❌ 加载配置文件失败", "error", "file", *configFileFlag)
	}
	logger.Infow("✅ 配置文件加载成功", "tenants", len(seedConfig.Tenants), "users", len(seedConfig.Users))

	// 确保密钥目录存在
	if err = ensureDir(*keysDirFlag); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create keys directory: %v\n", err)
		os.Exit(1)
	}

	// 连接数据库
	dsn := common.ResolveDSN(*dsnFlag)
	db := common.MustOpenGORM(dsn)
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

	// 连接 Store Redis（可选）
	redisStoreAddr := common.ResolveRedisAddr(*redisStoreFlag)
	var redisStoreClient *redis.Client
	if redisStoreAddr != "" {
		redisStoreClient = common.MustOpenRedisWithAuth(redisStoreAddr, *redisStoreUsernameFlag, *redisStorePasswordFlag)
		defer func() {
			_ = redisStoreClient.Close()
		}()
	}

	// 创建依赖对象
	deps := &dependencies{
		DB:          db,
		RedisCache:  redisCacheClient,
		RedisStore:  redisStoreClient,
		KeysDir:     *keysDirFlag,
		CasbinModel: *casbinModelFlag,
		Logger:      logger,
		Config:      seedConfig,
	}

	// 解析要执行的步骤
	stepOrder := parseSteps(*stepsFlag)
	ctx := context.Background()
	state := newSeedContext()

	logger.Infow("🚀 开始执行 seed 数据脚本", "steps", stepOrder)

	// 按顺序执行各个步骤
	for _, step := range stepOrder {
		switch step {
		case stepTenants:
			if err := seedTenants(ctx, deps); err != nil {
				logger.Fatalw("❌ 租户数据创建失败", "error", err)
			}
		case stepUserCenter:
			if err := seedUserCenter(ctx, deps, state); err != nil {
				logger.Fatalw("❌ 用户中心数据创建失败", "error", err)
			}
		case stepAuthn:
			if err := seedAuthn(ctx, deps, state); err != nil {
				logger.Fatalw("❌ 认证账号数据创建失败", "error", err)
			}
		case stepRoles:
			if err := seedAuthzRoles(ctx, deps); err != nil {
				logger.Fatalw("❌ 基础角色数据创建失败", "error", err)
			}
		case stepResources:
			if err := seedAuthzResources(ctx, deps, state); err != nil {
				logger.Fatalw("❌ 授权资源数据创建失败", "error", err)
			}
		case stepAssignments:
			if err := seedRoleAssignments(ctx, deps, state); err != nil {
				logger.Fatalw("❌ 角色分配数据创建失败", "error", err)
			}
		case stepCasbin:
			if err := seedCasbinPolicies(ctx, deps); err != nil {
				logger.Fatalw("❌ Casbin策略创建失败", "error", err)
			}
		case stepJWKS:
			if err := seedJWKS(ctx, deps); err != nil {
				logger.Fatalw("❌ JWKS密钥生成失败", "error", err)
			}
		case stepWechatApp:
			if err := seedWechatApps(ctx, deps); err != nil {
				logger.Fatalw("❌ 微信应用数据创建失败", "error", err)
			}
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
