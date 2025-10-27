package guardianship_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/application/child"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/application/guardianship"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/application/testutil"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/application/uow"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/uc/application/user"
)

// ==================== GuardianshipApplicationService 测试 ====================

func TestGuardianshipApplicationService_AddGuardian_Success(t *testing.T) {
	// Arrange
	db := testutil.SetupTestDB(t)
	unitOfWork := uow.NewUnitOfWork(db)
	ctx := context.Background()

	// 先创建用户和儿童
	userService := user.NewUserApplicationService(unitOfWork)
	userResult, err := userService.Register(ctx, user.RegisterUserDTO{
		Name:  "张三",
		Phone: "13800138000",
		Email: "zhang3@test.com",
	})
	require.NoError(t, err)

	childService := child.NewChildApplicationService(unitOfWork)
	childResult, err := childService.Register(ctx, child.RegisterChildDTO{
		Name:     "小明",
		Gender:   "male",
		Birthday: "2020-01-15",
	})
	require.NoError(t, err)

	guardianshipService := guardianship.NewGuardianshipApplicationService(unitOfWork)

	// Act - 添加监护关系
	dto := guardianship.AddGuardianDTO{
		UserID:   userResult.ID,
		ChildID:  childResult.ID,
		Relation: "parent",
	}
	err = guardianshipService.AddGuardian(ctx, dto)

	// Assert
	require.NoError(t, err)

	// 验证监护关系是否创建成功
	queryService := guardianship.NewGuardianshipQueryApplicationService(unitOfWork)
	isGuardian, err := queryService.IsGuardian(ctx, userResult.ID, childResult.ID)
	require.NoError(t, err)
	assert.True(t, isGuardian)
}

func TestGuardianshipApplicationService_AddGuardian_DuplicateGuardian(t *testing.T) {
	// Arrange
	db := testutil.SetupTestDB(t)
	unitOfWork := uow.NewUnitOfWork(db)
	ctx := context.Background()

	// 先创建用户和儿童
	userService := user.NewUserApplicationService(unitOfWork)
	userResult, err := userService.Register(ctx, user.RegisterUserDTO{
		Name:  "李四",
		Phone: "13800138001",
		Email: "li4@test.com",
	})
	require.NoError(t, err)

	childService := child.NewChildApplicationService(unitOfWork)
	childResult, err := childService.Register(ctx, child.RegisterChildDTO{
		Name:     "小红",
		Gender:   "female",
		Birthday: "2019-05-20",
	})
	require.NoError(t, err)

	guardianshipService := guardianship.NewGuardianshipApplicationService(unitOfWork)

	// 第一次添加监护关系
	dto := guardianship.AddGuardianDTO{
		UserID:   userResult.ID,
		ChildID:  childResult.ID,
		Relation: "parent",
	}
	err = guardianshipService.AddGuardian(ctx, dto)
	require.NoError(t, err)

	// Act - 尝试重复添加相同的监护关系
	err = guardianshipService.AddGuardian(ctx, dto)

	// Assert - 应该失败
	require.Error(t, err)
}

