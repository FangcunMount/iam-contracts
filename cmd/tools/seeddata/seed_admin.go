package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/FangcunMount/component-base/pkg/log"
	ucUOW "github.com/FangcunMount/iam-contracts/internal/apiserver/application/uc/uow"
	userApp "github.com/FangcunMount/iam-contracts/internal/apiserver/application/uc/user"
)

const qsErrUserAlreadyExists = 110002

// ==================== 系统初始化 Seed 函数 ====================

// seedSystemInit 系统初始化：创建管理员用户
//
// 业务说明：
// 1. 创建系统管理员用户（用于后续创建认证账号）
// 2. 返回的 state 保存用户ID，供后续步骤使用（如 authn 步骤）
//
// 幂等性：有手机号则按手机号查；手机号为空且配置了 id 则按用户 id 查；否则仅首次 Register 可成功
func seedAdmin(ctx context.Context, deps *dependencies, state *seedContext) error {
	if deps.Config == nil || len(deps.Config.Users) == 0 {
		deps.Logger.Warnw("⚠️  配置文件中没有用户数据，跳过")
		return nil
	}

	// 初始化用户中心的工作单元和应用服务
	uow := ucUOW.NewUnitOfWork(deps.DB)
	userAppSrv := userApp.NewUserApplicationService(uow)
	userProfileSrv := userApp.NewUserProfileApplicationService(uow)
	userQuerySrv := userApp.NewUserQueryApplicationService(uow)

	// 创建配置中的所有用户（通常只有管理员）
	for _, uc := range deps.Config.Users {
		id, err := ensureSystemUser(ctx, userAppSrv, userProfileSrv, userQuerySrv, uc)
		if err != nil {
			return fmt.Errorf("ensure user %s: %w", uc.Alias, err)
		}
		state.Users[uc.Alias] = id
		deps.Logger.Infow("✅ 用户创建成功",
			"alias", uc.Alias,
			"name", uc.Name,
			"user_id", id)
	}

	deps.Logger.Infow("✅ 系统用户初始化完成", "count", len(deps.Config.Users))
	return nil
}

// ensureSystemUser 确保系统用户存在（如不存在则创建，如存在则更新）
func ensureSystemUser(
	ctx context.Context,
	userAppSrv userApp.UserApplicationService,
	userProfileSrv userApp.UserProfileApplicationService,
	userQuerySrv userApp.UserQueryApplicationService,
	cfg UserConfig,
) (string, error) {
	// 有手机号：按手机号做幂等与更新
	if strings.TrimSpace(cfg.Phone) != "" {
		if res, err := userQuerySrv.GetByPhone(ctx, cfg.Phone); err == nil && res != nil {
			if cfg.ID > 0 && res.ID != strconv.FormatUint(cfg.ID, 10) {
				return "", fmt.Errorf("user id mismatch for phone %s: existing=%s expected=%d", cfg.Phone, res.ID, cfg.ID)
			}
			applySeedUserUpdates(ctx, userProfileSrv, res, cfg)
			return res.ID, nil
		}
	} else if cfg.ID > 0 {
		// 无手机号：无法用 GetByPhone；按固定 id 查询以实现重复执行 seed 时的幂等
		idStr := strconv.FormatUint(cfg.ID, 10)
		if res, err := userQuerySrv.GetByID(ctx, idStr); err == nil && res != nil {
			applySeedUserUpdates(ctx, userProfileSrv, res, cfg)
			return res.ID, nil
		}
	}

	// 用户不存在，创建新用户
	created, err := userAppSrv.Register(ctx, userApp.RegisterUserDTO{
		ID:    cfg.ID,
		Name:  cfg.Name,
		Phone: cfg.Phone,
		Email: cfg.Email,
	})
	if err != nil {
		return "", err
	}

	// 如果有身份证号，更新身份证信息
	if cfg.IDCard != "" {
		_ = userProfileSrv.UpdateIDCard(ctx, created.ID, cfg.IDCard)
	}
	return created.ID, nil
}

func applySeedUserUpdates(
	ctx context.Context,
	userProfileSrv userApp.UserProfileApplicationService,
	res *userApp.UserResult,
	cfg UserConfig,
) {
	if res.Name != cfg.Name {
		_ = userProfileSrv.Rename(ctx, res.ID, cfg.Name)
	}
	if res.Email != cfg.Email {
		_ = userProfileSrv.UpdateContact(ctx, userApp.UpdateContactDTO{
			UserID: res.ID,
			Phone:  cfg.Phone,
			Email:  cfg.Email,
		})
	}
	if cfg.IDCard != "" && res.IDCard != cfg.IDCard {
		_ = userProfileSrv.UpdateIDCard(ctx, res.ID, cfg.IDCard)
	}
}

