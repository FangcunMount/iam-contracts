package authz

import (
	"context"
	"strings"
	"testing"
)

func TestCheckObjectRejectsWithoutCallingTransport(t *testing.T) {
	c := &Client{}
	result, err := c.CheckObject(context.Background(), "user:1", "qs:evaluation:collection:assessments", "retry", "1", nil)
	if result != nil || err == nil || !strings.Contains(err.Error(), "retired") {
		t.Fatalf("result=%v err=%v", result, err)
	}
}
