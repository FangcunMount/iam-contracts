// Historical migration fixture only. Runtime roles never use this graph.
package migration

import (
	"fmt"
	"sort"

	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/role"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
)

const legacyMaxHierarchyDepth = 32 // 最大继承路径节点数，包含直接分配的角色

type legacyRoleNode struct {
	ManagementProtection role.ManagementProtection // 管理保护属性
	ID                   meta.ID                   // 角色ID
}

// validateLegacyGraph 校验角色引用、管理保护、环和继承深度。
// 深度按角色节点数计算，包含直接分配的角色；供原子写入、快照构建及预检复用。
// roles 为角色节点，edges 为继承关系。
func validateLegacyGraph(roles []legacyRoleNode, edges []*legacyInheritance) error {
	protections := make(map[meta.ID]role.ManagementProtection, len(roles)) // 管理保护属性映射
	identities := make(map[meta.ID]struct{}, len(roles))                   // 角色ID映射
	indegree := make(map[meta.ID]int, len(roles))                          // 入度映射
	depth := make(map[meta.ID]int, len(roles))                             // 深度映射
	previous := make(map[meta.ID]meta.ID)                                  // 前一个角色ID映射
	graph := make(map[meta.ID][]meta.ID)                                   // 角色继承图
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
			if depth[parent] > legacyMaxHierarchyDepth {
				path := []meta.ID{parent}
				for n := previous[parent]; !n.IsZero(); n = previous[n] {
					path = append(path, n)
				}
				for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
					path[i], path[j] = path[j], path[i]
				}
				return invalidGraph("role inheritance exceeds %d nodes: %v", legacyMaxHierarchyDepth, path)
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

type legacyInheritance struct{ RoleID, InheritedRoleID meta.ID }

func (*legacyInheritance) IsActive() bool { return true }
