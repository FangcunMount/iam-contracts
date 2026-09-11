package options

import authzruntime "github.com/FangcunMount/iam/v5/internal/apiserver/infra/authz/runtime"

type AuthzOptions struct {
	PolicySync authzruntime.Config `mapstructure:"policy-sync" json:"policy_sync"`
}

func NewAuthzOptions() *AuthzOptions {
	return &AuthzOptions{PolicySync: authzruntime.DefaultConfig()}
}
