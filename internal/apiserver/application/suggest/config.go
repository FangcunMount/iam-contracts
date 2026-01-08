package suggest

import (
	"strings"

	"github.com/spf13/viper"
)

// Config 控制 suggest 模块行为
type Config struct {
	Enable        bool
	DataDir       string
	FullSyncCron  string
	DeltaSyncCron string
	MaxResults    int
	KeyPadLen     int
	FullSQL       string
	DeltaSQL      string
	Snapshot      bool
}

// LoadConfig 从 viper 读取配置
func LoadConfig() Config {
	cfg := Config{
		MaxResults:    20,
		KeyPadLen:     25,
		FullSyncCron:  "@every 1h",
		DeltaSyncCron: "",
	}

	sub := viper.Sub("suggest")
	if sub == nil {
		return cfg
	}

	cfg.Enable = sub.GetBool("enable")
	cfg.DataDir = strings.TrimSpace(sub.GetString("data_dir"))
	if v := strings.TrimSpace(sub.GetString("full_sync_cron")); v != "" {
		cfg.FullSyncCron = v
	}
	cfg.DeltaSyncCron = strings.TrimSpace(sub.GetString("delta_sync_cron"))
	if v := sub.GetInt("max_results"); v > 0 {
		cfg.MaxResults = v
	}
	if v := sub.GetInt("key_pad_len"); v > 0 {
		cfg.KeyPadLen = v
	}
	cfg.FullSQL = sub.GetString("full_sql")
	cfg.DeltaSQL = sub.GetString("delta_sql")
	cfg.Snapshot = sub.GetBool("snapshot")
	if cfg.DataDir != "" && !sub.IsSet("snapshot") {
		cfg.Snapshot = true
	}

	return cfg
}

// ToUpdaterConfig 转换为 Updater 配置
func (c Config) ToUpdaterConfig() UpdaterConfig {
	return UpdaterConfig{
		FullCron:  c.FullSyncCron,
		DeltaCron: c.DeltaSyncCron,
		DataDir:   c.DataDir,
		Snapshot:  c.Snapshot,
	}
}
