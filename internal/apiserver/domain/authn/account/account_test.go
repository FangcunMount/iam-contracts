package account

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
)

func TestNewAccount_DefaultsAndOptions(t *testing.T) {
	acc := NewAccount(meta.FromUint64(1), AccountType("wc-minip"), ExternalID("ext-1"))
	require.NotNil(t, acc)
	assert.Equal(t, StatusActive, acc.Status)
	assert.Equal(t, ExternalID("ext-1"), acc.ExternalID)

	// options
	a2 := NewAccount(meta.FromUint64(2), AccountType("wc-offi"), ExternalID("e2"), WithUnionID(UnionID("u2")), WithAppID(AppId("appid")))
	assert.Equal(t, UnionID("u2"), a2.UniqueID)
	assert.Equal(t, AppId("appid"), a2.AppID)
}

func TestStatusQueriesAndTransitions(t *testing.T) {
	acc := NewAccount(meta.FromUint64(10), AccountType("t"), ExternalID("e"))
	// initial
	assert.True(t, acc.IsActive())
	assert.False(t, acc.IsDisabled())

	acc.Disable()
	assert.True(t, acc.IsDisabled())

	acc.Activate()
	assert.True(t, acc.IsActive())

	acc.Archive()
	assert.True(t, acc.IsArchived())

	acc.Delete()
	assert.True(t, acc.IsDeleted())
}

func TestSetUniqueID_Idempotency(t *testing.T) {
	acc := NewAccount(meta.FromUint64(3), AccountType("t"), ExternalID("e"))
	ok := acc.SetUniqueID(UnionID("u1"))
	assert.True(t, ok)
	assert.Equal(t, UnionID("u1"), acc.UniqueID)

	// second set should fail
	ok2 := acc.SetUniqueID(UnionID("u2"))
	assert.False(t, ok2)
	assert.Equal(t, UnionID("u1"), acc.UniqueID)
}

func TestProfileAndMetaMergeAndFields(t *testing.T) {
	acc := NewAccount(meta.FromUint64(4), AccountType("t"), ExternalID("e"))
	acc.UpdateProfile(map[string]string{"nick": "n1"})
	v, ok := acc.GetProfileField("nick")
	assert.True(t, ok)
	assert.Equal(t, "n1", v)

	acc.UpdateProfile(map[string]string{"avatar": "a1"})
	v2, ok2 := acc.GetProfileField("avatar")
	assert.True(t, ok2)
	assert.Equal(t, "a1", v2)

	acc.SetProfileField("nick", "n2")
	v3, _ := acc.GetProfileField("nick")
	assert.Equal(t, "n2", v3)

	// meta
	acc.UpdateMeta(map[string]string{"k": "v"})
	mv, mok := acc.GetMetaField("k")
	assert.True(t, mok)
	assert.Equal(t, "v", mv)

	acc.SetMetaField("k", "v2")
	mv2, _ := acc.GetMetaField("k")
	assert.Equal(t, "v2", mv2)
}

func TestCanTransitionToRules(t *testing.T) {
	acc := NewAccount(meta.FromUint64(5), AccountType("t"), ExternalID("e"))

	// Active -> Disabled, Archived, Deleted
	assert.True(t, acc.CanTransitionTo(StatusDisabled))
	assert.True(t, acc.CanTransitionTo(StatusArchived))
	assert.True(t, acc.CanTransitionTo(StatusDeleted))

	// Disabled -> Active
	acc.Disable()
	assert.True(t, acc.CanTransitionTo(StatusActive))

	// Archived -> Active
	acc.Archive()
	assert.True(t, acc.CanTransitionTo(StatusActive))

	// Deleted is terminal
	acc.Delete()
	assert.False(t, acc.CanTransitionTo(StatusActive))
	assert.True(t, acc.CanTransitionTo(StatusDeleted))
}

func TestCanUpdateWhenDeleted(t *testing.T) {
	acc := NewAccount(meta.FromUint64(6), AccountType("t"), ExternalID("e"))
	assert.True(t, acc.CanUpdateProfile())
	assert.True(t, acc.CanUpdateMeta())

	acc.Delete()
	assert.False(t, acc.CanUpdateProfile())
	assert.False(t, acc.CanUpdateMeta())
}
