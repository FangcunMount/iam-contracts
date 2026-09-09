package policy

import (
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
)

// PolicyVersion 全局授权事实版本，用于变更发布及运行时同步。
type PolicyVersion struct {
	ID        PolicyVersionID // 版本记录标识
	Version   int64           // 全局授权事实版本号
	ChangedBy string          // 变更者标识
	Reason    string          // 变更原因
}

// NewPolicyVersion 创建新版本
func NewPolicyVersion(version int64, opts ...PolicyVersionOption) PolicyVersion {
	pv := PolicyVersion{

		Version: version,
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

// PolicyVersionID 策略版本ID值对象
type PolicyVersionID meta.ID

func (id PolicyVersionID) Uint64() uint64 {
	return meta.ID(id).Uint64()
}

func (id PolicyVersionID) String() string {
	return meta.ID(id).String()
}
