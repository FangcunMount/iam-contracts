package roleinheritance

import (
	"fmt"
	"sort"

	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/role"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
)

const MaxHierarchyDepth = 32

type RoleNode struct {
	ManagementProtection role.ManagementProtection
	ID                   meta.ID
}

// ValidateGraph is shared by atomic writes, runtime compilation and preflight.
// Depth counts role nodes, including the directly assigned role.
func ValidateGraph(roles []RoleNode, edges []*Inheritance) error {
	protections := make(map[meta.ID]role.ManagementProtection, len(roles))
	identities := make(map[meta.ID]struct{}, len(roles))
	indegree := make(map[meta.ID]int, len(roles))
	depth := make(map[meta.ID]int, len(roles))
	previous := make(map[meta.ID]meta.ID)
	graph := make(map[meta.ID][]meta.ID)
	for _, node := range roles {
		if node.ID.IsZero() {
			return invalidGraph("invalid role node %s", node.ID)
		}
		if _, exists := identities[node.ID]; exists {
			return invalidGraph("duplicate role %s", node.ID)
		}
		identities[node.ID] = struct{}{}
		protections[node.ID] = node.ManagementProtection
		indegree[node.ID] = 0
		depth[node.ID] = 1
	}
	for _, edge := range edges {
		if edge == nil {
			return invalidGraph("nil inheritance")
		}
		if !edge.IsActive() {
			continue
		}
		_, childOK := identities[edge.RoleID]
		_, parentOK := identities[edge.InheritedRoleID]
		if !childOK || !parentOK {
			return invalidGraph("unknown role in edge %s -> %s", edge.RoleID, edge.InheritedRoleID)
		}
		if protections[edge.RoleID] != role.ManagementProtected && protections[edge.InheritedRoleID] == role.ManagementProtected {
			return invalidGraph("普通角色不能继承受保护角色")
		}
		graph[edge.RoleID] = append(graph[edge.RoleID], edge.InheritedRoleID)
		indegree[edge.InheritedRoleID]++
	}
	queue := make([]meta.ID, 0, len(roles))
	for id, count := range indegree {
		if count == 0 {
			queue = append(queue, id)
		}
	}
	sort.Slice(queue, func(i, j int) bool { return queue[i] < queue[j] })
	for index := 0; index < len(queue); index++ {
		child := queue[index]
		for _, parent := range graph[child] {
			if depth[child]+1 > depth[parent] {
				depth[parent] = depth[child] + 1
				previous[parent] = child
			}
			if depth[parent] > MaxHierarchyDepth {
				path := []meta.ID{parent}
				for n := previous[parent]; !n.IsZero(); n = previous[n] {
					path = append(path, n)
				}
				for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
					path[i], path[j] = path[j], path[i]
				}
				return invalidGraph("role inheritance exceeds %d nodes: %v", MaxHierarchyDepth, path)
			}
			indegree[parent]--
			if indegree[parent] == 0 {
				queue = append(queue, parent)
			}
		}
	}
	if len(queue) != len(roles) {
		cyclic := make([]meta.ID, 0)
		for id, count := range indegree {
			if count > 0 {
				cyclic = append(cyclic, id)
			}
		}
		sort.Slice(cyclic, func(i, j int) bool { return cyclic[i] < cyclic[j] })
		return invalidGraph("role inheritance contains a cycle involving %v", cyclic)
	}
	return nil
}
func invalidGraph(format string, args ...any) error {
	return perrors.WithCode(code.ErrInvalidArgument, "%s", fmt.Sprintf(format, args...))
}
