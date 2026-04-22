package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	registerApp "github.com/FangcunMount/iam-contracts/internal/apiserver/application/authn/register"
	authnUOW "github.com/FangcunMount/iam-contracts/internal/apiserver/application/authn/uow"
	assignmentApp "github.com/FangcunMount/iam-contracts/internal/apiserver/application/authz/assignment"
	authzUOW "github.com/FangcunMount/iam-contracts/internal/apiserver/application/authz/uow"
	ucUOW "github.com/FangcunMount/iam-contracts/internal/apiserver/application/uc/uow"
	userApp "github.com/FangcunMount/iam-contracts/internal/apiserver/application/uc/user"
	authnAuth "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authn/authentication"
	assignmentDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authz/assignment"
	roleDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authz/role"
	ucUserDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/uc/user"
	casbininfra "github.com/FangcunMount/iam-contracts/internal/apiserver/infra/casbin"
	"github.com/FangcunMount/iam-contracts/internal/apiserver/infra/crypto"
	accountRepo "github.com/FangcunMount/iam-contracts/internal/apiserver/infra/mysql/account"
	assignmentMysql "github.com/FangcunMount/iam-contracts/internal/apiserver/infra/mysql/assignment"
	credentialRepo "github.com/FangcunMount/iam-contracts/internal/apiserver/infra/mysql/credential"
	roleMysql "github.com/FangcunMount/iam-contracts/internal/apiserver/infra/mysql/role"
	userRepo "github.com/FangcunMount/iam-contracts/internal/apiserver/infra/mysql/user"
	wechatInfra "github.com/FangcunMount/iam-contracts/internal/apiserver/infra/wechat"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
	"gorm.io/gorm/clause"
)

const (
	tenantBootstrapGrantedBy = "seeddata"
)

type tenantBootstrapTemplates struct {
	iamSourceDomain string
	qsSourceDomain  string
	iamRoles        []RoleConfig
	qsRoles         []RoleConfig
	iamPolicies     []PolicyConfig
	qsPolicies      []PolicyConfig
	iamRoleNames    map[string]struct{}
	qsRoleNames     map[string]struct{}
}

type seedAuthnSupport struct {
	userRepository       ucUserDomain.Repository
	accountRepository    *accountRepo.AccountRepository
	credentialRepository *credentialRepo.Repository
	passwordHasher       authnAuth.PasswordHasher
	registerService      registerApp.RegisterApplicationService
}

