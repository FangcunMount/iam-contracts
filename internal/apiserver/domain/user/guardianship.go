package user

import "time"

// Guardianship 监护关系
type Guardianship struct {
	ID            int64
	User          UserID
	Child         ChildID
	Rel           Relation
	EstablishedAt time.Time
	RevokedAt     *time.Time
}

// IsActive 是否有效
func (g *Guardianship) IsActive() bool { return g.RevokedAt == nil }

// Revoke 撤销监护关系
func (g *Guardianship) Revoke(at time.Time) { g.RevokedAt = &at }
