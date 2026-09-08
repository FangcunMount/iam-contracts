package policy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPolicyVersionOptions(t *testing.T) {
	pv := NewPolicyVersion(5, WithChangedBy("alice"), WithReason("update"))
	assert.Equal(t, int64(5), pv.Version)
	assert.Equal(t, "alice", pv.ChangedBy)
	assert.Equal(t, "update", pv.Reason)
}
