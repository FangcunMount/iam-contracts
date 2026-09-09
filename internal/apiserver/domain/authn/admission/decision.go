package admission

// Outcome 表示登录准入的最终结果。
type Outcome string

const (
	OutcomeAdmitted Outcome = "admitted"
	OutcomeDenied   Outcome = "denied"
)

// DenialReason 表示认证主体被拒绝建立或维持登录态的领域原因。
type DenialReason string

const (
	ReasonLoginIdentityMissing  DenialReason = "login_identity_missing"
	ReasonLoginIdentityDisabled DenialReason = "login_identity_disabled"
	ReasonIdentityOwnerMismatch DenialReason = "identity_owner_mismatch"
	ReasonUserMissing           DenialReason = "user_missing"
	ReasonUserBlocked           DenialReason = "user_blocked"
	ReasonUserInactive          DenialReason = "user_inactive"
)

// Decision 表示 AdmissionPolicy 对认证主体身份组合作出的领域判定。
type Decision struct {
	Subject Subject      // 本次准入检查的用户与登录身份组合
	Outcome Outcome      // 是否允许建立或维持认证状态
	Reason  DenialReason // 拒绝准入的业务原因，允许时为空
}

// Admit 构造允许建立或维持登录态的判定。
func Admit(subject Subject) Decision {
	return Decision{Subject: subject, Outcome: OutcomeAdmitted}
}

// Deny 构造拒绝建立或维持登录态的判定。
func Deny(subject Subject, reason DenialReason) Decision {
	return Decision{Subject: subject, Outcome: OutcomeDenied, Reason: reason}
}

// IsAdmitted 返回当前认证主体身份组合是否允许建立或维持登录态。
func (d Decision) IsAdmitted() bool {
	return d.Outcome == OutcomeAdmitted
}
