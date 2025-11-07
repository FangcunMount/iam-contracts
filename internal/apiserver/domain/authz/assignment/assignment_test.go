package assignment

import (
	"testing"

	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
	"github.com/stretchr/testify/assert"
)

func TestAssignment_CreateAndKeys(t *testing.T) {
	a := NewAssignment(SubjectTypeUser, "u1", 42, "tenant", WithID(NewAssignmentID(5)), WithGrantedBy("admin"))
	assert.Equal(t, SubjectTypeUser, a.SubjectType)
	assert.Equal(t, "u1", a.SubjectID)
	assert.Equal(t, "admin", a.GrantedBy)
	assert.Equal(t, "user:u1", a.SubjectKey())
	// role key uses meta.NewID
	rk := a.RoleKey()
	// role: followed by numeric id
	assert.Contains(t, rk, "role:")
	id := meta.FromUint64(42)
	assert.Equal(t, id.String(), rk[len("role:"):])
}
