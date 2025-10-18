package account

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewWeChatAccount 测试创建微信账号
func TestNewWeChatAccount(t *testing.T) {
	// Arrange
	accountID := NewAccountID(123)
	appID := "wx1234567890abcdef"
	openID := "openid-xyz-123"

	// Act
	wechat := NewWeChatAccount(accountID, appID, openID)

	// Assert
	assert.Equal(t, accountID, wechat.AccountID)
	assert.Equal(t, appID, wechat.AppID)
	assert.Equal(t, openID, wechat.OpenID)
	assert.Nil(t, wechat.UnionID)
	assert.Nil(t, wechat.Nickname)
	assert.Nil(t, wechat.AvatarURL)
	assert.Nil(t, wechat.Meta)
}

// TestNewWeChatAccount_WithOptions 测试创建微信账号（带选项）
func TestNewWeChatAccount_WithOptions(t *testing.T) {
	// Arrange
	accountID := NewAccountID(456)
	appID := "wx1234567890abcdef"
	openID := "openid-abc-456"
	unionID := "unionid-xyz-789"
	nickname := "张三"
	avatarURL := "https://example.com/avatar.jpg"
	meta := []byte(`{"country":"CN","province":"Guangdong","city":"Shenzhen"}`)

	// Act
	wechat := NewWeChatAccount(
		accountID,
		appID,
		openID,
		WithWeChatUnionID(unionID),
		WithWeChatNickname(nickname),
		WithWeChatAvatarURL(avatarURL),
		WithWeChatMeta(meta),
	)

	// Assert
	assert.Equal(t, accountID, wechat.AccountID)
	assert.NotNil(t, wechat.UnionID)
	assert.Equal(t, unionID, *wechat.UnionID)
	assert.NotNil(t, wechat.Nickname)
	assert.Equal(t, nickname, *wechat.Nickname)
	assert.NotNil(t, wechat.AvatarURL)
	assert.Equal(t, avatarURL, *wechat.AvatarURL)
	assert.Equal(t, meta, wechat.Meta)
}

// TestWeChatAccount_UpdateMeta 测试更新 Meta 信息
func TestWeChatAccount_UpdateMeta(t *testing.T) {
	// Arrange
	wechat := NewWeChatAccount(NewAccountID(123), "wx123", "openid-123")
	newMeta := []byte(`{"country":"US","language":"en"}`)

	// Act
	wechat.UpdateMeta(newMeta)

	// Assert
	assert.Equal(t, newMeta, wechat.Meta)
}

// TestWeChatAccount_UpdateNickname 测试更新昵称
func TestWeChatAccount_UpdateNickname(t *testing.T) {
	// Arrange
	wechat := NewWeChatAccount(NewAccountID(123), "wx123", "openid-123")
	assert.Nil(t, wechat.Nickname, "初始昵称应为空")

	// Act
	newNickname := "李四"
	wechat.UpdateNickname(newNickname)

	// Assert
	assert.NotNil(t, wechat.Nickname)
	assert.Equal(t, newNickname, *wechat.Nickname)

	// Update again
	anotherNickname := "王五"
	wechat.UpdateNickname(anotherNickname)
	assert.Equal(t, anotherNickname, *wechat.Nickname)
}

// TestWeChatAccount_UpdateAvatarURL 测试更新头像 URL
func TestWeChatAccount_UpdateAvatarURL(t *testing.T) {
	// Arrange
	wechat := NewWeChatAccount(NewAccountID(123), "wx123", "openid-123")
	assert.Nil(t, wechat.AvatarURL, "初始头像 URL 应为空")

	// Act
	newAvatarURL := "https://example.com/new-avatar.png"
	wechat.UpdateAvatarURL(newAvatarURL)

	// Assert
	assert.NotNil(t, wechat.AvatarURL)
	assert.Equal(t, newAvatarURL, *wechat.AvatarURL)

	// Update again
	anotherAvatarURL := "https://example.com/another-avatar.png"
	wechat.UpdateAvatarURL(anotherAvatarURL)
	assert.Equal(t, anotherAvatarURL, *wechat.AvatarURL)
}