// seedTenantBootstrapAdmins 显式执行 per-tenant bootstrap admin 初始化。
// 这条流程既可用于未来新增 tenant，也可用于 fangcun 默认租户在 QS 空库时自举首个 operator；
// system/admin/content_manager 的 IAM 用户、账号与 assignment 真值仍由既有 users/accounts/assignments 管理。
func seedTenantBootstrapAdmins(ctx context.Context, deps *dependencies, state *seedContext) error {
	config := deps.Config
	if config == nil || len(config.TenantBootstrapAdmins) == 0 {
		deps.Logger.Infow("⏭️  未配置 tenant_bootstrap_admins，跳过显式租户管理员自举")
		return nil
	}

	templates, err := buildTenantBootstrapTemplates(config)
	if err != nil {
		return err
	}

	roleRepo := roleMysql.NewRoleRepository(deps.DB)

	casbinPort, err := casbininfra.NewCasbinAdapter(deps.DB, deps.CasbinModel)
	if err != nil {
		return fmt.Errorf("init casbin adapter: %w", err)
	}
	syncAdapter, ok := casbinPort.(casbinSyncAdapter)
	if !ok {
		return fmt.Errorf("casbin adapter does not expose sync capabilities")
	}

	assignmentRepo := assignmentMysql.NewAssignmentRepository(deps.DB)
	assignmentManager := assignmentDomain.NewValidator(assignmentRepo, roleRepo, userRepo.NewRepository(deps.DB))
	assignmentCommander := assignmentApp.NewAssignmentCommandService(
		assignmentManager,
		authzUOW.NewUnitOfWork(deps.DB),
		casbinPort,
		nil,
	)
	assignmentQueryer := assignmentApp.NewAssignmentQueryService(assignmentManager, assignmentRepo)

	userUOW := ucUOW.NewUnitOfWork(deps.DB)
	userAppSrv := userApp.NewUserApplicationService(userUOW)
	userProfileSrv := userApp.NewUserProfileApplicationService(userUOW)
	userQuerySrv := userApp.NewUserQueryApplicationService(userUOW)

	authnSupport, err := newSeedAuthnSupport(deps)
	if err != nil {
		return err
	}

	for _, bootstrap := range config.TenantBootstrapAdmins {
		if err := validateTenantBootstrapAdminConfig(bootstrap, templates); err != nil {
			return fmt.Errorf("invalid tenant bootstrap admin %s: %w", bootstrap.TenantCode, err)
		}

		if err := ensureTenantRecord(ctx, deps, bootstrap); err != nil {
			return fmt.Errorf("ensure tenant %s: %w", bootstrap.TenantCode, err)
		}

		iamRoleIDs, qsRoleIDs, err := ensureTenantBootstrapRoles(ctx, roleRepo, templates, bootstrap)
		if err != nil {
			return fmt.Errorf("ensure bootstrap roles for tenant %s: %w", bootstrap.TenantCode, err)
		}

		desiredState, err := buildTenantBootstrapDesiredState(templates, bootstrap)
		if err != nil {
			return fmt.Errorf("build desired policy state for tenant %s: %w", bootstrap.TenantCode, err)
		}
		if err := syncCasbinDesiredState(ctx, deps, syncAdapter, desiredState); err != nil {
			return fmt.Errorf("sync bootstrap policies for tenant %s: %w", bootstrap.TenantCode, err)
		}

		userCfg := bootstrap.BootstrapUser
		if userCfg.OrgID == 0 {
			userCfg.OrgID = int(bootstrap.QSOrgID)
		}
		userID, err := ensureSystemUser(ctx, userAppSrv, userProfileSrv, userQuerySrv, userCfg)
		if err != nil {
			return fmt.Errorf("ensure bootstrap user %s: %w", bootstrap.BootstrapUser.Alias, err)
		}
		if userCfg.Alias != "" {
			state.Users[userCfg.Alias] = userID
		}

		accountCfg := bootstrap.BootstrapAccount
		if strings.TrimSpace(accountCfg.UserAlias) == "" {
			accountCfg.UserAlias = userCfg.Alias
		}
		accountID, err := ensureSeedOperationAccount(ctx, deps, authnSupport, bootstrap, accountCfg, userID)
		if err != nil {
			return fmt.Errorf("ensure bootstrap account %s: %w", accountCfg.Alias, err)
		}
		if accountCfg.Alias != "" {
			state.Accounts[accountCfg.Alias] = accountID
		}

		if err := syncBootstrapAssignmentsForDomain(
			ctx,
			assignmentCommander,
			assignmentQueryer,
			userID,
			bootstrap.TenantCode,
			iamRoleIDs,
			bootstrap.Grants.IAMRoles,
		); err != nil {
			return fmt.Errorf("sync iam assignments for %s: %w", bootstrap.TenantCode, err)
		}

		qsDomain := strconv.FormatInt(bootstrap.QSOrgID, 10)
		if err := syncBootstrapAssignmentsForDomain(
			ctx,
			assignmentCommander,
			assignmentQueryer,
			userID,
			qsDomain,
			qsRoleIDs,
			bootstrap.Grants.QSRoles,
		); err != nil {
			return fmt.Errorf("sync qs assignments for %s/%s: %w", bootstrap.TenantCode, qsDomain, err)
		}

		if bootstrap.BootstrapQSOperator {
			if err := bootstrapQSOperator(ctx, deps.Config.QSInternalGRPC, bootstrap, userID); err != nil {
				return fmt.Errorf("bootstrap qs operator for tenant %s: %w", bootstrap.TenantCode, err)
			}
		}

		deps.Logger.Infow("✅ 租户 bootstrap admin 已收敛",
			"tenant", bootstrap.TenantCode,
			"qs_org_id", bootstrap.QSOrgID,
			"user_alias", bootstrap.BootstrapUser.Alias,
			"account_alias", bootstrap.BootstrapAccount.Alias,
		)
	}

	return nil
}

