package cache

// FamilyPolicy 描述某个缓存族的静态策略。
type FamilyPolicy struct {
	TTLSource                      string
	WriteMode                      string
	InvalidationMode               string
	HasInternalRefreshCoordination bool
}

// FamilyDescriptor 描述一个缓存族的治理元数据。
type FamilyDescriptor struct {
	Family          Family
	Backend         BackendKind
	RedisType       RedisDataType
	Codec           ValueCodecKind
	Role            DataRole
	OwnerModule     string
	KeyPattern      string
	TTLSource       string
	SelectionReason string
	Policy          FamilyPolicy
	Capabilities    []GovernanceCapability
}
