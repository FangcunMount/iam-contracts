package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	casbinv2 "github.com/casbin/casbin/v2"

	assignmentApp "github.com/FangcunMount/iam-contracts/internal/apiserver/application/authz/assignment"
	resourceApp "github.com/FangcunMount/iam-contracts/internal/apiserver/application/authz/resource"
	assignmentDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authz/assignment"
	policyDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authz/policy"
	resourceDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authz/resource"
	casbininfra "github.com/FangcunMount/iam-contracts/internal/apiserver/infra/casbin"
	assignmentMysql "github.com/FangcunMount/iam-contracts/internal/apiserver/infra/mysql/assignment"
	policyMysql "github.com/FangcunMount/iam-contracts/internal/apiserver/infra/mysql/policy"
	resourceMysql "github.com/FangcunMount/iam-contracts/internal/apiserver/infra/mysql/resource"
	roleMysql "github.com/FangcunMount/iam-contracts/internal/apiserver/infra/mysql/role"
)

const (
	managedIAMResourcePrefix = "iam:"
	managedQSResourcePrefix  = "qs:"
	seedAuthzChangedBy       = "seeddata"
	seedAuthzVersionReason   = "seeddata authz sync"
	seedResourceListLimit    = 10_000
)

type assignmentSubjectKey struct {
	SubjectType assignmentDomain.SubjectType
	SubjectID   string
	TenantID    string
}

type desiredAssignment struct {
	RoleID    uint64
	GrantedBy string
}

type desiredPolicyState struct {
	Policies       map[string]policyDomain.PolicyRule
	Groupings      map[string]policyDomain.GroupingRule
	ManagedRoleKey map[string]struct{}
	ManagedTenant  map[string]struct{}
}

type casbinEnforcerProvider interface {
	Enforcer() *casbinv2.CachedEnforcer
}

// ==================== 授权资源 Seed 函数 ====================

// seedAuthzResources 创建授权资源数据
//
// 业务说明：
// - 创建或更新系统基础资源定义
// - 每个资源包含允许的操作列表
// - 资源用于后续的权限策略配置
//
// 幂等性：按资源 key 做 upsert，并清理不在配置中的受管资源
func seedAuthzResources(ctx context.Context, deps *dependencies, state *seedContext) error {
	config := deps.Config
	if config == nil || len(config.Resources) == 0 {
		deps.Logger.Warnw("⚠️  配置文件中没有资源数据，跳过")
		return nil
	}

	resourceRepo := resourceMysql.NewResourceRepository(deps.DB)
	resourceValidator := resourceDomain.NewValidator(resourceRepo)
	resourceCommander := resourceApp.NewResourceCommandService(resourceValidator, resourceRepo)

	existingResources, _, err := resourceRepo.List(ctx, resourceDomain.ListResourcesQuery{
		Offset: 0,
		Limit:  seedResourceListLimit,
	})
	if err != nil {
		return fmt.Errorf("list resources: %w", err)
	}

	existingByKey := make(map[string]*resourceDomain.Resource, len(existingResources))
	for _, resource := range existingResources {
		existingByKey[resource.Key] = resource
	}

	desiredByKey := make(map[string]ResourceConfig, len(config.Resources))

	var createdCount, updatedCount, deletedCount int

	for _, rc := range config.Resources {
		desiredByKey[rc.Key] = rc

		existing := existingByKey[rc.Key]
		if existing == nil {
			created, err := resourceCommander.CreateResource(ctx, resourceDomain.CreateResourceCommand{
				Key:         rc.Key,
				DisplayName: rc.DisplayName,
				AppName:     rc.AppName,
				Domain:      rc.Domain,
				Type:        rc.Type,
				Actions:     rc.Actions,
				Description: rc.Description,
			})
			if err != nil {
				return fmt.Errorf("create resource %s: %w", rc.Key, err)
			}

			state.Resources[rc.Alias] = created.ID.Uint64()
			createdCount++
			continue
		}

		if resourceNeedsUpdate(existing, rc) {
			existing.DisplayName = rc.DisplayName
			existing.AppName = rc.AppName
			existing.Domain = rc.Domain
			existing.Type = rc.Type
			existing.Actions = append([]string(nil), rc.Actions...)
			existing.Description = rc.Description

			if err := resourceRepo.Update(ctx, existing); err != nil {
				return fmt.Errorf("update resource %s: %w", rc.Key, err)
			}
			updatedCount++
		}

		state.Resources[rc.Alias] = existing.ID.Uint64()
	}

	for key, existing := range existingByKey {
		if !isManagedResourceKey(key) {
			continue
		}
		if _, ok := desiredByKey[key]; ok {
			continue
		}
		if err := resourceRepo.Delete(ctx, existing.ID); err != nil {
			return fmt.Errorf("delete stale resource %s: %w", key, err)
		}
		deletedCount++
	}

	deps.Logger.Infow("✅ 授权资源数据已同步",
		"configured", len(config.Resources),
		"created", createdCount,
		"updated", updatedCount,
		"deleted", deletedCount,
	)
	return nil
}

