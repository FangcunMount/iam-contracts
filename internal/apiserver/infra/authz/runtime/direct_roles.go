package runtime

import (
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/authorization"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/subject"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	"sort"
)

type directRoleBuilder struct {
	roles map[subject.Ref]map[meta.ID]struct{}
}

func newDirectRoleBuilder() *directRoleBuilder {
	return &directRoleBuilder{roles: map[subject.Ref]map[meta.ID]struct{}{}}
}
func (b *directRoleBuilder) addAssignment(sub subject.Ref, id meta.ID) {
	if b.roles[sub] == nil {
		b.roles[sub] = map[meta.ID]struct{}{}
	}
	b.roles[sub][id] = struct{}{}
}
func (b *directRoleBuilder) build() *directRoles {
	r := &directRoles{roles: map[subject.Ref][]meta.ID{}}
	for sub, ids := range b.roles {
		values := make([]meta.ID, 0, len(ids))
		for id := range ids {
			values = append(values, id)
		}
		sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
		r.roles[sub] = values
	}
	return r
}

// directRoles owns immutable, deduplicated assignment facts. Multiple roles
// combine permissions; no job identities are inferred from other roles.
type directRoles struct{ roles map[subject.Ref][]meta.ID }

var _ authorization.RoleResolver = (*directRoles)(nil)

func (r *directRoles) DirectRoles(sub subject.Ref) ([]meta.ID, error) {
	result := make([]meta.ID, len(r.roles[sub]))
	copy(result, r.roles[sub])
	return result, nil
}

// EffectiveRoles retains the resolver contract while returning direct roles.
func (r *directRoles) EffectiveRoles(sub subject.Ref) ([]meta.ID, error) { return r.DirectRoles(sub) }
