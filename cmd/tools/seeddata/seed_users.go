package main

import (
	"context"
	"fmt"
	"strings"

	childApp "github.com/FangcunMount/iam-contracts/internal/apiserver/application/uc/child"
	guardApp "github.com/FangcunMount/iam-contracts/internal/apiserver/application/uc/guardianship"
	ucUOW "github.com/FangcunMount/iam-contracts/internal/apiserver/application/uc/uow"
	userApp "github.com/FangcunMount/iam-contracts/internal/apiserver/application/uc/user"
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
//
// 注意：所有操作必须在同一个事务外完成，确保每个应用服务调用都能看到之前创建的数据
func seedUserCenter(ctx context.Context, deps *dependencies, state *seedContext) error {
	// 初始化用户中心的工作单元和应用服务
	// 注意：每个应用服务方法都会开启独立事务，因此数据会立即提交
	uow := ucUOW.NewUnitOfWork(deps.DB)

	userAppSrv := userApp.NewUserApplicationService(uow)
	userProfileSrv := userApp.NewUserProfileApplicationService(uow)
	userQuerySrv := userApp.NewUserQueryApplicationService(uow)

	childAppSrv := childApp.NewChildApplicationService(uow)
	childQuerySrv := childApp.NewChildQueryApplicationService(uow)

	guardAppSrv := guardApp.NewGuardianshipApplicationService(uow)
	guardQuerySrv := guardApp.NewGuardianshipQueryApplicationService(uow)

	// 从配置文件读取用户数据
	users := make([]userSeed, 0, len(deps.Config.Users))
	for _, uc := range deps.Config.Users {
		users = append(users, userSeed{
			Alias:  uc.Alias,
			Name:   uc.Name,
			Phone:  uc.Phone,
			Email:  uc.Email,
			IDCard: uc.IDCard,
		})
	}

	// 从配置文件读取儿童数据
	children := make([]childSeed, 0, len(deps.Config.Children))
	for _, cc := range deps.Config.Children {
		// 将配置中的 gender (1=男, 2=女) 转换为 male/female
		gender := "male"
		if cc.Gender == 2 {
			gender = "female"
		}
		children = append(children, childSeed{
			Alias:    cc.Alias,
			Name:     cc.Name,
			IDCard:   cc.IDCard,
			Gender:   gender,
			Birthday: cc.Birthday,
			Height:   cc.Height,
			Weight:   cc.Weight,
		})
	}

	// 从配置文件读取监护关系数据
	guardianships := make([]guardianshipSeed, 0, len(deps.Config.Guardianships))
	for _, gc := range deps.Config.Guardianships {
		guardianships = append(guardianships, guardianshipSeed{
			UserAlias:  gc.UserAlias,
			ChildAlias: gc.ChildAlias,
			Relation:   gc.Relation,
		})
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

	// 重要：确保所有用户和儿童数据已经提交到数据库
	// 因为每个应用服务方法都在独立事务中运行，这里不需要显式提交
	// 但为了确保后续的 AddGuardian 能查询到数据，我们记录日志并继续
	deps.Logger.Infow("用户和儿童数据创建完成", "users", len(users), "children", len(children))

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

		// 如果已经是监护人则跳过（避免依赖错误字符串判断）
		isGuardian, err := guardQuerySrv.IsGuardian(ctx, userID, childID)
		if err != nil {
			return fmt.Errorf("add guardian %s->%s: failed to check existing guardian: %w", gs.UserAlias, gs.ChildAlias, err)
		}
		if isGuardian {
			deps.Logger.Infow("skip creating guardian because already exists", "user_alias", gs.UserAlias, "child_alias", gs.ChildAlias)
			continue
		}

		// 最后一次确认：使用非事务方式直接查询数据库，确保数据已提交
		// 这解决了应用服务在独立事务中查询不到刚创建数据的问题
		deps.Logger.Infow("验证用户和儿童数据存在", "user_id", userID, "child_id", childID)
		if u2, err := userQuerySrv.GetByID(ctx, userID); err != nil || u2 == nil {
			return fmt.Errorf("add guardian %s->%s: user verification failed before AddGuardian (id=%s, err=%v)", gs.UserAlias, gs.ChildAlias, userID, err)
		}
		if c2, err := childQuerySrv.GetByID(ctx, childID); err != nil || c2 == nil {
			return fmt.Errorf("add guardian %s->%s: child verification failed before AddGuardian (id=%s, err=%v)", gs.UserAlias, gs.ChildAlias, childID, err)
		}

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
