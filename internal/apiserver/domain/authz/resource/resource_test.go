package resource

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResourceAndActions(t *testing.T) {
	r := NewResource("scale:form:*", []string{"read", "write"}, WithDisplayName("Form"), WithAppName("scale"))
	assert.Equal(t, "scale:form:*", r.Key)
	assert.True(t, r.HasAction("read"))
	assert.False(t, r.HasAction("delete"))
}