// ==================== 角色分配 Seed 函数 ====================

// seedRoleAssignments 创建角色分配数据
//
// 业务说明：
// - 为用户分配系统角色
// - 角色决定用户在系统中的权限
// - 同时在 Casbin 中添加用户-角色分组规则
//
// 前置条件：必须先创建用户和资源
// 幂等性：对配置中的主体做精确同步
func seedRoleAssignments(ctx context.Context, deps *dependencies, state *seedContext) error {
	config := deps.Config
	if config == nil {
		deps.Logger.Warnw("⚠️  没有可用配置，跳过角色分配")
		return nil
	}

	modelPath := deps.CasbinModel
	if _, err := os.Stat(modelPath); err != nil {
		return fmt.Errorf("casbin model file not found: %w", err)
	}

	casbinPort, err := casbininfra.NewCasbinAdapter(deps.DB, modelPath)
	if err != nil {
		return fmt.Errorf("init casbin adapter: %w", err)
	}

	roleRepo := roleMysql.NewRoleRepository(deps.DB)
	assignmentRepo := assignmentMysql.NewAssignmentRepository(deps.DB)
	policyVersionRepo := policyMysql.NewPolicyVersionRepository(deps.DB)
	assignmentManager := assignmentDomain.NewValidator(assignmentRepo, roleRepo)
	assignmentCommander := assignmentApp.NewAssignmentCommandService(
		assignmentManager,
		assignmentRepo,
		roleRepo,
		casbinPort,
		policyVersionRepo,
		nil,
	)
	assignmentQueryer := assignmentApp.NewAssignmentQueryService(assignmentManager, assignmentRepo)

	managedRoleIDs, err := managedRoleIDsFromState(state)
	if err != nil {
		return err
	}

	managedTenants := collectAssignmentTenants(config)
	desiredAssignmentsBySubject := make(map[assignmentSubjectKey]map[uint64]desiredAssignment)
	subjectsToSync := make(map[assignmentSubjectKey]struct{})

	for _, userConfig := range config.Users {
		userID, ok := state.Users[userConfig.Alias]
		if !ok {
			continue
		}
		for tenantID := range managedTenants {
			subjectsToSync[assignmentSubjectKey{
				SubjectType: assignmentDomain.SubjectTypeUser,
				SubjectID:   userID,
				TenantID:    tenantID,
			}] = struct{}{}
		}
	}

	for _, ac := range config.Assignments {
		subjectType, err := parseAssignmentSubjectType(ac.SubjectType)
		if err != nil {
			return fmt.Errorf("parse assignment subject type: %w", err)
		}

		subjectID := resolveUserAlias(ac.SubjectID, state)
		roleID, err := resolveRoleAlias(ac.RoleID, ac.RoleAlias, state)
		if err != nil {
			return fmt.Errorf("resolve role for subject %s: %w", ac.SubjectID, err)
		}

		key := assignmentSubjectKey{
			SubjectType: subjectType,
			SubjectID:   subjectID,
			TenantID:    strings.TrimSpace(ac.TenantID),
		}
		subjectsToSync[key] = struct{}{}
		if desiredAssignmentsBySubject[key] == nil {
			desiredAssignmentsBySubject[key] = make(map[uint64]desiredAssignment)
		}

		grantedBy := strings.TrimSpace(ac.GrantedBy)
		if grantedBy == "" {
			grantedBy = seedAuthzChangedBy
		}

		desiredAssignmentsBySubject[key][roleID] = desiredAssignment{
			RoleID:    roleID,
			GrantedBy: grantedBy,
		}
	}

	if len(subjectsToSync) == 0 {
		deps.Logger.Warnw("⚠️  没有需要同步的角色分配主体，跳过")
		return nil
	}

	var grantedCount, revokedCount int

	for _, subjectKey := range sortedAssignmentSubjectKeys(subjectsToSync) {
		currentAssignments, err := assignmentQueryer.ListBySubject(ctx, assignmentDomain.ListBySubjectQuery{
			SubjectType: subjectKey.SubjectType,
			SubjectID:   subjectKey.SubjectID,
			TenantID:    subjectKey.TenantID,
		})
		if err != nil {
			return fmt.Errorf("list assignments for %s/%s: %w", subjectKey.SubjectType, subjectKey.SubjectID, err)
		}

		currentManaged := make(map[uint64]*assignmentDomain.Assignment)
		for _, assignment := range currentAssignments {
			if _, ok := managedRoleIDs[assignment.RoleID]; ok {
				currentManaged[assignment.RoleID] = assignment
			}
		}

		desiredForSubject := desiredAssignmentsBySubject[subjectKey]

		for roleID := range currentManaged {
			if _, ok := desiredForSubject[roleID]; ok {
				continue
			}
			if err := assignmentCommander.Revoke(ctx, assignmentDomain.RevokeCommand{
				SubjectType: subjectKey.SubjectType,
				SubjectID:   subjectKey.SubjectID,
				RoleID:      roleID,
				TenantID:    subjectKey.TenantID,
			}); err != nil {
				return fmt.Errorf("revoke role %d from %s/%s: %w", roleID, subjectKey.SubjectType, subjectKey.SubjectID, err)
			}
			revokedCount++
		}

		for roleID, assignment := range desiredForSubject {
			if _, ok := currentManaged[roleID]; ok {
				continue
			}
			if _, err := assignmentCommander.Grant(ctx, assignmentDomain.GrantCommand{
				SubjectType: subjectKey.SubjectType,
				SubjectID:   subjectKey.SubjectID,
				RoleID:      roleID,
				TenantID:    subjectKey.TenantID,
				GrantedBy:   assignment.GrantedBy,
			}); err != nil {
				return fmt.Errorf("grant role %d to %s/%s: %w", roleID, subjectKey.SubjectType, subjectKey.SubjectID, err)
			}
			grantedCount++
		}
	}

	deps.Logger.Infow("✅ 角色分配数据已同步",
		"configured", len(config.Assignments),
		"granted", grantedCount,
		"revoked", revokedCount,
	)
	return nil
}

