package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/FangcunMount/component-base/pkg/log"
	ucUOW "github.com/FangcunMount/iam-contracts/internal/apiserver/application/uc/uow"
	userApp "github.com/FangcunMount/iam-contracts/internal/apiserver/application/uc/user"
)

// ==================== 系统初始化 Seed 函数 ====================

// seedSystemInit 系统初始化：创建管理员用户
//
// 业务说明：
// 1. 创建系统管理员用户（用于后续创建认证账号）
// 2. 返回的 state 保存用户ID，供后续步骤使用（如 authn 步骤）
//
// 幂等性：通过手机号查询检查，已存在的用户会被更新而不是重复创建
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
	// 先尝试通过手机号查询
	if res, err := userQuerySrv.GetByPhone(ctx, cfg.Phone); err == nil && res != nil {
		// 用户已存在，更新信息
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
		return res.ID, nil
	}

	// 用户不存在，创建新用户
	created, err := userAppSrv.Register(ctx, userApp.RegisterUserDTO{
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

// ==================== 登录并创建员工（QS 服务） ====================

// seedStaff 登录获取 token 后创建员工
// 必须在 authn 步骤之后调用，因为需要使用刚创建的认证账号登录
func seedStaff(ctx context.Context, deps *dependencies, state *seedContext) error {
	if deps.Config.QSServiceURL == "" {
		deps.Logger.Warnw("⚠️  未配置 QS 服务 URL，跳过员工创建")
		return nil
	}
	if deps.Config.IAMServiceURL == "" {
		deps.Logger.Warnw("⚠️  未配置 IAM 服务 URL，跳过员工创建")
		return nil
	}

	// 查找需要创建员工的用户及其对应的账号配置
	for _, uc := range deps.Config.Users {
		if len(uc.Roles) == 0 || uc.OrgID == 0 {
			continue // 跳过没有配置员工信息的用户
		}

		userID, ok := state.Users[uc.Alias]
		if !ok {
			deps.Logger.Warnw("⚠️  用户不存在，跳过员工创建", "alias", uc.Alias)
			continue
		}

		// 查找该用户对应的账号配置
		var account *AccountConfig
		for i := range deps.Config.Accounts {
			if deps.Config.Accounts[i].UserAlias == uc.Alias {
				account = &deps.Config.Accounts[i]
				break
			}
		}
		if account == nil {
			deps.Logger.Warnw("⚠️  未找到用户的认证账号配置，跳过员工创建", "alias", uc.Alias)
			continue
		}

		// 运营账号优先使用手机号形态的 external_id/username，否则回退用户手机号
		loginID := resolveLoginID(*account, uc)

		// 登录获取 token
		token, err := loginWithPassword(deps.Config.IAMServiceURL, loginID, account.Password)
		if err != nil {
			deps.Logger.Warnw("⚠️  登录失败，跳过员工创建",
				"alias", uc.Alias,
				"username", account.Username,
				"error", err)
			continue
		}

		// 创建员工
		if err := createStaff(deps.Config.QSServiceURL, token, userID, uc, deps.Logger); err != nil {
			deps.Logger.Warnw("⚠️  创建员工失败（非致命错误）",
				"alias", uc.Alias,
				"error", err)
		} else {
			deps.Logger.Infow("✅ 员工创建成功",
				"alias", uc.Alias,
				"org_id", uc.OrgID,
				"roles", uc.Roles)
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

// loginWithPassword 使用登录标识（与账户 ExternalID 相同，如手机号）+密码登录 IAM 获取 token
func loginWithPassword(iamServiceURL, loginID, password string) (string, error) {
	credentials, err := json.Marshal(struct {
		Username string `json:"username"`
		Password string `json:"password"`
		// TenantID 可选，0 表示默认租户
		TenantID uint64 `json:"tenant_id,omitempty"`
	}{
		Username: loginID,
		Password: password,
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
	Roles    []string `json:"roles"`
	UserID   int64    `json:"user_id"`
	Phone    string   `json:"phone,omitempty"`
	Email    string   `json:"email,omitempty"`
	IsActive bool     `json:"is_active"`
}

// createStaff 调用 QS 服务创建员工
func createStaff(qsServiceURL, adminToken, userID string, cfg UserConfig, logger log.Logger) error {
	// 解析 userID 为整数（64 位）
	uid, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid user_id: %w", err)
	}

	reqBody := CreateStaffRequest{
		Name:     cfg.Name,
		Roles:    cfg.Roles,
		UserID:   uid,
		OrgID:    int64(cfg.OrgID),
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

	// 读取响应体
	var respBodyBytes bytes.Buffer
	_, _ = respBodyBytes.ReadFrom(resp.Body)
	respBodyStr := respBodyBytes.String()

	// 记录响应详情
	logger.Infow("📥 收到创建员工响应",
		"status_code", resp.StatusCode,
		"status", resp.Status,
		"response_headers", resp.Header,
		"response_body", respBodyStr)

	if resp.StatusCode >= 400 {
		var respBody map[string]interface{}
		_ = json.Unmarshal(respBodyBytes.Bytes(), &respBody)
		logger.Errorw("❌ 创建员工失败",
			"status_code", resp.StatusCode,
			"response_body", respBody)
		return fmt.Errorf("create staff failed: status=%d, response=%v", resp.StatusCode, respBody)
	}

	logger.Infow("✅ 创建员工请求成功", "status_code", resp.StatusCode)
	return nil
}