// ==================== 登录并创建员工（QS 服务） ====================

type qsBootstrapPrincipal struct {
	OrgID         int
	UserAlias     string
	LoginID       string
	Password      string
	Source        string
	SkipUserAlias string
}

type qsErrorResponse struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Reference string `json:"reference,omitempty"`
}

// seedStaff 使用每个 org 的 bootstrap admin token 统一创建员工。
// 必须在 authn 和 tenant-bootstrap-admin 之后调用，确保首个 active operator 已完成自举。
func seedStaff(ctx context.Context, deps *dependencies, state *seedContext) error {
	if deps.Config == nil {
		return nil
	}

	usersByOrg := collectQSStaffUsersByOrg(deps.Config)
	if len(usersByOrg) == 0 {
		deps.Logger.Infow("⏭️  没有需要创建的 QS 员工，跳过")
		return nil
	}
	if deps.Config.QSServiceURL == "" {
		deps.Logger.Warnw("⚠️  未配置 QS 服务 URL，跳过员工创建")
		return nil
	}
	if deps.Config.IAMServiceURL == "" {
		return fmt.Errorf("iam_service_url is required when seeding QS staff")
	}

	orgIDs := make([]int, 0, len(usersByOrg))
	for orgID := range usersByOrg {
		orgIDs = append(orgIDs, orgID)
	}
	sort.Ints(orgIDs)

	for _, orgID := range orgIDs {
		principal, err := selectQSBootstrapPrincipal(deps.Config, orgID)
		if err != nil {
			return fmt.Errorf("select bootstrap principal for org %d: %w", orgID, err)
		}

		token, err := loginWithPassword(
			deps.Config.IAMServiceURL,
			principal.LoginID,
			principal.Password,
			uint64(orgID),
		)
		if err != nil {
			return fmt.Errorf("login bootstrap principal %s for org %d: %w", principal.UserAlias, orgID, err)
		}

		deps.Logger.Infow("✅ 已获取 QS bootstrap token",
			"org_id", orgID,
			"user_alias", principal.UserAlias,
			"source", principal.Source)

		for _, uc := range usersByOrg[orgID] {
			qsRoles := resolveQSBootstrapRoles(deps.Config, uc)
			if len(qsRoles) == 0 {
				continue
			}

			if shouldSkipQSStaffCreate(principal, uc) {
				deps.Logger.Infow("⏭️  跳过 bootstrap admin 的 /staff 创建",
					"alias", uc.Alias,
					"org_id", uc.OrgID,
					"source", principal.Source)
				continue
			}

			userID, ok := state.Users[uc.Alias]
			if !ok {
				return fmt.Errorf("user %s not found in seed state", uc.Alias)
			}

			if err := createStaff(deps.Config.QSServiceURL, token, userID, uc, qsRoles, deps.Logger); err != nil {
				return fmt.Errorf("create staff for %s in org %d: %w", uc.Alias, orgID, err)
			}

			deps.Logger.Infow("✅ 员工创建成功",
				"alias", uc.Alias,
				"org_id", uc.OrgID,
				"roles", qsRoles,
				"bootstrap_alias", principal.UserAlias)
		}
	}

	return nil
}

// LoginRequest IAM 登录请求
type LoginRequest struct {
	Method      string          `json:"method"`
	Credentials json.RawMessage `json:"credentials"` // 直接传递接口期望的 JSON 对象
	DeviceID    string          `json:"device_id,omitempty"`
}

