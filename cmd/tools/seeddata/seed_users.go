package main

import (
	"context"
	"fmt"
	"strings"

	childApp "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/application/child"
	guardApp "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/application/guardianship"
	ucUOW "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/application/uow"
	userApp "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/application/user"
)

// ==================== 用户中心相关类型定义 ====================

// userSeed 用户种子数据
type userSeed struct {
	Alias  string // 别名，用于后续引用
	Name   string
	Phone  string
	Email  string
	IDCard string
}

// childSeed 儿童种子数据
type childSeed struct {
	Alias    string // 别名，用于后续引用
	Name     string
	IDCard   string
	Gender   string
	Birthday string
	Height   int // 厘米的十分之一 (e.g., 1450 -> 145.0cm)
	Weight   int // 公斤的十分之一 (e.g., 350 -> 35.0kg)
}

// guardianshipSeed 监护关系种子数据
type guardianshipSeed struct {
	UserAlias  string // 用户别名
	ChildAlias string // 儿童别名
	Relation   string // 监护关系类型
}

// ==================== 用户中心 Seed 函数 ====================

// seedUserCenter 创建用户中心数据
//
// 业务说明：
// 1. 创建系统管理员和测试用户
// 2. 创建儿童档案
// 3. 建立监护关系
// 4. 返回的 state 保存用户ID和儿童ID，供后续步骤使用
//
// 幂等性：通过查询检查，已存在的用户会被更新而不是重复创建
func seedUserCenter(ctx context.Context, deps *dependencies, state *seedContext) error {
	// 初始化用户中心的工作单元和应用服务
	uow := ucUOW.NewUnitOfWork(deps.DB)

	userAppSrv := userApp.NewUserApplicationService(uow)
	userProfileSrv := userApp.NewUserProfileApplicationService(uow)
	userQuerySrv := userApp.NewUserQueryApplicationService(uow)

	childAppSrv := childApp.NewChildApplicationService(uow)
	childQuerySrv := childApp.NewChildQueryApplicationService(uow)

	guardAppSrv := guardApp.NewGuardianshipApplicationService(uow)

	// 定义用户数据
	users := []userSeed{
		{
			Alias:  "admin",
			Name:   "系统管理员",
			Phone:  "10086000001",
			Email:  "admin@system.com",
			IDCard: "110101199001011001",
		},
		{
			Alias:  "zhangsan",
			Name:   "张三",
			Phone:  "13800138000",
			Email:  "zhangsan@example.com",
			IDCard: "110101199001011002",
		},
		{
			Alias:  "lisi",
			Name:   "李四",
			Phone:  "13800138001",
			Email:  "lisi@example.com",
			IDCard: "110101199001011003",
		},
		{
			Alias:  "wangwu",
			Name:   "王五",
			Phone:  "13800138002",
			Email:  "wangwu@example.com",
			IDCard: "110101198001011004",
		},
		{
			Alias:  "zhaoliu",
			Name:   "赵六",
			Phone:  "13800138003",
			Email:  "zhaoliu@example.com",
			IDCard: "110101198001011005",
		},
	}

	// 定义儿童数据
	children := []childSeed{
		{
			Alias:    "xiaoming",
			Name:     "小明",
			IDCard:   "110101201501011001",
			Gender:   "male",
			Birthday: "2015-01-01",
			Height:   1450, // 145.0cm
			Weight:   350,  // 35.0kg
		},
		{
			Alias:    "xiaohong",
			Name:     "小红",
			IDCard:   "110101201502011002",
			Gender:   "female",
			Birthday: "2015-02-01",
			Height:   1420,
			Weight:   330,
		},
		{
			Alias:    "xiaogang",
			Name:     "小刚",
			IDCard:   "110101201603011003",
			Gender:   "male",
			Birthday: "2016-03-01",
			Height:   1380,
			Weight:   310,
		},
		{
			Alias:    "xiaoli",
			Name:     "小丽",
			IDCard:   "", // 可以没有身份证号
			Gender:   "female",
			Birthday: "2018-05-15",
			Height:   1100,
			Weight:   200,
		},
	}

	// 定义监护关系
	guardianships := []guardianshipSeed{
		{UserAlias: "wangwu", ChildAlias: "xiaoming", Relation: "parent"},
		{UserAlias: "zhaoliu", ChildAlias: "xiaohong", Relation: "parent"},
		{UserAlias: "wangwu", ChildAlias: "xiaogang", Relation: "guardian"},
		{UserAlias: "zhaoliu", ChildAlias: "xiaoli", Relation: "parent"},
	}

	// 1. 创建用户
	for _, us := range users {
		id, err := ensureUser(ctx, userAppSrv, userProfileSrv, userQuerySrv, us)
		if err != nil {
			return fmt.Errorf("ensure user %s: %w", us.Alias, err)
		}
		state.Users[us.Alias] = id
	}

	// 2. 创建儿童档案
	for _, cs := range children {
		id, err := ensureChild(ctx, childAppSrv, childQuerySrv, cs)
		if err != nil {
			return fmt.Errorf("ensure child %s: %w", cs.Alias, err)
		}
		state.Children[cs.Alias] = id
	}

	// 3. 建立监护关系
	for _, gs := range guardianships {
		userID := state.Users[gs.UserAlias]
		childID := state.Children[gs.ChildAlias]
		if userID == "" || childID == "" {
			deps.Logger.Warnw("skip guardian creation due to missing id", "user_alias", gs.UserAlias, "child_alias", gs.ChildAlias, "user_id", userID, "child_id", childID)
			continue
		}

		// 诊断：在调用应用服务前先确认用户和儿童在 DB 中可被查询到
		if u, err := userQuerySrv.GetByID(ctx, userID); err != nil {
			return fmt.Errorf("add guardian %s->%s: failed to query user %s: %w", gs.UserAlias, gs.ChildAlias, userID, err)
		} else if u == nil {
			return fmt.Errorf("add guardian %s->%s: user not found (id=%s)", gs.UserAlias, gs.ChildAlias, userID)
		}

		if c, err := childQuerySrv.GetByID(ctx, childID); err != nil {
			return fmt.Errorf("add guardian %s->%s: failed to query child %s: %w", gs.UserAlias, gs.ChildAlias, childID, err)
		} else if c == nil {
			return fmt.Errorf("add guardian %s->%s: child not found (id=%s)", gs.UserAlias, gs.ChildAlias, childID)
		}

		deps.Logger.Infow("creating guardian", "user_alias", gs.UserAlias, "child_alias", gs.ChildAlias, "user_id", userID, "child_id", childID)

		dto := guardApp.AddGuardianDTO{
			UserID:   userID,
			ChildID:  childID,
			Relation: gs.Relation,
		}
		if err := guardAppSrv.AddGuardian(ctx, dto); err != nil && !duplicateGuardian(err) {
			return fmt.Errorf("add guardian %s->%s: %w", gs.UserAlias, gs.ChildAlias, err)
		}
	}

	deps.Logger.Infow("✅ 用户中心数据已创建",
		"users", len(users),
		"children", len(children),
		"guardianships", len(guardianships),
	)
	return nil
}