func newSeedAuthnSupport(deps *dependencies) (*seedAuthnSupport, error) {
	unitOfWork := authnUOW.NewUnitOfWork(deps.DB)
	userRepository := userRepo.NewRepository(deps.DB)
	accountRepository := accountRepo.NewAccountRepository(deps.DB)
	credentialRepository := credentialRepo.NewRepository(deps.DB)

	pepper := os.Getenv("SEEDDATA_PASSWORD_PEPPER")
	passwordHasher := crypto.NewArgon2Hasher(pepper)
	idp := wechatInfra.NewIdentityProvider(nil, nil)

	registerService := registerApp.NewRegisterApplicationService(
		unitOfWork,
		passwordHasher,
		idp,
		userRepository,
		nil,
		nil,
	)

	return &seedAuthnSupport{
		userRepository:       userRepository,
		accountRepository:    accountRepository,
		credentialRepository: credentialRepository,
		passwordHasher:       passwordHasher,
		registerService:      registerService,
	}, nil
}

func ensureSeedOperationAccount(
	ctx context.Context,
	deps *dependencies,
	support *seedAuthnSupport,
	bootstrap TenantBootstrapAdminConfig,
	ac AccountConfig,
	userIDStr string,
) (uint64, error) {
	if support == nil {
		return 0, fmt.Errorf("authn support is nil")
	}
	scopedFallback := bootstrap.ScopedTenantID
	if scopedFallback == 0 && bootstrap.QSOrgID > 0 {
		scopedFallback = uint64(bootstrap.QSOrgID)
	}
	if err := validateOperationAccountConfig(ac, scopedFallback); err != nil {
		return 0, err
	}

	userID, err := parseAuthnUserID(userIDStr)
	if err != nil {
		return 0, err
	}
	user, err := support.userRepository.FindByID(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("get user %s: %w", userID, err)
	}

	loginExternalID := accountOperaExternalID(ac, user.Email.String())
	scopedTenantID := ac.ScopedTenantID
	if scopedTenantID == 0 {
		scopedTenantID = scopedFallback
	}
	req := registerApp.RegisterRequest{
		Name:           user.Name,
		Phone:          user.Phone,
		Email:          user.Email,
		ExistingUserID: userID,
		OperaLoginID:   loginExternalID,
		ScopedTenantID: meta.FromUint64(scopedTenantID),
		AccountType:    "opera",
		CredentialType: registerApp.CredTypePassword,
		Password:       &ac.Password,
	}

	result, err := support.registerService.Register(ctx, req)
	if err != nil {
		if handled, accID, handleErr := handleAuthnConflict(
			ctx,
			deps,
			support.accountRepository,
			support.credentialRepository,
			support.passwordHasher,
			ac,
			userID,
			loginExternalID,
			err,
		); handled {
			if handleErr != nil {
				return 0, handleErr
			}
			if syncErr := syncSeedAccountStatus(ctx, support.accountRepository, ac, meta.FromUint64(accID)); syncErr != nil {
				return 0, syncErr
			}
			return accID, nil
		}
		return 0, err
	}

	if err := syncSeedAccountStatus(ctx, support.accountRepository, ac, result.AccountID); err != nil {
		return 0, err
	}
	return result.AccountID.Uint64(), nil
}

