package options

import (
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestResourceAudienceConfiguration(t *testing.T) {
	opts := NewOptions()
	require.Equal(t, "iam-api", opts.Auth.ResourceAudience)
	require.Contains(t, opts.Auth.AccessTokenAudience, "iam-api")
	for _, aud := range []string{"", "unissued-recipient"} {
		opts.Auth.ResourceAudience = aud
		errs := opts.Validate()
		found := false
		for _, err := range errs {
			if strings.Contains(err.Error(), "auth.resource_audience") {
				found = true
			}
		}
		require.True(t, found)
	}
}
