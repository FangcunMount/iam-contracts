package cache

// RedisDataType 表示治理面视角下的 Redis 数据结构。
type RedisDataType string

const (
	RedisDataTypeNone   RedisDataType = "none"
	RedisDataTypeString RedisDataType = "string"
	RedisDataTypeHash   RedisDataType = "hash"
	RedisDataTypeSet    RedisDataType = "set"
	RedisDataTypeZSet   RedisDataType = "zset"
)

// ValueCodecKind 表示 family 的 value 编码方式。
type ValueCodecKind string

const (
	ValueCodecKindMemoryObject ValueCodecKind = "memory_object"
	ValueCodecKindJSON         ValueCodecKind = "json"
	ValueCodecKindMarker       ValueCodecKind = "marker"
	ValueCodecKindString       ValueCodecKind = "string"
	ValueCodecKindLeaseToken   ValueCodecKind = "lease_token"
)