// ==================== Casbin 策略 Seed 函数 ====================

// seedCasbinPolicies 创建 Casbin 策略规则
//
// 业务说明：
// - 从配置文件读取基础的 RBAC 策略规则
// - 定义角色的资源访问权限
// - 设置角色继承关系
//
// 幂等性：按受管角色 + 租户精确同步
func seedCasbinPolicies(ctx context.Context, deps *dependencies) error {
	if deps.Config == nil || len(deps.Config.Policies) == 0 {
		deps.Logger.Warnw("⚠️  配置文件中没有 Casbin 策略，跳过")
		return nil
	}

	casbinPort, err := casbininfra.NewCasbinAdapter(deps.DB, deps.CasbinModel)
	if err != nil {
		return fmt.Errorf("init casbin adapter: %w", err)
	}

	enforcerProvider, ok := casbinPort.(casbinEnforcerProvider)
	if !ok {
		return fmt.Errorf("casbin adapter does not expose enforcer")
	}

	desiredState, err := buildDesiredPolicyState(deps.Config.Policies, deps.Config.Roles)
	if err != nil {
		return err
	}

	rawPolicies, err := enforcerProvider.Enforcer().GetPolicy()
	if err != nil {
		return fmt.Errorf("list current casbin policies: %w", err)
	}

	currentPolicies := make(map[string]policyDomain.PolicyRule)
	for _, raw := range rawPolicies {
		if len(raw) < 4 {
			continue
		}
		rule := policyDomain.PolicyRule{
			Sub: raw[0],
			Dom: raw[1],
			Obj: raw[2],
			Act: raw[3],
		}
		if isManagedPolicyRule(rule, desiredState.ManagedRoleKey, desiredState.ManagedTenant) {
			currentPolicies[policyRuleKey(rule)] = rule
		}
	}

	rawGroupings, err := enforcerProvider.Enforcer().GetGroupingPolicy()
	if err != nil {
		return fmt.Errorf("list current casbin groupings: %w", err)
	}

	currentGroupings := make(map[string]policyDomain.GroupingRule)
	for _, raw := range rawGroupings {
		if len(raw) < 3 {
			continue
		}
		rule := policyDomain.GroupingRule{
			Sub:  raw[0],
			Role: raw[1],
			Dom:  raw[2],
		}
		if isManagedGroupingRule(rule, desiredState.ManagedRoleKey, desiredState.ManagedTenant) {
			currentGroupings[groupingRuleKey(rule)] = rule
		}
	}

	var (
		removePolicies  []policyDomain.PolicyRule
		addPolicies     []policyDomain.PolicyRule
		removeGroupings []policyDomain.GroupingRule
		addGroupings    []policyDomain.GroupingRule
	)

	touchedTenants := make(map[string]struct{})

	for key, current := range currentPolicies {
		if _, ok := desiredState.Policies[key]; ok {
			continue
		}
		removePolicies = append(removePolicies, current)
		touchedTenants[current.Dom] = struct{}{}
	}
	for key, desired := range desiredState.Policies {
		if _, ok := currentPolicies[key]; ok {
			continue
		}
		addPolicies = append(addPolicies, desired)
		touchedTenants[desired.Dom] = struct{}{}
	}

	for key, current := range currentGroupings {
		if _, ok := desiredState.Groupings[key]; ok {
			continue
		}
		removeGroupings = append(removeGroupings, current)
		touchedTenants[current.Dom] = struct{}{}
	}
	for key, desired := range desiredState.Groupings {
		if _, ok := currentGroupings[key]; ok {
			continue
		}
		addGroupings = append(addGroupings, desired)
		touchedTenants[desired.Dom] = struct{}{}
	}

	if len(removePolicies) > 0 {
		if err := casbinPort.RemovePolicy(ctx, removePolicies...); err != nil {
			return fmt.Errorf("remove stale policy rules: %w", err)
		}
	}
	if len(addPolicies) > 0 {
		if err := casbinPort.AddPolicy(ctx, addPolicies...); err != nil {
			return fmt.Errorf("add policy rules: %w", err)
		}
	}
	if len(removeGroupings) > 0 {
		if err := casbinPort.RemoveGroupingPolicy(ctx, removeGroupings...); err != nil {
			return fmt.Errorf("remove stale grouping rules: %w", err)
		}
	}
	if len(addGroupings) > 0 {
		if err := casbinPort.AddGroupingPolicy(ctx, addGroupings...); err != nil {
			return fmt.Errorf("add grouping rules: %w", err)
		}
	}

	if err := syncPolicyVersions(ctx, deps, desiredState.ManagedTenant, touchedTenants); err != nil {
		return err
	}

	deps.Logger.Infow("✅ Casbin 策略规则已同步",
		"policies_added", len(addPolicies),
		"policies_removed", len(removePolicies),
		"groupings_added", len(addGroupings),
		"groupings_removed", len(removeGroupings),
	)
	return nil
}

