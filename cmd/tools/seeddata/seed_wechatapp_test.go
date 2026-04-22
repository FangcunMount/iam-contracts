package main

import (
	"encoding/base64"
	"testing"
)

func TestResolveWechatAppEncryptionKeyAllowsEmptyKeyWhenNoSecrets(t *testing.T) {
	cfg := &SeedConfig{
		EncryptionKey: "",
		WechatApps: []WechatAppConfig{
			{Alias: "questionnaire_notebook", AppID: "wx-app", Type: "MiniProgram", AppSecret: ""},
		},
	}

	key, err := resolveWechatAppEncryptionKey(cfg)
	if err != nil {
		t.Fatalf("resolveWechatAppEncryptionKey() error = %v, want nil", err)
	}
	if key != nil {
		t.Fatalf("resolveWechatAppEncryptionKey() key = %v, want nil", key)
	}
}

func TestResolveWechatAppEncryptionKeyRejectsSecretWithoutKey(t *testing.T) {
	cfg := &SeedConfig{
		EncryptionKey: "",
		WechatApps: []WechatAppConfig{
			{Alias: "questionnaire_notebook", AppID: "wx-app", Type: "MiniProgram", AppSecret: "abcdefghijklmnop"},
		},
	}

	if _, err := resolveWechatAppEncryptionKey(cfg); err == nil {
		t.Fatalf("resolveWechatAppEncryptionKey() error = nil, want failure")
	}
}

func TestResolveWechatAppEncryptionKeyAcceptsBase64Key(t *testing.T) {
	raw := "0123456789abcdef0123456789abcdef"
	cfg := &SeedConfig{
		EncryptionKey: base64.StdEncoding.EncodeToString([]byte(raw)),
	}

	key, err := resolveWechatAppEncryptionKey(cfg)
	if err != nil {
		t.Fatalf("resolveWechatAppEncryptionKey() error = %v, want nil", err)
	}
	if string(key) != raw {
		t.Fatalf("resolveWechatAppEncryptionKey() key = %q, want %q", string(key), raw)
	}
}

func TestResolveWechatAppEncryptionKeyAcceptsRaw32ByteKey(t *testing.T) {
	raw := "0123456789abcdef0123456789abcdef"
	cfg := &SeedConfig{
		EncryptionKey: raw,
	}

	key, err := resolveWechatAppEncryptionKey(cfg)
	if err != nil {
		t.Fatalf("resolveWechatAppEncryptionKey() error = %v, want nil", err)
	}
	if string(key) != raw {
		t.Fatalf("resolveWechatAppEncryptionKey() key = %q, want %q", string(key), raw)
	}
}