func buildTenantBootstrapTemplates(config *SeedConfig) (*tenantBootstrapTemplates, error) {
	if config == nil {
		return nil, fmt.Errorf("seed config is nil")
	}

	templates := &tenantBootstrapTemplates{
		iamRoleNames: map[string]struct{}{
			"tenant_admin": {},
			"user":         {},
		},
		qsRoleNames: make(map[string]struct{}),
	}

	for _, role := range config.Roles {
		name := strings.TrimSpace(role.Name)
		tenantID := strings.TrimSpace(role.TenantID)
		switch {
		case name == "tenant_admin" || name == "user":
			if templates.iamSourceDomain == "" {
				templates.iamSourceDomain = tenantID
			}
			if tenantID == templates.iamSourceDomain {
				templates.iamRoles = append(templates.iamRoles, role)
			}
		case strings.HasPrefix(name, "qs:"):
			if templates.qsSourceDomain == "" {
				templates.qsSourceDomain = tenantID
			}
			if tenantID == templates.qsSourceDomain {
				templates.qsRoles = append(templates.qsRoles, role)
				templates.qsRoleNames[name] = struct{}{}
			}
		}
	}

	if templates.iamSourceDomain == "" || len(templates.iamRoles) == 0 {
		return nil, fmt.Errorf("missing iam bootstrap role templates")
	}
	if templates.qsSourceDomain == "" || len(templates.qsRoles) == 0 {
		return nil, fmt.Errorf("missing qs bootstrap role templates")
	}

	for _, policy := range config.Policies {
		switch strings.ToLower(strings.TrimSpace(policy.Type)) {
		case "p":
			if len(policy.Values) != 3 {
				continue
			}
			domain := strings.TrimSpace(policy.Values[0])
			subject := strings.TrimSpace(policy.Subject)
			if domain == templates.iamSourceDomain && isIAMBootstrapPolicySubject(subject) {
				templates.iamPolicies = append(templates.iamPolicies, policy)
			}
			if domain == templates.qsSourceDomain && isQSBootstrapPolicySubject(subject) {
				templates.qsPolicies = append(templates.qsPolicies, policy)
			}
		case "g":
			if len(policy.Values) != 2 {
				continue
			}
			domain := strings.TrimSpace(policy.Values[1])
			subject := strings.TrimSpace(policy.Subject)
			role := strings.TrimSpace(policy.Values[0])
			if domain == templates.iamSourceDomain && isIAMBootstrapPolicySubject(subject) && isIAMBootstrapPolicySubject(role) {
				templates.iamPolicies = append(templates.iamPolicies, policy)
			}
			if domain == templates.qsSourceDomain && isQSBootstrapPolicySubject(subject) && isQSBootstrapPolicySubject(role) {
				templates.qsPolicies = append(templates.qsPolicies, policy)
			}
		}
	}

	if len(templates.iamPolicies) == 0 {
		return nil, fmt.Errorf("missing iam bootstrap policy templates")
	}
	if len(templates.qsPolicies) == 0 {
		return nil, fmt.Errorf("missing qs bootstrap policy templates")
	}

	return templates, nil
}

func validateTenantBootstrapAdminConfig(cfg TenantBootstrapAdminConfig, templates *tenantBootstrapTemplates) error {
	if strings.TrimSpace(cfg.TenantCode) == "" {
		return fmt.Errorf("tenant_code is required")
	}
	if strings.TrimSpace(cfg.BootstrapUser.Alias) == "" {
		return fmt.Errorf("bootstrap_user.alias is required")
	}
	if cfg.BootstrapUser.ID == 0 {
		return fmt.Errorf("bootstrap_user.id is required for idempotent seed")
	}
	if strings.TrimSpace(cfg.BootstrapUser.Name) == "" {
		return fmt.Errorf("bootstrap_user.name is required")
	}
	if strings.TrimSpace(cfg.BootstrapAccount.Alias) == "" {
		return fmt.Errorf("bootstrap_account.alias is required")
	}
	if strings.TrimSpace(cfg.BootstrapAccount.ExternalID) == "" && strings.TrimSpace(cfg.BootstrapAccount.Username) == "" {
		return fmt.Errorf("bootstrap_account.external_id is required")
	}
	if strings.TrimSpace(cfg.BootstrapAccount.Provider) == "" {
		cfg.BootstrapAccount.Provider = "operation"
	}
	if cfg.BootstrapAccount.Provider != "operation" {
		return fmt.Errorf("bootstrap_account.provider must be operation")
	}
	if cfg.QSOrgID <= 0 {
		return fmt.Errorf("qs_org_id must be positive")
	}
	if len(cfg.Grants.IAMRoles) == 0 {
		return fmt.Errorf("grants.iam_roles is required")
	}
	for _, roleName := range cfg.Grants.IAMRoles {
		if _, ok := templates.iamRoleNames[strings.TrimSpace(roleName)]; !ok {
			return fmt.Errorf("unsupported iam grant role %q", roleName)
		}
	}
	for _, roleName := range cfg.Grants.QSRoles {
		if _, ok := templates.qsRoleNames[strings.TrimSpace(roleName)]; !ok {
			return fmt.Errorf("unsupported qs grant role %q", roleName)
		}
	}
	return nil
}

