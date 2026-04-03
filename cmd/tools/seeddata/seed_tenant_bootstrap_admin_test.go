package main

import "testing"

func TestBuildTenantBootstrapTemplatesAndDesiredState(t *testing.T) {
	cfg := &SeedConfig{
		Roles: []RoleConfig{
			{Name: "tenant_admin", TenantID: "fangcun"},
			{Name: "user", TenantID: "fangcun"},
			{Name: "qs:admin", TenantID: "1"},
			{Name: "qs:content_manager", TenantID: "1"},
		},
		Policies: []PolicyConfig{
			{Type: "p", Subject: "role:tenant_admin", Values: []string{"fangcun", "iam:users", "read|create"}},
			{Type: "g", Subject: "role:tenant_admin", Values: []string{"role:user", "fangcun"}},
			{Type: "p", Subject: "role:qs:admin", Values: []string{"1", "qs:*", ".*"}},
			{Type: "g", Subject: "role:qs:admin", Values: []string{"role:qs:content_manager", "1"}},
		},
	}

	templates, err := buildTenantBootstrapTemplates(cfg)
	if err != nil {
		t.Fatalf("buildTenantBootstrapTemplates() error = %v", err)
	}
	if templates.iamSourceDomain != "fangcun" {
		t.Fatalf("iamSourceDomain = %q, want fangcun", templates.iamSourceDomain)
	}
	if templates.qsSourceDomain != "1" {
		t.Fatalf("qsSourceDomain = %q, want 1", templates.qsSourceDomain)
	}

	desired, err := buildTenantBootstrapDesiredState(templates, TenantBootstrapAdminConfig{
		TenantCode: "acme",
		QSOrgID:    7,
	})
	if err != nil {
		t.Fatalf("buildTenantBootstrapDesiredState() error = %v", err)
	}

	if _, ok := desired.Policies["role:tenant_admin\x1facme\x1fiam:users\x1fread|create"]; !ok {
		t.Fatalf("expected tenant_admin policy to be cloned into acme domain")
	}
	if _, ok := desired.Groupings["role:tenant_admin\x1frole:user\x1facme"]; !ok {
		t.Fatalf("expected tenant_admin->user grouping to be cloned into acme domain")
	}
	if _, ok := desired.Policies["role:qs:admin\x1f7\x1fqs:*\x1f.*"]; !ok {
		t.Fatalf("expected qs admin policy to be cloned into org domain 7")
	}
	if _, ok := desired.Groupings["role:qs:admin\x1frole:qs:content_manager\x1f7"]; !ok {
		t.Fatalf("expected qs grouping to be cloned into org domain 7")
	}
}

func TestValidateTenantBootstrapAdminConfig(t *testing.T) {
	templates := &tenantBootstrapTemplates{
		iamRoleNames: map[string]struct{}{"tenant_admin": {}, "user": {}},
		qsRoleNames:  map[string]struct{}{"qs:admin": {}, "qs:content_manager": {}},
	}

	err := validateTenantBootstrapAdminConfig(TenantBootstrapAdminConfig{
		TenantCode: "acme",
		QSOrgID:    1,
		BootstrapUser: UserConfig{
			Alias: "acme_admin",
			ID:    200001,
			Name:  "Acme Admin",
		},
		BootstrapAccount: AccountConfig{
			Alias:      "acme_admin_account",
			Provider:   "operation",
			ExternalID: "admin@acme.example",
			Password:   "Admin@123",
		},
		Grants: TenantBootstrapGrantConfig{
			IAMRoles: []string{"tenant_admin"},
			QSRoles:  []string{"qs:admin"},
		},
	}, templates)
	if err != nil {
		t.Fatalf("validateTenantBootstrapAdminConfig() error = %v", err)
	}

	err = validateTenantBootstrapAdminConfig(TenantBootstrapAdminConfig{
		TenantCode: "acme",
		QSOrgID:    1,
		BootstrapUser: UserConfig{
			Alias: "acme_admin",
			ID:    200001,
			Name:  "Acme Admin",
		},
		BootstrapAccount: AccountConfig{
			Alias:      "acme_admin_account",
			Provider:   "operation",
			ExternalID: "admin@acme.example",
			Password:   "Admin@123",
		},
		Grants: TenantBootstrapGrantConfig{
			IAMRoles: []string{"super_admin"},
		},
	}, templates)
	if err == nil {
		t.Fatalf("expected unsupported iam grant role validation error")
	}
}
