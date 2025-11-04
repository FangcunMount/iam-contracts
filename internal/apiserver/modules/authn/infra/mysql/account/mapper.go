package account

import (
	"encoding/json"
	"time"

	"github.com/FangcunMount/component-base/pkg/util/idutil"
	domain "github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
)

// Mapper 负责领域模型与持久化对象之间的转换。
type Mapper struct{}

// NewMapper 创建新的映射器实例。
func NewMapper() *Mapper {
	return &Mapper{}
}

// ==================== Account 映射 ====================

// ToAccountPO 将账号领域模型转换为持久化对象。
func (m *Mapper) ToAccountPO(acc *domain.Account) *AccountPO {
	if acc == nil {
		return nil
	}

	po := &AccountPO{
		UserID:     idutil.NewID(acc.UserID.ToUint64()),
		Type:       string(acc.Type),
		ExternalID: string(acc.ExternalID),
		Profile:    mapToJSON(acc.Profile),
		Meta:       mapToJSON(acc.Meta),
		Status:     int8(acc.Status),
	}

	// 设置 ID（如果已存在）
	if acc.ID.ToUint64() != 0 {
		po.ID = idutil.NewID(acc.ID.ToUint64())
	}

	// 设置 AppID
	if acc.AppID != "" {
		appIDStr := string(acc.AppID)
		po.AppID = &appIDStr
	}

	// 设置 UniqueID
	if acc.UniqueID != "" {
		uniqueIDStr := string(acc.UniqueID)
		po.UniqueID = &uniqueIDStr
	}

	return po
}

// ToAccountDO 将持久化对象转换为账号领域模型。
func (m *Mapper) ToAccountDO(po *AccountPO) *domain.Account {
	if po == nil {
		return nil
	}

	acc := &domain.Account{
		ID:         meta.NewID(po.ID.Uint64()),
		UserID:     meta.NewID(po.UserID.Uint64()),
		Type:       domain.AccountType(po.Type),
		ExternalID: domain.ExternalID(po.ExternalID),
		Profile:    jsonToMap(po.Profile),
		Meta:       jsonToMap(po.Meta),
		Status:     domain.AccountStatus(po.Status),
	}

	// 设置 AppID
	if po.AppID != nil {
		acc.AppID = domain.AppId(*po.AppID)
	}

	// 设置 UniqueID
	if po.UniqueID != nil {
		acc.UniqueID = domain.UnionID(*po.UniqueID)
	}

	return acc
}

// ==================== Credential 映射 ====================

// ToCredentialPO 将凭据领域模型转换为持久化对象。
func (m *Mapper) ToCredentialPO(cred *domain.Credential) *CredentialPO {
	if cred == nil {
		return nil
	}

	// 根据凭据字段推断类型
	credType := inferCredentialType(cred)

	po := &CredentialPO{
		AccountID:      cred.AccountID,
		Type:           string(credType),
		IDP:            copyStringPtr(cred.IDP),
		IDPIdentifier:  cred.IDPIdentifier,
		AppID:          copyStringPtr(cred.AppID),
		Material:       cloneBytes(cred.Material),
		Algo:           copyStringPtr(cred.Algo),
		Params:         cloneBytes(cred.ParamsJSON),
		Status:         int8(cred.Status),
		FailedAttempts: cred.FailedAttempts,
		LockedUntil:    copyTimePtr(cred.LockedUntil),
		LastSuccessAt:  copyTimePtr(cred.LastSuccessAt),
		LastFailureAt:  copyTimePtr(cred.LastFailureAt),
		Rev:            cred.Rev,
	}

	// 设置 ID（如果已存在）
	if cred.ID != 0 {
		po.ID = idutil.NewID(uint64(cred.ID))
	}

	return po
}

// inferCredentialType 根据 Credential 字段推断类型
func inferCredentialType(cred *domain.Credential) domain.CredentialType {
	// 密码类型：有 Material、Algo，没有 IDP
	if cred.IDP == nil && len(cred.Material) > 0 && cred.Algo != nil {
		return domain.CredPassword
	}

	// 手机 OTP：IDP 为 "phone"
	if cred.IDP != nil && *cred.IDP == "phone" {
		return domain.CredPhoneOTP
	}

	// WeChat OAuth：IDP 为 "wechat"
	if cred.IDP != nil && *cred.IDP == "wechat" {
		return domain.CredOAuthWxMinip
	}

	// WeCom OAuth：IDP 为 "wecom"
	if cred.IDP != nil && *cred.IDP == "wecom" {
		return domain.CredOAuthWecom
	}

	// 默认返回密码类型
	return domain.CredPassword
}

// ToCredentialDO 将持久化对象转换为凭据领域模型。
func (m *Mapper) ToCredentialDO(po *CredentialPO) *domain.Credential {
	if po == nil {
		return nil
	}

	return &domain.Credential{
		ID:             int64(po.ID.Uint64()),
		AccountID:      po.AccountID,
		IDP:            po.IDP,
		IDPIdentifier:  po.IDPIdentifier,
		AppID:          po.AppID,
		Material:       cloneBytes(po.Material),
		Algo:           po.Algo,
		ParamsJSON:     cloneBytes(po.Params),
		Status:         domain.CredentialStatus(po.Status),
		FailedAttempts: po.FailedAttempts,
		LockedUntil:    copyTimePtr(po.LockedUntil),
		LastSuccessAt:  copyTimePtr(po.LastSuccessAt),
		LastFailureAt:  copyTimePtr(po.LastFailureAt),
		Rev:            po.Rev,
	}
}

// ==================== 辅助方法 ====================

// cloneBytes 复制字节切片。
func cloneBytes(src []byte) []byte {
	if len(src) == 0 {
		return nil
	}
	dst := make([]byte, len(src))
	copy(dst, src)
	return dst
}

// copyStringPtr 复制字符串指针。
func copyStringPtr(src *string) *string {
	if src == nil {
		return nil
	}
	s := *src
	return &s
}

// copyTimePtr 复制时间指针。
func copyTimePtr(src *time.Time) *time.Time {
	if src == nil {
		return nil
	}
	t := *src
	return &t
}

// mapToJSON 将 map 转换为 JSON 字节数组。
func mapToJSON(m map[string]string) []byte {
	if len(m) == 0 {
		return nil
	}
	data, _ := json.Marshal(m)
	return data
}

// jsonToMap 将 JSON 字节数组转换为 map。
func jsonToMap(data []byte) map[string]string {
	if len(data) == 0 {
		return make(map[string]string)
	}
	var m map[string]string
	_ = json.Unmarshal(data, &m)
	if m == nil {
		return make(map[string]string)
	}
	return m
}