// TokenPair IAM 登录响应
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// loginWithPassword 使用登录标识（与账户 ExternalID 相同，如手机号）+密码登录 IAM 获取 token。
// 对 QS / collection 这类要求 JWT tenant_id=org_id 的调用方，应显式传入 org 作用域。
func loginWithPassword(iamServiceURL, loginID, password string, tenantID uint64) (string, error) {
	credentials, err := json.Marshal(struct {
		Username string `json:"username"`
		Password string `json:"password"`
		// TenantID 可选；0 表示不显式指定，由调用方自行承担默认域语义。
		TenantID uint64 `json:"tenant_id,omitempty"`
	}{
		Username: loginID,
		Password: password,
		TenantID: tenantID,
	})
	if err != nil {
		return "", fmt.Errorf("marshal credentials: %w", err)
	}

	reqBody := LoginRequest{
		Method:      "password",
		Credentials: credentials,
		DeviceID:    "seeddata-tool",
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	url := iamServiceURL + "/authn/login"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var respBody map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&respBody)
		return "", fmt.Errorf("login failed: status=%d, response=%v", resp.StatusCode, respBody)
	}

	// 响应包装格式为 {"code":0,"data":{...},"message":"..."}
	var wrapper struct {
		Code    int       `json:"code"`
		Message string    `json:"message"`
		Data    TokenPair `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if wrapper.Code != 0 {
		return "", fmt.Errorf("login failed: code=%d, message=%s, data=%v", wrapper.Code, wrapper.Message, wrapper.Data)
	}

	return wrapper.Data.AccessToken, nil
}

// ==================== 创建员工（QS 服务） ====================

// CreateStaffRequest 创建员工请求体
type CreateStaffRequest struct {
	Name     string   `json:"name"`
	OrgID    int64    `json:"org_id"`
	UserID   int64    `json:"user_id"`
	Roles    []string `json:"roles,omitempty"`
	Phone    string   `json:"phone,omitempty"`
	Email    string   `json:"email,omitempty"`
	IsActive bool     `json:"is_active"`
}

// createStaff 调用 QS 服务创建员工
func createStaff(qsServiceURL, adminToken, userID string, cfg UserConfig, roles []string, logger log.Logger) error {
	// 解析 userID 为整数（64 位）
	uid, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid user_id: %w", err)
	}

	reqBody := CreateStaffRequest{
		Name:     cfg.Name,
		UserID:   uid,
		OrgID:    int64(cfg.OrgID),
		Roles:    append([]string(nil), roles...),
		Phone:    cfg.Phone,
		Email:    cfg.Email,
		IsActive: cfg.IsActive,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	url := qsServiceURL + "/staff"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if adminToken != "" {
		req.Header.Set("Authorization", "Bearer "+adminToken)
	}

	// 记录请求详情
	logger.Infow("📤 发送创建员工请求",
		"url", url,
		"method", "POST",
		"request_body", string(body),
		"has_token", adminToken != "",
		"token_prefix", func() string {
			if len(adminToken) > 20 {
				return adminToken[:20] + "..."
			}
			return adminToken
		}())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Errorw("❌ 请求失败", "error", err)
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBodyBytes, _ := io.ReadAll(resp.Body)
	respBodyStr := string(respBodyBytes)

	// 记录响应详情
	logger.Infow("📥 收到创建员工响应",
		"status_code", resp.StatusCode,
		"status", resp.Status,
		"response_headers", resp.Header,
		"response_body", respBodyStr)

	if resp.StatusCode >= 400 {
		var errResp qsErrorResponse
		if err := json.Unmarshal(respBodyBytes, &errResp); err == nil &&
			resp.StatusCode == http.StatusBadRequest &&
			errResp.Code == qsErrUserAlreadyExists {
			logger.Infow("⏭️  员工已存在，按幂等成功处理",
				"org_id", cfg.OrgID,
				"user_id", uid,
				"roles", roles,
				"code", errResp.Code,
				"message", errResp.Message)
			return nil
		}

		var respBody map[string]interface{}
		_ = json.Unmarshal(respBodyBytes, &respBody)
		logger.Errorw("❌ 创建员工失败",
			"status_code", resp.StatusCode,
			"response_body", respBody)
		return fmt.Errorf("create staff failed: status=%d, response=%v", resp.StatusCode, respBody)
	}

	logger.Infow("✅ 创建员工请求成功", "status_code", resp.StatusCode)
	return nil
}

func collectQSStaffUsersByOrg(cfg *SeedConfig) map[int][]UserConfig {
	if cfg == nil {
		return nil
	}

	usersByOrg := make(map[int][]UserConfig)
	for _, uc := range cfg.Users {
		qsRoles := resolveQSBootstrapRoles(cfg, uc)
		if uc.OrgID == 0 || len(qsRoles) == 0 {
			continue
		}
		usersByOrg[uc.OrgID] = append(usersByOrg[uc.OrgID], uc)
	}
	return usersByOrg
}

func selectQSBootstrapPrincipal(cfg *SeedConfig, orgID int) (*qsBootstrapPrincipal, error) {
	if cfg == nil {
		return nil, fmt.Errorf("seed config is nil")
	}

	for _, bootstrap := range cfg.TenantBootstrapAdmins {
		if int(bootstrap.QSOrgID) != orgID || !bootstrap.BootstrapQSOperator {
			continue
		}

		account := bootstrap.BootstrapAccount
		if strings.TrimSpace(account.Provider) == "" {
			account.Provider = "operation"
		}
		if account.Provider != "operation" {
			return nil, fmt.Errorf("bootstrap account for org %d must use operation provider", orgID)
		}

		loginID := resolveLoginID(account, bootstrap.BootstrapUser)
		if strings.TrimSpace(loginID) == "" {
			return nil, fmt.Errorf("bootstrap account for org %d has empty login id", orgID)
		}
		if strings.TrimSpace(account.Password) == "" {
			return nil, fmt.Errorf("bootstrap account for org %d has empty password", orgID)
		}

		return &qsBootstrapPrincipal{
			OrgID:         orgID,
			UserAlias:     bootstrap.BootstrapUser.Alias,
			LoginID:       loginID,
			Password:      account.Password,
			Source:        "tenant_bootstrap_admins",
			SkipUserAlias: bootstrap.BootstrapUser.Alias,
		}, nil
	}

	for _, uc := range cfg.Users {
		if uc.OrgID != orgID {
			continue
		}
		qsRoles := resolveQSBootstrapRoles(cfg, uc)
		if !containsString(qsRoles, "qs:admin") {
			continue
		}

		account, ok := findOperationAccountByUserAlias(cfg, uc.Alias)
		if !ok {
			continue
		}
		loginID := resolveLoginID(account, uc)
		if strings.TrimSpace(loginID) == "" {
			continue
		}
		if strings.TrimSpace(account.Password) == "" {
			continue
		}

		return &qsBootstrapPrincipal{
			OrgID:     orgID,
			UserAlias: uc.Alias,
			LoginID:   loginID,
			Password:  account.Password,
			Source:    "qs_admin_assignment",
		}, nil
	}

	return nil, fmt.Errorf("no bootstrap principal with qs:admin assignment and operation account found")
}

func findOperationAccountByUserAlias(cfg *SeedConfig, userAlias string) (AccountConfig, bool) {
	if cfg == nil {
		return AccountConfig{}, false
	}

	for _, account := range cfg.Accounts {
		if account.UserAlias != userAlias {
			continue
		}
		provider := strings.TrimSpace(account.Provider)
		if provider == "" {
			provider = "operation"
		}
		if provider != "operation" {
			continue
		}
		return account, true
	}
	return AccountConfig{}, false
}

func shouldSkipQSStaffCreate(principal *qsBootstrapPrincipal, uc UserConfig) bool {
	if principal == nil {
		return false
	}
	if principal.Source != "tenant_bootstrap_admins" {
		return false
	}
	return uc.Alias != "" && uc.Alias == principal.SkipUserAlias
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == target {
			return true
		}
	}
	return false
}

func resolveQSBootstrapRoles(cfg *SeedConfig, uc UserConfig) []string {
	if cfg == nil || uc.OrgID == 0 {
		return append([]string(nil), uc.Roles...)
	}

	roleNameByAlias := make(map[string]string, len(cfg.Roles))
	for _, role := range cfg.Roles {
		alias := strings.TrimSpace(role.Alias)
		name := strings.TrimSpace(role.Name)
		if alias != "" && name != "" {
			roleNameByAlias[alias] = name
		}
	}

	tenantID := strconv.Itoa(uc.OrgID)
	seen := make(map[string]struct{})
	roles := make([]string, 0)

	for _, assignment := range cfg.Assignments {
		subjectType := strings.TrimSpace(assignment.SubjectType)
		if subjectType != "" && subjectType != "user" {
			continue
		}
		if strings.TrimSpace(assignment.SubjectID) != "@"+uc.Alias {
			continue
		}
		if strings.TrimSpace(assignment.TenantID) != tenantID {
			continue
		}

		roleAlias := strings.TrimPrefix(strings.TrimSpace(assignment.RoleAlias), "@")
		roleName := strings.TrimSpace(roleNameByAlias[roleAlias])
		if roleName == "" || !strings.HasPrefix(roleName, "qs:") {
			continue
		}
		if _, ok := seen[roleName]; ok {
			continue
		}
		seen[roleName] = struct{}{}
		roles = append(roles, roleName)
	}

	if len(roles) > 0 {
		return roles
	}
	return append([]string(nil), uc.Roles...)
}