// ==================== 辅助函数 ====================

// ensureUser 确保用户存在（如不存在则创建，如存在则更新）
func ensureUser(
	ctx context.Context,
	userAppSrv userApp.UserApplicationService,
	userProfileSrv userApp.UserProfileApplicationService,
	userQuerySrv userApp.UserQueryApplicationService,
	seed userSeed,
) (string, error) {
	// 先尝试通过手机号查询
	if res, err := userQuerySrv.GetByPhone(ctx, seed.Phone); err == nil && res != nil {
		// 用户已存在，更新信息
		if res.Name != seed.Name {
			_ = userProfileSrv.Rename(ctx, res.ID, seed.Name)
		}
		if res.Email != seed.Email || res.Phone != seed.Phone {
			_ = userProfileSrv.UpdateContact(ctx, userApp.UpdateContactDTO{
				UserID: res.ID,
				Phone:  seed.Phone,
				Email:  seed.Email,
			})
		}
		if seed.IDCard != "" && res.IDCard != seed.IDCard {
			_ = userProfileSrv.UpdateIDCard(ctx, res.ID, seed.IDCard)
		}
		return res.ID, nil
	}

	// 用户不存在，创建新用户
	created, err := userAppSrv.Register(ctx, userApp.RegisterUserDTO{
		Name:  seed.Name,
		Phone: seed.Phone,
		Email: seed.Email,
	})
	if err != nil {
		return "", err
	}

	// 如果有身份证号，更新身份证信息
	if seed.IDCard != "" {
		_ = userProfileSrv.UpdateIDCard(ctx, created.ID, seed.IDCard)
	}
	return created.ID, nil
}

// ensureChild 确保儿童档案存在（如不存在则创建）
func ensureChild(
	ctx context.Context,
	childAppSrv childApp.ChildApplicationService,
	childQuerySrv childApp.ChildQueryApplicationService,
	seed childSeed,
) (string, error) {
	// 如果有身份证号，先查询是否已存在
	if seed.IDCard != "" {
		if res, err := childQuerySrv.GetByIDCard(ctx, seed.IDCard); err == nil && res != nil {
			return res.ID, nil
		}
	}

	// 转换高度和体重单位
	heightCM := uint32(seed.Height / 10)     // 十分之一厘米 -> 厘米
	weightGrams := uint32(seed.Weight * 100) // 十分之一公斤 -> 克

	dto := childApp.RegisterChildDTO{
		Name:     seed.Name,
		Gender:   seed.Gender,
		Birthday: seed.Birthday,
		IDCard:   seed.IDCard,
		Height:   &heightCM,
		Weight:   &weightGrams,
	}

	created, err := childAppSrv.Register(ctx, dto)
	if err != nil {
		return "", err
	}

	return created.ID, nil
}

// duplicateGuardian 检查是否是重复的监护关系错误
func duplicateGuardian(err error) bool {
	return err != nil && strings.Contains(err.Error(), "already exists")
}
