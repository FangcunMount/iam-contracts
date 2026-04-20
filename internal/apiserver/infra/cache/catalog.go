package cache

var inspectOnly = []GovernanceCapability{GovernanceCapabilityInspect}

var catalog = []FamilyDescriptor{
	{
		Family:          FamilyAuthnRefreshToken,
		Backend:         BackendKindRedis,
		RedisType:       RedisDataTypeString,
		Codec:           ValueCodecKindJSON,
		Role:            DataRoleAuthoritativeState,
		OwnerModule:     "authn",
		KeyPattern:      "refresh_token:{tokenValue}",
		TTLSource:       "token.RemainingDuration()",
		SelectionReason: "单 token 单对象、整对象读写、key 级 TTL。",
		Policy: FamilyPolicy{
			TTLSource:                      "token.RemainingDuration()",
			WriteMode:                      "整体写入",
			InvalidationMode:               "TTL 到期或显式删除",
			HasInternalRefreshCoordination: false,
		},
		Capabilities: inspectOnly,
	},
	{
		Family:          FamilyAuthnRevokedAccessToken,
		Backend:         BackendKindRedis,
		RedisType:       RedisDataTypeString,
		Codec:           ValueCodecKindMarker,
		Role:            DataRoleMarkerState,
		OwnerModule:     "authn",
		KeyPattern:      "revoked_access_token:{tokenID}",
		TTLSource:       "调用方传入 expiry",
		SelectionReason: "只关心存在性，且逐 token 独立 TTL。",
		Policy: FamilyPolicy{
			TTLSource:                      "调用方传入 expiry",
			WriteMode:                      "marker 写入",
			InvalidationMode:               "TTL 到期",
			HasInternalRefreshCoordination: false,
		},
		Capabilities: inspectOnly,
	},
	{
		Family:          FamilyAuthnSession,
		Backend:         BackendKindRedis,
		RedisType:       RedisDataTypeString,
		Codec:           ValueCodecKindJSON,
		Role:            DataRoleAuthoritativeState,
		OwnerModule:     "authn",
		KeyPattern:      "session:{sid}",
		TTLSource:       "session.ExpiresAt",
		SelectionReason: "会话主对象按 sid 独立寻址，整对象读写且保留 key 级 TTL。",
		Policy: FamilyPolicy{
			TTLSource:                      "session.ExpiresAt",
			WriteMode:                      "整体写入",
			InvalidationMode:               "TTL 到期或主动撤销后保留到自然过期",
			HasInternalRefreshCoordination: false,
		},
		Capabilities: inspectOnly,
	},
	{
		Family:          FamilyAuthnUserSessionIndex,
		Backend:         BackendKindRedis,
		RedisType:       RedisDataTypeZSet,
		Codec:           ValueCodecKindString,
		Role:            DataRoleAuthoritativeState,
		OwnerModule:     "authn",
		KeyPattern:      "user_session_index:{userID}",
		TTLSource:       "不设置独立 TTL，成员 score 取会话过期时间",
		SelectionReason: "需要按用户批量回收和列举活跃会话，ZSet 适合按过期时间懒清理。",
		Policy: FamilyPolicy{
			TTLSource:                      "成员 score 取会话过期时间",
			WriteMode:                      "ZADD sid -> expiresAtUnix",
			InvalidationMode:               "撤销时移除成员，读取前懒清理过期成员",
			HasInternalRefreshCoordination: false,
		},
		Capabilities: inspectOnly,
	},
	{
		Family:          FamilyAuthnAccountSessionIndex,
		Backend:         BackendKindRedis,
		RedisType:       RedisDataTypeZSet,
		Codec:           ValueCodecKindString,
		Role:            DataRoleAuthoritativeState,
		OwnerModule:     "authn",
		KeyPattern:      "account_session_index:{accountID}",
		TTLSource:       "不设置独立 TTL，成员 score 取会话过期时间",
		SelectionReason: "需要按账号批量回收和列举活跃会话，ZSet 适合按过期时间懒清理。",
		Policy: FamilyPolicy{
			TTLSource:                      "成员 score 取会话过期时间",
			WriteMode:                      "ZADD sid -> expiresAtUnix",
			InvalidationMode:               "撤销时移除成员，读取前懒清理过期成员",
			HasInternalRefreshCoordination: false,
		},
		Capabilities: inspectOnly,
	},
	{
		Family:          FamilyAuthnLoginOTP,
		Backend:         BackendKindRedis,
		RedisType:       RedisDataTypeString,
		Codec:           ValueCodecKindMarker,
		Role:            DataRoleMarkerState,
		OwnerModule:     "authn",
		KeyPattern:      "otp:{scene}:{phoneE164}:{code}",
		TTLSource:       "OTP 有效期",
		SelectionReason: "一次性存在性语义，适合单 key String 和原子消费。",
		Policy: FamilyPolicy{
			TTLSource:                      "OTP 有效期",
			WriteMode:                      "marker 写入 + 原子消费",
			InvalidationMode:               "消费删除或 TTL 到期",
			HasInternalRefreshCoordination: false,
		},
		Capabilities: inspectOnly,
	},
	{
		Family:          FamilyAuthnLoginOTPSendGate,
		Backend:         BackendKindRedis,
		RedisType:       RedisDataTypeString,
		Codec:           ValueCodecKindMarker,
		Role:            DataRoleMarkerState,
		OwnerModule:     "authn",
		KeyPattern:      "otp:sendgate:{scene}:{phoneE164}",
		TTLSource:       "发送冷却时间",
		SelectionReason: "本质是 cooldown 占位 key。",
		Policy: FamilyPolicy{
			TTLSource:                      "发送冷却时间",
			WriteMode:                      "SET NX EX",
			InvalidationMode:               "TTL 到期",
			HasInternalRefreshCoordination: false,
		},
		Capabilities: inspectOnly,
	},
	{
		Family:          FamilyIDPWechatAccessToken,
		Backend:         BackendKindRedis,
		RedisType:       RedisDataTypeString,
		Codec:           ValueCodecKindJSON,
		Role:            DataRoleRemoteTokenCache,
		OwnerModule:     "idp",
		KeyPattern:      "idp:wechat:token:{appID}",
		TTLSource:       "应用层计算后传入",
		SelectionReason: "单 app 单对象缓存，整体读写。",
		Policy: FamilyPolicy{
			TTLSource:                      "应用层计算后传入",
			WriteMode:                      "整体写入",
			InvalidationMode:               "TTL 到期或刷新覆盖",
			HasInternalRefreshCoordination: true,
		},
		Capabilities: inspectOnly,
	},
	{
		Family:          FamilyIDPWechatSDK,
		Backend:         BackendKindRedis,
		RedisType:       RedisDataTypeString,
		Codec:           ValueCodecKindString,
		Role:            DataRoleRemoteTokenCache,
		OwnerModule:     "idp",
		KeyPattern:      "由微信 SDK 调用方提供",
		TTLSource:       "调用方传入",
		SelectionReason: "当前缓存值就是字符串 token 或 ticket。",
		Policy: FamilyPolicy{
			TTLSource:                      "调用方传入",
			WriteMode:                      "字符串整体写入",
			InvalidationMode:               "TTL 到期或显式删除",
			HasInternalRefreshCoordination: false,
		},
		Capabilities: inspectOnly,
	},
	{
		Family:          FamilyAuthnJWKSPublishSnapshot,
		Backend:         BackendKindMemory,
		RedisType:       RedisDataTypeNone,
		Codec:           ValueCodecKindMemoryObject,
		Role:            DataRoleDerivedSnapshot,
		OwnerModule:     "authn",
		KeyPattern:      "进程内快照，无 Redis key",
		TTLSource:       "基于最后构建时间的内存复用窗口",
		SelectionReason: "当前是单进程派生发布快照，尚无跨实例共享需求。",
		Policy: FamilyPolicy{
			TTLSource:                      "BuildJWKS 内部 1 分钟内复用",
			WriteMode:                      "重建快照并覆盖内存字段",
			InvalidationMode:               "重建刷新",
			HasInternalRefreshCoordination: false,
		},
		Capabilities: inspectOnly,
	},
}

// Families 返回当前 IAM 缓存目录快照。
func Families() []FamilyDescriptor {
	descriptors := make([]FamilyDescriptor, len(catalog))
	copy(descriptors, catalog)
	return descriptors
}

// GetFamily 返回指定 family 的静态描述。
func GetFamily(family Family) (FamilyDescriptor, bool) {
	for _, descriptor := range catalog {
		if descriptor.Family == family {
			return descriptor, true
		}
	}
	return FamilyDescriptor{}, false
}