// ==================== 辅助函数 ====================

func resourceNeedsUpdate(existing *resourceDomain.Resource, desired ResourceConfig) bool {
	if existing == nil {
		return true
	}
	if existing.DisplayName != desired.DisplayName {
		return true
	}
	if existing.AppName != desired.AppName {
		return true
	}
	if existing.Domain != desired.Domain {
		return true
	}
	if existing.Type != desired.Type {
		return true
	}
	if existing.Description != desired.Description {
		return true
	}
	return !stringSlicesEqual(existing.Actions, desired.Actions)
}

func isManagedResourceKey(key string) bool {
	return strings.HasPrefix(key, managedIAMResourcePrefix) || strings.HasPrefix(key, managedQSResourcePrefix)
}

func stringSlicesEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func managedRoleIDsFromState(state *seedContext) (map[uint64]struct{}, error) {
	managed := make(map[uint64]struct{}, len(state.Roles))
	for alias := range state.Roles {
		id, err := resolveRoleAlias(0, "@"+alias, state)
		if err != nil {
			return nil, fmt.Errorf("resolve managed role %s: %w", alias, err)
		}
		managed[id] = struct{}{}
	}
	return managed, nil
}

func collectAssignmentTenants(config *SeedConfig) map[string]struct{} {
	tenants := make(map[string]struct{})
	for _, role := range config.Roles {
		tenantID := strings.TrimSpace(role.TenantID)
		if tenantID != "" {
			tenants[tenantID] = struct{}{}
		}
	}
	for _, assignment := range config.Assignments {
		tenantID := strings.TrimSpace(assignment.TenantID)
		if tenantID != "" {
			tenants[tenantID] = struct{}{}
		}
	}
	return tenants
}

