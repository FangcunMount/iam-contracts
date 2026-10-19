package policychange

import (
	"context"
	"slices"
	"strings"
	"sync"
)

type Permission struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
}
type permissionKey struct{}
type permissionCapture struct {
	mu       sync.Mutex
	entries  map[Permission]struct{}
	overflow bool
}

// CapturePermissions collects the original successful read-planning decisions,
// including dynamic protected-role rights. It cannot authorize an operation and
// does no I/O. Overflow is an error, never a truncated query-authorization policy.
func CapturePermissions(ctx context.Context) (context.Context, func() ([]Permission, error), error) {
	if ctx == nil {
		return nil, nil, ErrInvalidChangePlan
	}
	capture := &permissionCapture{entries: map[Permission]struct{}{}}
	return context.WithValue(ctx, permissionKey{}, capture), func() ([]Permission, error) {
		capture.mu.Lock()
		defer capture.mu.Unlock()
		if capture.overflow {
			return nil, ErrInvalidChangePlan
		}
		result := make([]Permission, 0, len(capture.entries))
		for p := range capture.entries {
			result = append(result, p)
		}
		slices.SortFunc(result, func(a, b Permission) int {
			if n := strings.Compare(a.Resource, b.Resource); n != 0 {
				return n
			}
			return strings.Compare(a.Action, b.Action)
		})
		return result, nil
	}, nil
}

// RecordPermission is called only after the existing authorization decision and
// its policy-version validation succeed. Outside read planning it is inert.
func RecordPermission(ctx context.Context, resource, action string) {
	if ctx == nil {
		return
	}
	capture, ok := ctx.Value(permissionKey{}).(*permissionCapture)
	if !ok {
		return
	}
	capture.mu.Lock()
	defer capture.mu.Unlock()
	permission := Permission{resource, action}
	if _, exists := capture.entries[permission]; exists {
		return
	}
	if len(capture.entries) >= 64 || len(resource) > 255 || len(action) > 255 {
		capture.overflow = true
		return
	}
	capture.entries[permission] = struct{}{}
}
