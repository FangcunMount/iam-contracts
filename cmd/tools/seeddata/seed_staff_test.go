package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/FangcunMount/component-base/pkg/log"
)

func TestDefaultStepsPlaceTenantBootstrapBeforeAdminInit(t *testing.T) {
	var tenantBootstrapIdx, adminIdx int = -1, -1
	for i, step := range defaultSteps {
		switch step {
		case stepTenantBootstrapAdmin:
			tenantBootstrapIdx = i
		case stepAdminInit:
			adminIdx = i
		}
	}

	if tenantBootstrapIdx == -1 || adminIdx == -1 {
		t.Fatalf("defaultSteps must contain tenant-bootstrap-admin and admin-init")
	}
	if tenantBootstrapIdx > adminIdx {
		t.Fatalf("tenant-bootstrap-admin must run before admin-init, got %d > %d", tenantBootstrapIdx, adminIdx)
	}
}

func TestSelectQSBootstrapPrincipalPrefersTenantBootstrapAdmin(t *testing.T) {
	cfg := &SeedConfig{
		Users: []UserConfig{
			{Alias: "system", OrgID: 1},
			{Alias: "admin", OrgID: 1, Email: "admin@fangcunmount.com"},
		},
		Accounts: []AccountConfig{
			{Alias: "system_account", UserAlias: "system", Provider: "operation", ExternalID: "system@fangcunmount.com", Password: "Admin@123"},
			{Alias: "admin_account", UserAlias: "admin", Provider: "operation", ExternalID: "admin@fangcunmount.com", Password: "Admin@123"},
		},
		TenantBootstrapAdmins: []TenantBootstrapAdminConfig{
			{
				TenantCode:          "fangcun",
				QSOrgID:             1,
				BootstrapQSOperator: true,
				BootstrapUser: UserConfig{
					Alias: "admin",
					Name:  "租户管理员",
					Email: "admin@fangcunmount.com",
				},
				BootstrapAccount: AccountConfig{
					Alias:      "admin_account",
					UserAlias:  "admin",
					Provider:   "operation",
					ExternalID: "admin@fangcunmount.com",
					Password:   "Admin@123",
				},
			},
		},
	}

	principal, err := selectQSBootstrapPrincipal(cfg, 1)
	if err != nil {
		t.Fatalf("selectQSBootstrapPrincipal() error = %v", err)
	}
	if principal.Source != "tenant_bootstrap_admins" {
		t.Fatalf("principal.Source = %q, want tenant_bootstrap_admins", principal.Source)
	}
	if principal.UserAlias != "admin" {
		t.Fatalf("principal.UserAlias = %q, want admin", principal.UserAlias)
	}
	if !shouldSkipQSStaffCreate(principal, UserConfig{Alias: "admin"}) {
		t.Fatalf("bootstrap admin should be skipped from /staff creation")
	}
}

func TestSelectQSBootstrapPrincipalFallsBackToQSAdminAssignment(t *testing.T) {
	cfg := &SeedConfig{
		Users: []UserConfig{
			{Alias: "system", OrgID: 1, Email: "system@fangcunmount.com"},
			{Alias: "content_manager", OrgID: 1, Email: "content_manager@fangcunmount.com"},
		},
		Accounts: []AccountConfig{
			{Alias: "system_account", UserAlias: "system", Provider: "operation", ExternalID: "system@fangcunmount.com", Password: "Admin@123"},
		},
		Roles: []RoleConfig{
			{Alias: "qs_admin", Name: "qs:admin", TenantID: "1"},
			{Alias: "qs_content_manager", Name: "qs:content_manager", TenantID: "1"},
		},
		Assignments: []AssignmentConfig{
			{SubjectType: "user", SubjectID: "@system", RoleAlias: "@qs_admin", TenantID: "1"},
			{SubjectType: "user", SubjectID: "@content_manager", RoleAlias: "@qs_content_manager", TenantID: "1"},
		},
	}

	principal, err := selectQSBootstrapPrincipal(cfg, 1)
	if err != nil {
		t.Fatalf("selectQSBootstrapPrincipal() error = %v", err)
	}
	if principal.Source != "qs_admin_assignment" {
		t.Fatalf("principal.Source = %q, want qs_admin_assignment", principal.Source)
	}
	if principal.UserAlias != "system" {
		t.Fatalf("principal.UserAlias = %q, want system", principal.UserAlias)
	}
	if shouldSkipQSStaffCreate(principal, UserConfig{Alias: "system"}) {
		t.Fatalf("fallback bootstrap principal should not be skipped automatically")
	}
}

func TestCreateStaffTreatsDuplicateAsSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":110002,"message":"User already exists"}`))
	}))
	defer server.Close()

	logger := log.New(log.NewOptions())
	err := createStaff(server.URL, "token", "123", UserConfig{
		Alias:    "system",
		Name:     "系统用户",
		OrgID:    1,
		Email:    "system@fangcunmount.com",
		IsActive: true,
	}, []string{"qs:admin"}, logger)
	if err != nil {
		t.Fatalf("createStaff() error = %v, want nil for duplicate user", err)
	}
}

func TestCreateStaffFailsOnNonDuplicateError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"code":110006,"message":"User is inactive"}`))
	}))
	defer server.Close()

	logger := log.New(log.NewOptions())
	err := createStaff(server.URL, "token", "123", UserConfig{
		Alias:    "system",
		Name:     "系统用户",
		OrgID:    1,
		Email:    "system@fangcunmount.com",
		IsActive: true,
	}, []string{"qs:admin"}, logger)
	if err == nil {
		t.Fatalf("createStaff() error = nil, want failure on non-duplicate error")
	}
}
