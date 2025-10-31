package common

import (
	"context"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	redis "github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

const (
	dsnEnvKey   = "IAM_SEEDER_DSN"
	redisEnvKey = "IAM_SEEDER_REDIS"
)

// ResolveDSN returns the DSN to use, preferring an explicit value and
// falling back to the IAM_SEEDER_DSN environment variable.
func ResolveDSN(explicit string) string {
	if explicit != "" {
		return explicit
	}

	if env := os.Getenv(dsnEnvKey); env != "" {
		return env
	}

	log.Fatalf("mysql dsn is required (use --dsn flag or set %s)", dsnEnvKey)
	return ""
}

// ResolveRedisAddr resolves the redis address from flag or environment.
func ResolveRedisAddr(explicit string) string {
	if explicit != "" {
		return explicit
	}
	if env := os.Getenv(redisEnvKey); env != "" {
		return env
	}
	return ""
}

// MustOpenGORM opens a GORM MySQL connection and verifies it.
func MustOpenGORM(dsn string) *gorm.DB {
	cfg := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
		DisableAutomaticPing: false,
	}

	db, err := gorm.Open(mysql.Open(dsn), cfg)
	if err != nil {
		log.Fatalf("failed to open gorm connection: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("failed to get sql DB from gorm: %v", err)
	}

	sqlDB.SetConnMaxIdleTime(30 * time.Second)
	sqlDB.SetConnMaxLifetime(10 * time.Minute)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(50)

	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("failed to ping mysql: %v", err)
	}

	return db
}

// CloseGORM closes the underlying sql.DB of a gorm instance.
func CloseGORM(db *gorm.DB) {
	if db == nil {
		return
	}
	sqlDB, err := db.DB()
	if err != nil {
		return
	}
	_ = sqlDB.Close()
}

// MustOpenRedis creates a Redis client if address is provided.
func MustOpenRedis(addr string) *redis.Client {
	if addr == "" {
		return nil
	}

	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		PoolTimeout:  30 * time.Second,
		MinIdleConns: 10,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("failed to ping redis: %v", err)
	}

	return client
}