func ensureTenantRecord(ctx context.Context, deps *dependencies, cfg TenantBootstrapAdminConfig) error {
	name := strings.TrimSpace(cfg.TenantName)
	if name == "" {
		name = cfg.TenantCode
	}
	now := time.Now()
	po := tenantPO{
		ID:        cfg.TenantCode,
		Code:      cfg.TenantCode,
		Name:      name,
		Status:    "active",
		MaxUsers:  1000,
		MaxRoles:  100,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return deps.DB.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"name",
				"code",
				"status",
				"max_users",
				"max_roles",
				"updated_at",
			}),
		}).
		Create(&po).Error
}

func ensureTenantBootstrapRoles(
	ctx context.Context,
	roleRepo roleDomain.Repository,
	templates *tenantBootstrapTemplates,
	cfg TenantBootstrapAdminConfig,
) (map[string]uint64, map[string]uint64, error) {
	iamRoleIDs := make(map[string]uint64, len(templates.iamRoles))
	for _, role := range templates.iamRoles {
		cloned := cloneRoleForTenant(role, cfg.TenantCode)
		idStr, err := ensureRole(ctx, roleRepo, cloned)
		if err != nil {
			return nil, nil, err
		}
		id, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			return nil, nil, err
		}
		iamRoleIDs[cloned.Name] = id
	}

	qsDomain := strconv.FormatInt(cfg.QSOrgID, 10)
	qsRoleIDs := make(map[string]uint64, len(templates.qsRoles))
	for _, role := range templates.qsRoles {
		cloned := cloneRoleForTenant(role, qsDomain)
		idStr, err := ensureRole(ctx, roleRepo, cloned)
		if err != nil {
			return nil, nil, err
		}
		id, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			return nil, nil, err
		}
		qsRoleIDs[cloned.Name] = id
	}

	return iamRoleIDs, qsRoleIDs, nil
}

func buildTenantBootstrapDesiredState(
	templates *tenantBootstrapTemplates,
	cfg TenantBootstrapAdminConfig,
) (*desiredPolicyState, error) {
	roleConfigs := make([]RoleConfig, 0, len(templates.iamRoles)+len(templates.qsRoles))
	for _, role := range templates.iamRoles {
		roleConfigs = append(roleConfigs, cloneRoleForTenant(role, cfg.TenantCode))
	}
	qsDomain := strconv.FormatInt(cfg.QSOrgID, 10)
	for _, role := range templates.qsRoles {
		roleConfigs = append(roleConfigs, cloneRoleForTenant(role, qsDomain))
	}

	policies := make([]PolicyConfig, 0, len(templates.iamPolicies)+len(templates.qsPolicies))
	for _, policy := range templates.iamPolicies {
		policies = append(policies, clonePolicyForDomain(policy, cfg.TenantCode))
	}
	for _, policy := range templates.qsPolicies {
		policies = append(policies, clonePolicyForDomain(policy, qsDomain))
	}

	return buildDesiredPolicyState(policies, roleConfigs)
}