func parseAssignmentSubjectType(raw string) (assignmentDomain.SubjectType, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", string(assignmentDomain.SubjectTypeUser):
		return assignmentDomain.SubjectTypeUser, nil
	case string(assignmentDomain.SubjectTypeGroup):
		return assignmentDomain.SubjectTypeGroup, nil
	default:
		return "", fmt.Errorf("unsupported subject type %q", raw)
	}
}

func sortedAssignmentSubjectKeys(subjects map[assignmentSubjectKey]struct{}) []assignmentSubjectKey {
	keys := make([]assignmentSubjectKey, 0, len(subjects))
	for key := range subjects {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].TenantID != keys[j].TenantID {
			return keys[i].TenantID < keys[j].TenantID
		}
		if keys[i].SubjectType != keys[j].SubjectType {
			return keys[i].SubjectType < keys[j].SubjectType
		}
		return keys[i].SubjectID < keys[j].SubjectID
	})
	return keys
}

func buildDesiredPolicyState(policyConfigs []PolicyConfig, roleConfigs []RoleConfig) (*desiredPolicyState, error) {
	state := &desiredPolicyState{
		Policies:       make(map[string]policyDomain.PolicyRule, len(policyConfigs)),
		Groupings:      make(map[string]policyDomain.GroupingRule, len(policyConfigs)),
		ManagedRoleKey: make(map[string]struct{}, len(roleConfigs)),
		ManagedTenant:  make(map[string]struct{}),
	}

	for _, role := range roleConfigs {
		state.ManagedRoleKey["role:"+strings.TrimSpace(role.Name)] = struct{}{}
		tenantID := strings.TrimSpace(role.TenantID)
		if tenantID != "" {
			state.ManagedTenant[tenantID] = struct{}{}
		}
	}

	for _, policyConfig := range policyConfigs {
		policyType := strings.ToLower(strings.TrimSpace(policyConfig.Type))
		subject := strings.TrimSpace(policyConfig.Subject)
		if subject == "" {
			return nil, fmt.Errorf("policy subject is required")
		}

		switch policyType {
		case "p":
			if len(policyConfig.Values) != 3 {
				return nil, fmt.Errorf("policy %s expects 3 values, got %d", subject, len(policyConfig.Values))
			}
			rule := policyDomain.PolicyRule{
				Sub: subject,
				Dom: strings.TrimSpace(policyConfig.Values[0]),
				Obj: strings.TrimSpace(policyConfig.Values[1]),
				Act: normalizePolicyActionPattern(policyConfig.Values[2]),
			}
			if rule.Dom == "" || rule.Obj == "" || rule.Act == "" {
				return nil, fmt.Errorf("policy %s contains empty values", subject)
			}
			state.Policies[policyRuleKey(rule)] = rule
			state.ManagedRoleKey[rule.Sub] = struct{}{}
			state.ManagedTenant[rule.Dom] = struct{}{}

		case "g":
			if len(policyConfig.Values) != 2 {
				return nil, fmt.Errorf("grouping %s expects 2 values, got %d", subject, len(policyConfig.Values))
			}
			rule := policyDomain.GroupingRule{
				Sub:  subject,
				Role: strings.TrimSpace(policyConfig.Values[0]),
				Dom:  strings.TrimSpace(policyConfig.Values[1]),
			}
			if rule.Role == "" || rule.Dom == "" {
				return nil, fmt.Errorf("grouping %s contains empty values", subject)
			}
			state.Groupings[groupingRuleKey(rule)] = rule
			state.ManagedRoleKey[rule.Sub] = struct{}{}
			state.ManagedRoleKey[rule.Role] = struct{}{}
			state.ManagedTenant[rule.Dom] = struct{}{}

		default:
			return nil, fmt.Errorf("unsupported policy type %q", policyConfig.Type)
		}
	}

	return state, nil
}

