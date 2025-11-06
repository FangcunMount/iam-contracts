package account

import "github.com/FangcunMount/iam-contracts/internal/pkg/meta"

// ==================== 数据传输对象（DTO）====================
// 用于跨层传递数据，避免直接暴露领域对象

// CreateAccountDTO 创建账号数据传输对象
type CreateAccountDTO struct {
	UserID      meta.ID     // 用户ID
	AccountType AccountType // 账号类型
	ExternalID  ExternalID  // 外部平台用户标识
	AppID       AppId       // 应用ID
}

// UpdateProfileDTO 更新资料数据传输对象
type UpdateProfileDTO struct {
	AccountID meta.ID           // 账号ID
	Profile   map[string]string // 用户资料
}

// UpdateMetaDTO 更新元数据数据传输对象
type UpdateMetaDTO struct {
	AccountID meta.ID           // 账号ID
	Meta      map[string]string // 元数据
}

// SetUniqueIDDTO 设置唯一标识数据传输对象
type SetUniqueIDDTO struct {
	AccountID meta.ID // 账号ID
	UniqueID  UnionID // 全局唯一标识
}
