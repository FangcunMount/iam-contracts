package main

import (
	"testing"

	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
)

func TestCollectSimulatedWechatUserAliasesFiltersBusinessUsers(t *testing.T) {
	aliases, err := collectSimulatedWechatUserAliases(map[string]string{
		"system":          "10001",
		"admin":           "110001",
		"content_manager": "110002",
		"boundary":        "100000",
	})
	if err != nil {
		t.Fatalf("collectSimulatedWechatUserAliases() error = %v", err)
	}

	if len(aliases) != 2 {
		t.Fatalf("collectSimulatedWechatUserAliases() len = %d, want 2", len(aliases))
	}
	if aliases[0] != "admin" || aliases[1] != "content_manager" {
		t.Fatalf("collectSimulatedWechatUserAliases() = %v, want [admin content_manager]", aliases)
	}
}

func TestConfiguredSimulatedWechatMiniProgramAppIDPrefersQuestionnaireNotebook(t *testing.T) {
	cfg := &SeedConfig{
		WechatApps: []WechatAppConfig{
			{Alias: "other_minip", AppID: "wx-other", Type: "MiniProgram"},
			{Alias: simulatedWechatAppConfigAlias, AppID: "wx-questionnaire", Type: "MiniProgram"},
		},
	}

	appID := configuredSimulatedWechatMiniProgramAppID(cfg)
	if appID != "wx-questionnaire" {
		t.Fatalf("configuredSimulatedWechatMiniProgramAppID() = %q, want wx-questionnaire", appID)
	}
}

func TestSimulatedWechatIdentityIsDeterministic(t *testing.T) {
	openID, unionID := simulatedWechatIdentity(meta.FromUint64(110001))

	if openID != "seed-wx-openid-110001" {
		t.Fatalf("openID = %q, want seed-wx-openid-110001", openID)
	}
	if unionID != "seed-wx-unionid-110001" {
		t.Fatalf("unionID = %q, want seed-wx-unionid-110001", unionID)
	}
}

func TestPreferredAuthnBackfillUserAliasPrefersExistingAlias(t *testing.T) {
	alias := preferredAuthnBackfillUserAlias(map[string]string{
		"admin": "110001",
	}, meta.FromUint64(110001))

	if alias != "admin" {
		t.Fatalf("preferredAuthnBackfillUserAlias() = %q, want admin", alias)
	}
}

func TestPreferredAuthnBackfillUserAliasFallsBackToSyntheticAlias(t *testing.T) {
	alias := preferredAuthnBackfillUserAlias(map[string]string{}, meta.FromUint64(110123))

	if alias != "user_110123" {
		t.Fatalf("preferredAuthnBackfillUserAlias() = %q, want user_110123", alias)
	}
}
