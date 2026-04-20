package assembler

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAuthnModuleInitializeWithRedisAdapters(t *testing.T) {
	t.Setenv("TZ", "Asia/Shanghai")
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.Set("jwks.auto_init", false)
	viper.Set("migration.autoseed", false)
	viper.Set("app.mode", "test")
	viper.Set("sms.provider", "log")
	viper.Set("sms.login_otp_ttl", "5m")
	viper.Set("sms.login_otp_send_cooldown", "1m")
	viper.Set("sms.login_otp_code_length", 6)

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open() error = %v", err)
	}

	mr := miniredis.RunT(t)
	redisClient := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = redisClient.Close()
	})

	module := NewAuthnModule()
	if err := module.Initialize(db, redisClient); err != nil {
		t.Fatalf("AuthnModule.Initialize() error = %v", err)
	}

	if module.LoginPreparationService == nil {
		t.Fatalf("expected LoginPreparationService to be initialized")
	}
	if module.TokenService == nil {
		t.Fatalf("expected TokenService to be initialized")
	}
}

func TestIDPModuleInitializeWithRedisAdapters(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open() error = %v", err)
	}

	mr := miniredis.RunT(t)
	redisClient := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = redisClient.Close()
	})

	module := NewIDPModule()
	if err := module.Initialize(db, redisClient, []byte("0123456789abcdef0123456789abcdef")); err != nil {
		t.Fatalf("IDPModule.Initialize() error = %v", err)
	}

	if module.WechatAppTokenService == nil {
		t.Fatalf("expected WechatAppTokenService to be initialized")
	}
	if module.WechatAppHandler == nil {
		t.Fatalf("expected WechatAppHandler to be initialized")
	}
}