func TestGuardianshipApplicationService_RemoveGuardian_Success(t *testing.T) {
	// Arrange
	db := testutil.SetupTestDB(t)
	unitOfWork := uow.NewUnitOfWork(db)
	ctx := context.Background()

	// 先创建用户、儿童和监护关系
	userService := user.NewUserApplicationService(unitOfWork)
	userResult, err := userService.Register(ctx, user.RegisterUserDTO{
		Name:  "王五",
		Phone: "13800138002",
		Email: "wang5@test.com",
	})
	require.NoError(t, err)

	childService := child.NewChildApplicationService(unitOfWork)
	childResult, err := childService.Register(ctx, child.RegisterChildDTO{
		Name:     "小强",
		Gender:   "male",
		Birthday: "2021-03-10",
	})
	require.NoError(t, err)

	guardianshipService := guardianship.NewGuardianshipApplicationService(unitOfWork)
	err = guardianshipService.AddGuardian(ctx, guardianship.AddGuardianDTO{
		UserID:   userResult.ID,
		ChildID:  childResult.ID,
		Relation: "parent",
	})
	require.NoError(t, err)

	// Act - 移除监护关系
	dto := guardianship.RemoveGuardianDTO{
		UserID:  userResult.ID,
		ChildID: childResult.ID,
	}
	err = guardianshipService.RemoveGuardian(ctx, dto)

	// Assert
	require.NoError(t, err)

	// 验证监护关系是否已移除
	queryService := guardianship.NewGuardianshipQueryApplicationService(unitOfWork)
	result, err := queryService.GetByUserIDAndChildID(ctx, userResult.ID, childResult.ID)
	// 监护关系应该仍然存在但状态为已撤销
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGuardianshipApplicationService_RemoveGuardian_NotFound(t *testing.T) {
	// Arrange
	db := testutil.SetupTestDB(t)
	unitOfWork := uow.NewUnitOfWork(db)
	guardianshipService := guardianship.NewGuardianshipApplicationService(unitOfWork)
	ctx := context.Background()

	// Act - 尝试移除不存在的监护关系
	dto := guardianship.RemoveGuardianDTO{
		UserID:  "999999999999999999",
		ChildID: "888888888888888888",
	}
	err := guardianshipService.RemoveGuardian(ctx, dto)

	// Assert - 应该失败
	require.Error(t, err)
}

func TestGuardianshipApplicationService_RegisterChildWithGuardian_Success(t *testing.T) {
	// Arrange
	db := testutil.SetupTestDB(t)
	unitOfWork := uow.NewUnitOfWork(db)
	ctx := context.Background()

	// 先创建用户
	userService := user.NewUserApplicationService(unitOfWork)
	userResult, err := userService.Register(ctx, user.RegisterUserDTO{
		Name:  "赵六",
		Phone: "13800138003",
		Email: "zhao6@test.com",
	})
	require.NoError(t, err)

	guardianshipService := guardianship.NewGuardianshipApplicationService(unitOfWork)

	// Act - 同时注册儿童和监护关系
	dto := guardianship.RegisterChildWithGuardianDTO{
		ChildName:     "小丽",
		ChildGender:   "female",
		ChildBirthday: "2020-06-15",
		UserID:        userResult.ID,
		Relation:      "parent",
	}
	result, err := guardianshipService.RegisterChildWithGuardian(ctx, dto)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.ChildID)
	assert.Equal(t, userResult.ID, result.UserID)
	assert.Equal(t, "小丽", result.ChildName)
}

func TestGuardianshipApplicationService_RegisterChildWithGuardian_UserNotFound(t *testing.T) {
	// Arrange
	db := testutil.SetupTestDB(t)
	unitOfWork := uow.NewUnitOfWork(db)
	guardianshipService := guardianship.NewGuardianshipApplicationService(unitOfWork)
	ctx := context.Background()

	// Act - 使用不存在的用户ID注册
	dto := guardianship.RegisterChildWithGuardianDTO{
		ChildName:     "小明",
		ChildGender:   "male",
		ChildBirthday: "2020-01-01",
		UserID:        "999999999999999999", // 不存在的用户
		Relation:      "parent",
	}
	result, err := guardianshipService.RegisterChildWithGuardian(ctx, dto)

	// Assert - 应该失败
	require.Error(t, err)
	assert.Nil(t, result)
}

// ==================== GuardianshipQueryApplicationService 测试 ====================

