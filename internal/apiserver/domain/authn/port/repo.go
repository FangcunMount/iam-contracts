package port

import (
	"context"
	"time"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/authn/account"
)

// 标准仓储错误（由 infra 层返回，应用/领域可据此分支）
var (
	ErrNotFound = errorString("not found")
	ErrConflict = errorString("conflict") // 唯一键冲突/并发插入
)

type errorString string

func (e errorString) Error() string { return string(e) }

// AccountRepo —— 聚合根（统一锚点）
// 唯一性：UNIQUE(provider, app_id, external_id)
type AccountRepo interface {
	FindByRef(ctx context.Context, provider account.Provider, externalID string, appID *string) (*account.Account, error)
	Create(ctx context.Context, a *account.Account) error
	Disable(ctx context.Context, id string) error
}

// WeChatRepo —— 子实体（微信画像/专有字段）
type WeChatRepo interface {
	// Upsert：按 (app_id, openid) 幂等写入/更新画像
	Upsert(ctx context.Context, wx *account.WeChatAccount) error
	// FindByAppOpenID：便于外部身份直查
	FindByAppOpenID(ctx context.Context, appID, openid string) (*account.WeChatAccount, error)
}

// OperationRepo —— 子实体（用户名/口令）
type OperationRepo interface {
	GetByUsername(ctx context.Context, username string) (*account.OperationCredential, error)
	Create(ctx context.Context, cred *account.OperationCredential) error
	UpdateHash(ctx context.Context, username string, hash []byte, algo string, params []byte) error

	// 失败次数累计与临时锁定（应在单条 SQL/事务内完成）
	IncFailAndMaybeLock(ctx context.Context, username string, maxFail int, lockFor time.Duration) (newFailCount int, err error)
	ResetFailures(ctx context.Context, username string) error
}
