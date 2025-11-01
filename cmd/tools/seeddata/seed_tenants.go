package main

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm/clause"
)

// ==================== 租户相关类型定义 ====================

// tenantRecord 租户种子数据
type tenantRecord struct {
	ID           string
	Name         string
	Code         string
	ContactName  string
	ContactPhone string
	ContactEmail string
	Status       string
	MaxUsers     int
	MaxRoles     int
}

// tenantPO 租户持久化对象
type tenantPO struct {
	ID           string `gorm:"primaryKey;column:id"`
	Name         string
	Code         string
	ContactName  string
	ContactPhone string
	ContactEmail string
	Status       string
	MaxUsers     int
	MaxRoles     int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// TableName 指定表名
func (tenantPO) TableName() string {
	return "iam_tenants"
}

// ==================== 租户 Seed 函数 ====================

// seedTenants 创建租户数据
//
// 业务说明：
// - 租户是系统的顶层隔离单位
// - 从配置文件读取租户数据
// - 使用 UPSERT 策略，避免重复执行时报错
//
// 幂等性：使用 ON CONFLICT UPDATE 策略，可以安全地重复执行
func seedTenants(ctx context.Context, deps *dependencies) error {
	config := deps.Config
	if config == nil || len(config.Tenants) == 0 {
		deps.Logger.Warnw("⚠️  配置文件中没有租户数据，跳过")
		return nil
	}

	deps.Logger.Infow("📋 开始创建租户数据...", "count", len(config.Tenants))

	// 从配置读取租户
	var tenants []tenantRecord
	for _, tc := range config.Tenants {
		tenants = append(tenants, tenantRecord{
			ID:           tc.Code, // 使用 code 作为 ID
			Name:         tc.Name,
			Code:         tc.Code,
			Status:       "active",
			MaxUsers:     100000,
			MaxRoles:     1000,
			ContactName:  "系统管理员",
			ContactPhone: "10086000000",
			ContactEmail: "support@example.com",
		})
	}

	for _, t := range tenants {
		po := tenantPO{
			ID:           t.ID,
			Name:         t.Name,
			Code:         t.Code,
			ContactName:  t.ContactName,
			ContactPhone: t.ContactPhone,
			ContactEmail: t.ContactEmail,
			Status:       t.Status,
			MaxUsers:     t.MaxUsers,
			MaxRoles:     t.MaxRoles,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		// 使用 UPSERT 策略：如果存在则更新，不存在则插入
		if err := deps.DB.WithContext(ctx).
			Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "id"}},
				UpdateAll: true,
			}).
			Create(&po).Error; err != nil {
			return fmt.Errorf("upsert tenant %s: %w", t.ID, err)
		}
	}

	deps.Logger.Infow("✅ 租户数据已创建", "count", len(tenants))
	return nil
}