func TestGuardianshipQueryApplicationService_IsGuardian_True(t *testing.T) {
	// Arrange
	db := testutil.SetupTestDB(t)
	unitOfWork := uow.NewUnitOfWork(db)
	ctx := context.Background()

	// 创建用户、儿童和监护关系
	userService := user.NewUserApplicationService(unitOfWork)
	userResult, err := userService.Register(ctx, user.RegisterUserDTO{
		Name:  "孙七",
		Phone: "13800138004",
		Email: "sun7@test.com",
	})
	require.NoError(t, err)

	childService := child.NewChildApplicationService(unitOfWork)
	childResult, err := childService.Register(ctx, child.RegisterChildDTO{
		Name:     "小虎",
		Gender:   "male",
		Birthday: "2020-08-20",
	})
	require.NoError(t, err)

	guardianshipService := guardianship.NewGuardianshipApplicationService(unitOfWork)
	err = guardianshipService.AddGuardian(ctx, guardianship.AddGuardianDTO{
		UserID:   userResult.ID,
		ChildID:  childResult.ID,
		Relation: "parent",
	})
	require.NoError(t, err)

	queryService := guardianship.NewGuardianshipQueryApplicationService(unitOfWork)

	// Act - 检查是否为监护人
	isGuardian, err := queryService.IsGuardian(ctx, userResult.ID, childResult.ID)

	// Assert
	require.NoError(t, err)
	assert.True(t, isGuardian)
}

func TestGuardianshipQueryApplicationService_IsGuardian_False(t *testing.T) {
	// Arrange
	db := testutil.SetupTestDB(t)
	unitOfWork := uow.NewUnitOfWork(db)
	queryService := guardianship.NewGuardianshipQueryApplicationService(unitOfWork)
	ctx := context.Background()

	// Act - 检查不存在的监护关系
	isGuardian, err := queryService.IsGuardian(ctx, "999999999999999999", "888888888888888888")

	// Assert
	require.NoError(t, err)
	assert.False(t, isGuardian)
}

func TestGuardianshipQueryApplicationService_GetByUserIDAndChildID_Success(t *testing.T) {
	// Arrange
	db := testutil.SetupTestDB(t)
	unitOfWork := uow.NewUnitOfWork(db)
	ctx := context.Background()

	// 创建用户、儿童和监护关系
	userService := user.NewUserApplicationService(unitOfWork)
	userResult, err := userService.Register(ctx, user.RegisterUserDTO{
		Name:  "周八",
		Phone: "13800138005",
		Email: "zhou8@test.com",
	})
	require.NoError(t, err)

	childService := child.NewChildApplicationService(unitOfWork)
	childResult, err := childService.Register(ctx, child.RegisterChildDTO{
		Name:     "小龙",
		Gender:   "male",
		Birthday: "2019-12-25",
	})
	require.NoError(t, err)

	guardianshipService := guardianship.NewGuardianshipApplicationService(unitOfWork)
	err = guardianshipService.AddGuardian(ctx, guardianship.AddGuardianDTO{
		UserID:   userResult.ID,
		ChildID:  childResult.ID,
		Relation: "grandparents",
	})
	require.NoError(t, err)

	queryService := guardianship.NewGuardianshipQueryApplicationService(unitOfWork)

	// Act - 查询监护关系
	result, err := queryService.GetByUserIDAndChildID(ctx, userResult.ID, childResult.ID)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, userResult.ID, result.UserID)
	assert.Equal(t, childResult.ID, result.ChildID)
	assert.Equal(t, "小龙", result.ChildName)
}

func TestGuardianshipQueryApplicationService_GetByUserIDAndChildID_NotFound(t *testing.T) {
	// Arrange
	db := testutil.SetupTestDB(t)
	unitOfWork := uow.NewUnitOfWork(db)
	queryService := guardianship.NewGuardianshipQueryApplicationService(unitOfWork)
	ctx := context.Background()

	// Act - 查询不存在的监护关系
	result, err := queryService.GetByUserIDAndChildID(ctx, "999999999999999999", "888888888888888888")

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
}

