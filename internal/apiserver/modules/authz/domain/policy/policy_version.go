package policy

import "github.com/fangcun-mount/iam-contracts/pkg/util/idutil"

// PolicyVersion 策略版本（用于缓存失效通知）
type PolicyVersion struct {
	ID        PolicyVersionID
	TenantID  string // 租户ID
	Version   int64  // 版本号
	ChangedBy string // 变更人
	Reason    string // 变更原因
}

// NewPolicyVersion 创建新版本
func NewPolicyVersion(tenantID string, version int64, opts ...PolicyVersionOption) PolicyVersion {
	pv := PolicyVersion{
		TenantID: tenantID,
		Version:  version,
	}
	for _, opt := range opts {
		opt(&pv)
	}
	return pv
}

// PolicyVersionOption 版本选项
type PolicyVersionOption func(*PolicyVersion)

func WithID(id PolicyVersionID) PolicyVersionOption { return func(pv *PolicyVersion) { pv.ID = id } }
func WithChangedBy(by string) PolicyVersionOption {
	return func(pv *PolicyVersion) { pv.ChangedBy = by }
}
func WithReason(reason string) PolicyVersionOption {
	return func(pv *PolicyVersion) { pv.Reason = reason }
}

// RedisKey 返回 Redis 中的版本键
func (pv *PolicyVersion) RedisKey() string {
	return "authz:policy_version:" + pv.TenantID
}

// PubSubChannel 返回发布订阅通道
func (pv *PolicyVersion) PubSubChannel() string {
	return "authz:policy_changed"
}

// PolicyVersionID 策略版本ID值对象
type PolicyVersionID idutil.ID

func NewPolicyVersionID(value uint64) PolicyVersionID {
	return PolicyVersionID(idutil.NewID(value))
}

func (id PolicyVersionID) Uint64() uint64 {
	return idutil.ID(id).Uint64()
}

func (id PolicyVersionID) String() string {
	return idutil.ID(id).String()
}