func normalizePolicyActionPattern(action string) string {
	action = strings.TrimSpace(action)
	if action == "*" {
		return ".*"
	}
	return action
}

func isManagedPolicyRule(rule policyDomain.PolicyRule, managedRoles, managedTenants map[string]struct{}) bool {
	if _, ok := managedRoles[rule.Sub]; !ok {
		return false
	}
	_, tenantManaged := managedTenants[rule.Dom]
	return tenantManaged
}

func isManagedGroupingRule(rule policyDomain.GroupingRule, managedRoles, managedTenants map[string]struct{}) bool {
	if _, ok := managedRoles[rule.Sub]; !ok {
		return false
	}
	if _, ok := managedRoles[rule.Role]; !ok {
		return false
	}
	_, tenantManaged := managedTenants[rule.Dom]
	return tenantManaged
}

func policyRuleKey(rule policyDomain.PolicyRule) string {
	return strings.Join([]string{rule.Sub, rule.Dom, rule.Obj, rule.Act}, "\x1f")
}

func groupingRuleKey(rule policyDomain.GroupingRule) string {
	return strings.Join([]string{rule.Sub, rule.Role, rule.Dom}, "\x1f")
}

func syncPolicyVersions(
	ctx context.Context,
	deps *dependencies,
	managedTenants map[string]struct{},
	touchedTenants map[string]struct{},
) error {
	if len(managedTenants) == 0 {
		return nil
	}

	versionRepo := policyMysql.NewPolicyVersionRepository(deps.DB)
	for _, tenantID := range sortedMapKeys(managedTenants) {
		current, err := versionRepo.GetCurrent(ctx, tenantID)
		if err != nil {
			return fmt.Errorf("get current policy version for tenant %s: %w", tenantID, err)
		}
		if current == nil {
			if _, err := versionRepo.GetOrCreate(ctx, tenantID); err != nil {
				return fmt.Errorf("create initial policy version for tenant %s: %w", tenantID, err)
			}
			continue
		}
		if _, changed := touchedTenants[tenantID]; !changed {
			continue
		}
		if _, err := versionRepo.Increment(ctx, tenantID, seedAuthzChangedBy, seedAuthzVersionReason); err != nil {
			return fmt.Errorf("increment policy version for tenant %s: %w", tenantID, err)
		}
	}
	return nil
}

func sortedMapKeys(set map[string]struct{}) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