func syncBootstrapAssignmentsForDomain(
	ctx context.Context,
	commander *assignmentApp.AssignmentCommandService,
	queryer *assignmentApp.AssignmentQueryService,
	userID string,
	tenantID string,
	roleIDsByName map[string]uint64,
	desiredRoleNames []string,
) error {
	desiredRoleIDs := make(map[uint64]struct{}, len(desiredRoleNames))
	managedRoleIDs := make(map[uint64]struct{}, len(roleIDsByName))
	for _, roleID := range roleIDsByName {
		managedRoleIDs[roleID] = struct{}{}
	}
	for _, roleName := range desiredRoleNames {
		roleID, ok := roleIDsByName[strings.TrimSpace(roleName)]
		if !ok {
			return fmt.Errorf("role %q not found in domain %s", roleName, tenantID)
		}
		desiredRoleIDs[roleID] = struct{}{}
	}

	currentAssignments, err := queryer.ListBySubject(ctx, assignmentDomain.ListBySubjectQuery{
		SubjectType: assignmentDomain.SubjectTypeUser,
		SubjectID:   userID,
		TenantID:    tenantID,
	})
	if err != nil {
		return err
	}

	for _, assignment := range currentAssignments {
		if _, ok := managedRoleIDs[assignment.RoleID]; !ok {
			continue
		}
		if _, keep := desiredRoleIDs[assignment.RoleID]; keep {
			continue
		}
		if err := commander.Revoke(ctx, assignmentDomain.RevokeCommand{
			SubjectType: assignmentDomain.SubjectTypeUser,
			SubjectID:   userID,
			RoleID:      assignment.RoleID,
			TenantID:    tenantID,
		}); err != nil {
			return err
		}
	}

	for roleID := range desiredRoleIDs {
		exists := false
		for _, assignment := range currentAssignments {
			if assignment.RoleID == roleID {
				exists = true
				break
			}
		}
		if exists {
			continue
		}
		if _, err := commander.Grant(ctx, assignmentDomain.GrantCommand{
			SubjectType: assignmentDomain.SubjectTypeUser,
			SubjectID:   userID,
			RoleID:      roleID,
			TenantID:    tenantID,
			GrantedBy:   tenantBootstrapGrantedBy,
		}); err != nil {
			return err
		}
	}
	return nil
}

func cloneRoleForTenant(src RoleConfig, tenantID string) RoleConfig {
	cloned := src
	cloned.TenantID = tenantID
	cloned.Alias = fmt.Sprintf("tenant_bootstrap_%s_%s", sanitizeRoleAliasPart(tenantID), sanitizeRoleAliasPart(src.Name))
	return cloned
}

func clonePolicyForDomain(src PolicyConfig, targetDomain string) PolicyConfig {
	cloned := src
	cloned.Values = append([]string(nil), src.Values...)
	switch strings.ToLower(strings.TrimSpace(cloned.Type)) {
	case "p":
		if len(cloned.Values) >= 1 {
			cloned.Values[0] = targetDomain
		}
	case "g":
		if len(cloned.Values) >= 2 {
			cloned.Values[1] = targetDomain
		}
	}
	return cloned
}

func isIAMBootstrapPolicySubject(subject string) bool {
	subject = strings.TrimSpace(subject)
	return subject == "role:tenant_admin" || subject == "role:user"
}

func isQSBootstrapPolicySubject(subject string) bool {
	subject = strings.TrimSpace(subject)
	return strings.HasPrefix(subject, "role:qs:")
}

func sanitizeRoleAliasPart(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.ReplaceAll(s, ":", "_")
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, ".", "_")
	return s
}

func bootstrapQSOperator(
	ctx context.Context,
	grpcCfg QSInternalGRPCConfig,
	bootstrap TenantBootstrapAdminConfig,
	userID string,
) error {
	if strings.TrimSpace(grpcCfg.Address) == "" {
		return fmt.Errorf("qs_internal_grpc.address is required when bootstrap_qs_operator=true")
	}

	uid, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return fmt.Errorf("parse bootstrap user id %q: %w", userID, err)
	}

	req := qsBootstrapOperatorRequest{
		OrgID:    bootstrap.QSOrgID,
		UserID:   uid,
		Name:     bootstrap.BootstrapUser.Name,
		Email:    bootstrap.BootstrapUser.Email,
		Phone:    bootstrap.BootstrapUser.Phone,
		IsActive: bootstrap.BootstrapUser.IsActive,
	}
	_, err = callQSBootstrapOperator(ctx, grpcCfg, req)
	return err
}
