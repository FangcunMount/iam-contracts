// @title           IAM API Documentation
// @version         1.0.0
// @description     IAM 系统 REST API 文档，包含认证(Authentication)、授权(Authorization)、身份管理(Identity)和身份提供商(IDP)模块
// @termsOfService  https://iam.yangshujie.com/terms

// @contact.name   API Support
// @contact.url    https://github.com/FangcunMount/iam-contracts
// @contact.email  support@yangshujie.com

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      iam.yangshujie.com
// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description JWT 认证令牌，格式: Bearer {access_token}

// @tag.name Authentication
// @tag.description 认证相关接口 - 登录、登出、令牌管理
// @tag.name Tokens
// @tag.description 令牌管理 - 刷新、验证、撤销
// @tag.name Accounts
// @tag.description 账号管理 - 创建、查询、绑定第三方账号
// @tag.name JWKS
// @tag.description 公钥管理 - JWT 签名验证公钥集

// @tag.name Users
// @tag.description 用户管理 - 创建、查询、更新用户信息
// @tag.name Children
// @tag.description 儿童管理 - 注册、查询、更新儿童档案
// @tag.name Guardianship
// @tag.description 监护关系 - 授予、撤销、查询监护权

// @tag.name Roles
// @tag.description 角色管理 - 创建、查询、更新、删除角色
// @tag.name Assignments
// @tag.description 角色分配 - 授予、撤销用户或组的角色
// @tag.name Policies
// @tag.description 策略管理 - 添加、移除 RBAC 策略规则
// @tag.name Resources
// @tag.description 资源管理 - 创建、查询、更新受保护资源

// @tag.name Wechat Apps
// @tag.description 微信应用管理 - 配置、密钥轮换、令牌获取
// @tag.name Health
// @tag.description 健康检查 - 各模块健康状态

package main

import (
	"github.com/FangcunMount/iam-contracts/internal/apiserver"
)

func main() {
	apiserver.NewApp("iam-apiserver").Run()
}
