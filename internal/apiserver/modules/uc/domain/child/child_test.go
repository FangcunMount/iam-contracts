package child_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/child"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/meta"
	"github.com/fangcun-mount/iam-contracts/pkg/errors"
)

// ==================== Child 实体创建测试 ====================

func TestNewChild_Success(t *testing.T) {
	// Arrange
	name := "小明"

	// Act
	c, err := child.NewChild(name)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, c)
	assert.Equal(t, name, c.Name)
}

func TestNewChild_WithOptions(t *testing.T) {
	// Arrange
	name := "小红"
	childID := child.NewChildID(12345)
	idCard := meta.NewIDCard("小红", "110101201501011234")
	gender := meta.GenderFemale
	birthday := meta.NewBirthday("2015-01-01")
	height, _ := meta.NewHeightFromFloat(120.5)
	weight, _ := meta.NewWeightFromFloat(25.8)

	// Act
	c, err := child.NewChild(
		name,
		child.WithChildID(childID),
		child.WithIDCard(idCard),
		child.WithGender(gender),
		child.WithBirthday(birthday),
		child.WithHeight(height),
		child.WithWeight(weight),
	)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, c)
	assert.Equal(t, name, c.Name)
	assert.Equal(t, childID, c.ID)
	assert.Equal(t, idCard, c.IDCard)
	assert.Equal(t, gender, c.Gender)
	assert.Equal(t, birthday, c.Birthday)
	assert.Equal(t, height, c.Height)
	assert.Equal(t, weight, c.Weight)
}

func TestNewChild_EmptyName_ShouldFail(t *testing.T) {
	// Arrange
	name := ""

	// Act
	c, err := child.NewChild(name)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, c)
	assert.True(t, errors.IsCode(err, code.ErrUserBasicInfoInvalid))
	errMsg := fmt.Sprintf("%+v", err)
	assert.Contains(t, errMsg, "name cannot be empty")
}

// ==================== Child 信息更新测试 ====================

func TestChild_Rename(t *testing.T) {
	// Arrange
	c, _ := child.NewChild("原名字")
	newName := "新名字"

	// Act
	c.Rename(newName)

	// Assert
	assert.Equal(t, newName, c.Name)
}

func TestChild_UpdateIDCard(t *testing.T) {
	// Arrange
	c, _ := child.NewChild("小李")
	newIDCard := meta.NewIDCard("小李", "320106201501010001")

	// Act
	c.UpdateIDCard(newIDCard)

	// Assert
	assert.Equal(t, newIDCard, c.IDCard)
}

func TestChild_UpdateProfile(t *testing.T) {
	// Arrange
	c, _ := child.NewChild("小王")
	newGender := meta.GenderMale
	newBirthday := meta.NewBirthday("2016-06-01")

	// Act
	c.UpdateProfile(newGender, newBirthday)

	// Assert
	assert.Equal(t, newGender, c.Gender)
	assert.Equal(t, newBirthday, c.Birthday)
}

func TestChild_UpdateHeightWeight(t *testing.T) {
	// Arrange
	c, _ := child.NewChild("小赵")
	newHeight, _ := meta.NewHeightFromFloat(125.0)
	newWeight, _ := meta.NewWeightFromFloat(28.5)

	// Act
	c.UpdateHeightWeight(newHeight, newWeight)

	// Assert
	assert.Equal(t, newHeight, c.Height)
	assert.Equal(t, newWeight, c.Weight)
}

// ==================== ChildID 值对象测试 ====================

func TestNewChildID(t *testing.T) {
	// Act
	id := child.NewChildID(67890)

	// Assert
	assert.Equal(t, uint64(67890), id.Value())
}

// ==================== Child 综合场景测试 ====================

func TestChild_CompleteLifecycle(t *testing.T) {
	// 创建儿童档案
	c, err := child.NewChild("小测试")
	require.NoError(t, err)
	assert.Equal(t, "小测试", c.Name)

	// 设置身份证
	idCard := meta.NewIDCard("小测试", "110101201701011234")
	c.UpdateIDCard(idCard)
	assert.Equal(t, idCard, c.IDCard)

	// 更新基本信息
	gender := meta.GenderMale
	birthday := meta.NewBirthday("2017-01-01")
	c.UpdateProfile(gender, birthday)
	assert.Equal(t, gender, c.Gender)
	assert.Equal(t, birthday, c.Birthday)

	// 更新身高体重
	height, _ := meta.NewHeightFromFloat(110.0)
	weight, _ := meta.NewWeightFromFloat(20.0)
	c.UpdateHeightWeight(height, weight)
	assert.Equal(t, height, c.Height)
	assert.Equal(t, weight, c.Weight)

	// 重命名
	c.Rename("小测试改名")
	assert.Equal(t, "小测试改名", c.Name)
}
