package guardianship

import (
	"time"

	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/child"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/modules/uc/domain/user"
)

type Relation string // 监护关系
const (
	RelSelf         Relation = "self"         // 自己
	RelParent       Relation = "parent"       // 父母
	RelGrandparents Relation = "grandparents" // 祖父母
	RelOther        Relation = "other"        // 其他
)

// Guardianship 监护关系
type Guardianship struct {
	ID            int64
	User          user.UserID
	Child         child.ChildID
	Rel           Relation
	EstablishedAt time.Time
	RevokedAt     *time.Time
}

// IsActive 是否有效
func (g *Guardianship) IsActive() bool { return g.RevokedAt == nil }

// Revoke 撤销监护关系
func (g *Guardianship) Revoke(at time.Time) { g.RevokedAt = &at }
