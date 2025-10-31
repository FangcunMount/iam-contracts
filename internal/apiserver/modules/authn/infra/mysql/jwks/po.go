package jwks

import (
	"time"
)

// KeyPO 密钥持久化对象，对应 jwks_keys 表
//
// 表结构说明：
// - id: 自增主键
// - kid: 密钥唯一标识符（Key ID），符合 JWK 规范
// - status: 密钥状态（1=Active, 2=Grace, 3=Retired）
// - kty: 密钥类型（Key Type），如 "RSA"、"EC"
// - use: 密钥用途（"sig"=签名，"enc"=加密）
// - alg: 算法标识（如 "RS256"、"RS384"、"RS512"）
// - jwk_json: 完整的公钥 JWK JSON（包含 n, e 等参数）
// - not_before: 密钥生效时间（NotBefore）
// - not_after: 密钥过期时间（NotAfter）
// - created_at: 创建时间（审计字段）
// - updated_at: 更新时间（审计字段）
// - version: 乐观锁版本号（审计字段）
type KeyPO struct {
	ID        uint64     `gorm:"primaryKey;autoIncrement"`
	Kid       string     `gorm:"column:kid;type:varchar(64);not null;uniqueIndex:idx_kid"`
	Status    int8       `gorm:"column:status;type:tinyint;not null;default:1;index:idx_status"`
	Kty       string     `gorm:"column:kty;type:varchar(32);not null"`
	Use       string     `gorm:"column:use;type:varchar(16);not null"`
	Alg       string     `gorm:"column:alg;type:varchar(32);not null;index:idx_alg"`
	JwkJSON   []byte     `gorm:"column:jwk_json;type:json;not null"`
	NotBefore *time.Time `gorm:"column:not_before;type:datetime"`
	NotAfter  *time.Time `gorm:"column:not_after;type:datetime;index:idx_not_after"`
	CreatedAt time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

// TableName 指定表名
func (KeyPO) TableName() string {
	return "iam_jwks_keys"
}
