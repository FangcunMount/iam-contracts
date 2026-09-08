package runtime

import (
	"sort"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/authorization"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/subject"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
)

type roleIDSet map[meta.ID]struct{}

// roleGraphBuilder owns the mutable construction phase. build converts every
// set into a stable slice so the published roleGraph is read-only.
type roleGraphBuilder struct {
	directRoles    map[subject.Ref]roleIDSet
	inheritedRoles map[meta.ID]roleIDSet
}

func newRoleGraphBuilder() *roleGraphBuilder {
	return &roleGraphBuilder{directRoles: make(map[subject.Ref]roleIDSet), inheritedRoles: make(map[meta.ID]roleIDSet)}
}
func (b *roleGraphBuilder) addAssignment(sub subject.Ref, id meta.ID) {
	if b.directRoles[sub] == nil {
		b.directRoles[sub] = make(roleIDSet)
	}
	b.directRoles[sub][id] = struct{}{}
}
func (b *roleGraphBuilder) addInheritance(child, parent meta.ID) {
	if b.inheritedRoles[child] == nil {
		b.inheritedRoles[child] = make(roleIDSet)
	}
	b.inheritedRoles[child][parent] = struct{}{}
}
func (b *roleGraphBuilder) build(maxHierarchyLevel int) *roleGraph {
	g := &roleGraph{maxHierarchyLevel: maxHierarchyLevel, directRoles: make(map[subject.Ref][]meta.ID), inheritedRoles: make(map[meta.ID][]meta.ID)}
	for sub, ids := range b.directRoles {
		g.directRoles[sub] = sortedRoleIDs(ids)
	}
	for child, parents := range b.inheritedRoles {
		g.inheritedRoles[child] = sortedRoleIDs(parents)
	}
	return g
}

// roleGraph is the immutable runtime projection of Subject -> Role and
// Role -> inherited Role facts for one published authorization snapshot.
type roleGraph struct {
	maxHierarchyLevel int
	directRoles       map[subject.Ref][]meta.ID
	inheritedRoles    map[meta.ID][]meta.ID
}

var _ authorization.RoleResolver = (*roleGraph)(nil)

func (g *roleGraph) DirectRoles(sub subject.Ref) ([]meta.ID, error) {
	if g == nil {
		return nil, perrors.WithCode(code.ErrInternalServerError, "authorization role resolver is unavailable")
	}
	return cloneRoleIDs(g.directRoles[sub]), nil
}

func (g *roleGraph) EffectiveRoles(sub subject.Ref) ([]meta.ID, error) {
	if g == nil {
		return nil, perrors.WithCode(code.ErrInternalServerError, "authorization role resolver is unavailable")
	}
	if g.maxHierarchyLevel <= 0 {
		return make([]meta.ID, 0), nil
	}

	frontier := g.directRoles[sub]
	seen := make(roleIDSet)
	effective := make([]meta.ID, 0, len(frontier))
	for level := 0; level < g.maxHierarchyLevel && len(frontier) > 0; level++ {
		next := make(roleIDSet)
		for _, current := range frontier {
			if _, exists := seen[current]; exists {
				continue
			}
			seen[current] = struct{}{}
			effective = append(effective, current)
		}
		for _, current := range frontier {
			for _, parent := range g.inheritedRoles[current] {
				if _, exists := seen[parent]; !exists {
					next[parent] = struct{}{}
				}
			}
		}
		frontier = sortedRoleIDs(next)
	}
	if len(frontier) > 0 {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "role hierarchy exceeds maximum depth")
	}
	sort.Slice(effective, func(i, j int) bool {
		return effective[i] < effective[j]
	})
	return effective, nil
}

func sortedRoleIDs(values roleIDSet) []meta.ID {
	result := make([]meta.ID, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i] < result[j]
	})
	return result
}

func cloneRoleIDs(values []meta.ID) []meta.ID {
	result := make([]meta.ID, len(values))
	copy(result, values)
	return result
}