func TestGuardianshipQueryApplicationService_ListChildrenByUserID_Success(t *testing.T) {
	// Arrange
	db := testutil.SetupTestDB(t)
	unitOfWork := uow.NewUnitOfWork(db)
	ctx := context.Background()

	// 创建用户
	userService := user.NewUserApplicationService(unitOfWork)
	userResult, err := userService.Register(ctx, user.RegisterUserDTO{
		Name:  "吴九",
		Phone: "13800138006",
		Email: "wu9@test.com",
	})
	require.NoError(t, err)

	// 创建多个儿童
	childService := child.NewChildApplicationService(unitOfWork)
	child1, err := childService.Register(ctx, child.RegisterChildDTO{
		Name:     "大宝",
		Gender:   "male",
		Birthday: "2018-01-01",
		IDCard:   "110101201801011111",
	})
	require.NoError(t, err)

	child2, err := childService.Register(ctx, child.RegisterChildDTO{
		Name:     "二宝",
		Gender:   "female",
		Birthday: "2020-06-01",
		IDCard:   "110101202006012222",
	})
	require.NoError(t, err)

	// 添加监护关系
	guardianshipService := guardianship.NewGuardianshipApplicationService(unitOfWork)
	err = guardianshipService.AddGuardian(ctx, guardianship.AddGuardianDTO{
		UserID:   userResult.ID,
		ChildID:  child1.ID,
		Relation: "parent",
	})
	require.NoError(t, err)

	err = guardianshipService.AddGuardian(ctx, guardianship.AddGuardianDTO{
		UserID:   userResult.ID,
		ChildID:  child2.ID,
		Relation: "parent",
	})
	require.NoError(t, err)

	queryService := guardianship.NewGuardianshipQueryApplicationService(unitOfWork)

	// Act - 列出用户的所有儿童
	results, err := queryService.ListChildrenByUserID(ctx, userResult.ID)

	// Assert
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestGuardianshipQueryApplicationService_ListGuardiansByChildID_Success(t *testing.T) {
	// Arrange
	db := testutil.SetupTestDB(t)
	unitOfWork := uow.NewUnitOfWork(db)
	ctx := context.Background()

	// 创建多个用户 (设置唯一的email避免UNIQUE约束冲突)
	userService := user.NewUserApplicationService(unitOfWork)
	user1, err := userService.Register(ctx, user.RegisterUserDTO{
		Name:  "爸爸",
		Phone: "13800138007",
		Email: "father@example.com", // 唯一email
	})
	require.NoError(t, err)

	// 设置唯一的 IDCard 避免 UNIQUE 约束冲突
	profileService := user.NewUserProfileApplicationService(unitOfWork)
	err = profileService.UpdateIDCard(ctx, user1.ID, "320106198001011111")
	require.NoError(t, err)

	user2, err := userService.Register(ctx, user.RegisterUserDTO{
		Name:  "妈妈",
		Phone: "13800138008",
		Email: "mother@example.com", // 唯一email
	})
	require.NoError(t, err)

	// 设置唯一的 IDCard 避免 UNIQUE 约束冲突
	err = profileService.UpdateIDCard(ctx, user2.ID, "320106198001012222")
	require.NoError(t, err)

	// 创建儿童
	childService := child.NewChildApplicationService(unitOfWork)
	childResult, err := childService.Register(ctx, child.RegisterChildDTO{
		Name:     "宝宝",
		Gender:   "female",
		Birthday: "2021-01-01",
	})
	require.NoError(t, err)

	// 添加多个监护人
	guardianshipService := guardianship.NewGuardianshipApplicationService(unitOfWork)
	err = guardianshipService.AddGuardian(ctx, guardianship.AddGuardianDTO{
		UserID:   user1.ID,
		ChildID:  childResult.ID,
		Relation: "parent",
	})
	require.NoError(t, err)

	err = guardianshipService.AddGuardian(ctx, guardianship.AddGuardianDTO{
		UserID:   user2.ID,
		ChildID:  childResult.ID,
		Relation: "parent",
	})
	require.NoError(t, err)

	queryService := guardianship.NewGuardianshipQueryApplicationService(unitOfWork)

	// Act - 列出儿童的所有监护人
	results, err := queryService.ListGuardiansByChildID(ctx, childResult.ID)

	// Assert
	require.NoError(t, err)
	assert.Len(t, results, 2)
}
