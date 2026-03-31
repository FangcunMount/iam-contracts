package main

import "testing"

func TestAccountOperaExternalID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		account   AccountConfig
		userEmail string
		expect    string
	}{
		{
			name:      "prefer external id",
			account:   AccountConfig{ExternalID: "system@fangcunmount.com"},
			userEmail: "system@fangcunmount.com",
			expect:    "system@fangcunmount.com",
		},
		{
			name:      "fallback to legacy username",
			account:   AccountConfig{Username: "legacy-admin"},
			userEmail: "admin@fangcunmount.com",
			expect:    "legacy-admin",
		},
		{
			name:      "fallback to email",
			account:   AccountConfig{},
			userEmail: "content_manager@fangcunmount.com",
			expect:    "content_manager@fangcunmount.com",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := accountOperaExternalID(tc.account, tc.userEmail); got != tc.expect {
				t.Fatalf("accountOperaExternalID() = %q, want %q", got, tc.expect)
			}
		})
	}
}

func TestResolveAdminLoginUsesFirstOperationAccount(t *testing.T) {
	t.Parallel()

	cfg := &SeedConfig{
		Users: []UserConfig{
			{Alias: "system", Email: "system@fangcunmount.com"},
			{Alias: "admin", Email: "admin@fangcunmount.com"},
		},
		Accounts: []AccountConfig{
			{
				Alias:      "system_account",
				UserAlias:  "system",
				Provider:   "operation",
				ExternalID: "system@fangcunmount.com",
				Password:   "Admin@123",
			},
			{
				Alias:      "admin_account",
				UserAlias:  "admin",
				Provider:   "operation",
				ExternalID: "admin@fangcunmount.com",
				Password:   "Admin@123",
			},
		},
	}

	loginID, password := resolveAdminLogin(cfg)
	if loginID != "system@fangcunmount.com" {
		t.Fatalf("resolveAdminLogin() loginID = %q, want %q", loginID, "system@fangcunmount.com")
	}
	if password != "Admin@123" {
		t.Fatalf("resolveAdminLogin() password = %q, want %q", password, "Admin@123")
	}
}

func TestConfiguredAccountStatus(t *testing.T) {
	t.Parallel()

	active := 1
	disabled := 0
	invalid := 9

	tests := []struct {
		name      string
		account   AccountConfig
		expectOK  bool
		expect    int
		expectErr bool
	}{
		{name: "unset", account: AccountConfig{}, expectOK: false},
		{name: "active", account: AccountConfig{Status: &active}, expectOK: true, expect: 1},
		{name: "disabled", account: AccountConfig{Status: &disabled}, expectOK: true, expect: 0},
		{name: "invalid", account: AccountConfig{Status: &invalid}, expectErr: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			status, ok, err := configuredAccountStatus(tc.account)
			if tc.expectErr {
				if err == nil {
					t.Fatalf("configuredAccountStatus() expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("configuredAccountStatus() unexpected error: %v", err)
			}
			if ok != tc.expectOK {
				t.Fatalf("configuredAccountStatus() ok = %v, want %v", ok, tc.expectOK)
			}
			if ok && int(status) != tc.expect {
				t.Fatalf("configuredAccountStatus() status = %d, want %d", status, tc.expect)
			}
		})
	}
}

func TestValidateOperationAccountConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		account   AccountConfig
		expectErr bool
	}{
		{name: "empty app id", account: AccountConfig{}},
		{name: "explicit opera app id", account: AccountConfig{AppID: "opera"}},
		{name: "invalid app id", account: AccountConfig{AppID: "tenant-app"}, expectErr: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateOperationAccountConfig(tc.account)
			if tc.expectErr && err == nil {
				t.Fatalf("validateOperationAccountConfig() expected error")
			}
			if !tc.expectErr && err != nil {
				t.Fatalf("validateOperationAccountConfig() unexpected error: %v", err)
			}
		})
	}
}
