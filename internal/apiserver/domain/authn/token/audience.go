package token

import (
	"strings"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
)

// NormalizeExpectedAudience validates caller-supplied recipient constraints, never token data.
func NormalizeExpectedAudience(values []string) ([]string, error) {
	out, err := normalizeAudience(values)
	if err != nil {
		audienceFailures.WithLabelValues("missing_or_invalid").Inc()
	}
	return out, err
}
func normalizeAudience(values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "expected_audience is required")
	}
	out := make([]string, 0, len(values))
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, perrors.WithCode(code.ErrInvalidArgument, "expected_audience entries must be non-empty")
		}
		if !seen[value] {
			out = append(out, value)
			seen[value] = true
		}
	}
	return out, nil
}
func matchesAudience(actual, expected []string) bool {
	for _, want := range expected {
		for _, got := range actual {
			if want == got {
				return true
			}
		}
	}
	return false
}
