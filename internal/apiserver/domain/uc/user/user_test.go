package user

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
)

func TestNewUser_Success(t *testing.T) {
	user, err := NewUser(
		"小明",
		meta.NewPhone("+8613012345678"),
		WithID(meta.NewID(10)),
		WithEmail(meta.NewEmail("test@example.com")),
		WithIDCard(meta.NewIDCard("tester", "123456")),
		WithStatus(UserInactive),
	)

	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, "小明", user.Name)
	assert.True(t, user.Phone.Equal(meta.NewPhone("+8613012345678")))
	assert.Equal(t, meta.NewEmail("test@example.com"), user.Email)
	assert.True(t, user.IDCard.Equal(meta.NewIDCard("tester", "123456")))
	assert.Equal(t, UserInactive, user.Status)
}

func TestNewUser_Validations(t *testing.T) {
	_, err := NewUser("", meta.NewPhone("+8613012345678"))
	require.Error(t, err)
	assert.Contains(t, fmt.Sprintf("%-v", err), "name cannot be empty")

	_, err = NewUser("name", meta.Phone{})
	require.Error(t, err)
	assert.Contains(t, fmt.Sprintf("%-v", err), "phone cannot be empty")
}

func TestUserLifecycleAndUpdates(t *testing.T) {
	user, err := NewUser("tester", meta.NewPhone("10086"))
	require.NoError(t, err)

	user.Deactivate()
	assert.True(t, user.IsInactive())

	user.Block()
	assert.True(t, user.IsBlocked())

	user.Activate()
	assert.True(t, user.IsUsable())

	user.Rename("new name")
	assert.Equal(t, "new name", user.Name)

	newPhone := meta.NewPhone("10010")
	user.UpdatePhone(newPhone)
	assert.True(t, user.Phone.Equal(newPhone))

	newEmail := meta.NewEmail("user@example.com")
	user.UpdateEmail(newEmail)
	assert.Equal(t, newEmail, user.Email)

	newIDCard := meta.NewIDCard("tester", "654321")
	user.UpdateIDCard(newIDCard)
	assert.True(t, user.IDCard.Equal(newIDCard))
}
