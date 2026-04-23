package options

import (
	"github.com/spf13/pflag"
)

// MigrationOptions 数据库迁移配置选项
type MigrationOptions struct {
	Enabled  bool   `json:"enabled"  mapstructure:"enabled"`  // 是否启用自动迁移
	Database string `json:"database" mapstructure:"database"` // 数据库名称
}

// NewMigrationOptions 创建默认的迁移选项
func NewMigrationOptions() *MigrationOptions {
	return &MigrationOptions{
		Enabled:  true, // 默认启用自动迁移
		Database: "iam",
	}
}

// Validate 验证选项
func (o *MigrationOptions) Validate() []error {
	return nil
}

// AddFlags 添加命令行参数
func (o *MigrationOptions) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&o.Enabled, "migration.enabled", o.Enabled,
		"Enable automatic database migration on startup. "+
			"Migration uses version control to ensure each version runs only once.")

	fs.StringVar(&o.Database, "migration.database", o.Database,
		"Database name for migration.")
}
