package policy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPolicyRuleAndGroupingRule(t *testing.T) {
	r := NewPolicyRule("sub", "dom", "obj", "act")
	assert.Equal(t, "sub", r.Sub)
	assert.Equal(t, "act", r.Act)

	g := NewGroupingRule("user1", "dom", "role1")
	assert.Equal(t, "role1", g.Role)
}

func TestPolicyVersionOptionsAndKeys(t *testing.T) {
	pv := NewPolicyVersion("t1", 5, WithChangedBy("alice"), WithReason("update"))
	assert.Equal(t, "t1", pv.TenantID)
	assert.Equal(t, int64(5), pv.Version)
	assert.Equal(t, "alice", pv.ChangedBy)
	assert.Equal(t, "update", pv.Reason)

	// keys
	assert.Contains(t, pv.RedisKey(), "authz:policy_version:")
	assert.Equal(t, "authz:policy_changed", pv.PubSubChannel())
}
