package policychange

import (
	"context"
	"strings"
	"unicode/utf8"
)

type requestKey struct{}

func WithRequestKey(ctx context.Context, key string) (context.Context, error) {
	if ctx == nil || key == "" || len(key) > 255 || !utf8.ValidString(key) || strings.TrimSpace(key) != key {
		return nil, ErrInvalidChangePlan
	}
	return context.WithValue(ctx, requestKey{}, key), nil
}
func RequestKey(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	key, ok := ctx.Value(requestKey{}).(string)
	return key, ok && key != ""
}
