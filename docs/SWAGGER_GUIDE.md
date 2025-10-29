# Swagger UI 集成指南

本文档介绍如何在 IAM 项目中使用 Swagger UI 查看和测试 API 文档。

---

## 目录

- [方案选择](#方案选择)
- [方案一：在线工具（快速预览）](#方案一在线工具快速预览)
- [方案二：swaggo 集成（推荐）](#方案二swaggo-集成推荐)
- [方案三：独立部署](#方案三独立部署)
- [最佳实践](#最佳实践)

---

## 方案选择

| 方案 | 适用场景 | 优点 | 缺点 |
|------|----------|------|------|
| **在线工具** | 快速预览、分享 | 无需安装、即开即用 | 需要网络、无法调试 |
| **swaggo 集成** | 开发环境、自动化 | 文档随代码更新、可直接测试 | 需要集成代码 |
| **独立部署** | 生产环境、团队共享 | 独立服务、安全可控 | 需要额外部署 |

---

## 方案一：在线工具（快速预览）

### 1. Swagger Editor

**最快速的预览方式**

1. 访问 https://editor.swagger.io/
2. 复制 YAML 文件内容（如 `api/rest/authn.v1.yaml`）
3. 粘贴到左侧编辑器
4. 右侧自动显示 API 文档

**优点**:

- ✅ 无需安装任何工具
- ✅ 实时验证语法错误
- ✅ 可以导出为 HTML/PDF

**缺点**:

- ❌ 无法直接调用本地 API
- ❌ 每次需要手动复制粘贴

### 2. Redocly

**更美观的在线预览**

```bash
# 安装 Redocly CLI
npm install -g @redocly/cli

# 预览单个文档
redocly preview-docs api/rest/authn.v1.yaml

# 构建静态 HTML
redocly build-docs api/rest/authn.v1.yaml -o docs/authn-api.html
```

访问 http://localhost:8080 查看

---

## 方案二：swaggo 集成（推荐）

### 架构说明

```
项目结构:
├── cmd/apiserver/
│   └── apiserver.go          # 主程序，添加 Swagger 注解
├── internal/apiserver/
│   ├── docs/                 # swag 生成的文档（自动生成）
│   │   ├── docs.go
│   │   ├── swagger.json
│   │   └── swagger.yaml
│   └── modules/
│       ├── authn/
│       │   └── interface/restful/
│       │       └── handler/   # API 处理器，添加注解
│       ├── authz/
│       └── idp/
└── api/rest/                 # 手写的 OpenAPI 规范（保留）
    ├── authn.v1.yaml
    ├── authz.v1.yaml
    ├── identity.v1.yaml
    └── idp.v1.yaml
```

### 步骤 1: 安装工具

```bash
# 安装 swag 命令行工具
go install github.com/swaggo/swag/cmd/swag@latest

# 验证安装
swag --version
```

### 步骤 2: 安装依赖包

```bash
# 在项目根目录执行
go get -u github.com/swaggo/swag
go get -u github.com/swaggo/gin-swagger
go get -u github.com/swaggo/files
```

### 步骤 3: 添加主程序注解

编辑 `cmd/apiserver/apiserver.go`，在 `package main` 上方添加：

```go
// @title           IAM API Documentation
// @version         1.0
// @description     IAM 系统 API 文档，包含认证、授权、身份管理和 IDP 模块
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
// @description JWT 认证，格式: Bearer {token}

// @tag.name Authentication
// @tag.description 认证相关接口
// @tag.name Authorization
// @tag.description 授权相关接口
// @tag.name Identity
// @tag.description 身份管理接口
// @tag.name IDP
// @tag.description 身份提供商接口

package main
```

### 步骤 4: 添加 Handler 注解

以 `authn` 模块的 `Login` 接口为例：

```go
// internal/apiserver/modules/authn/interface/restful/handler/login_handler.go

// Login 用户登录
// @Summary      用户登录
// @Description  支持多种登录方式：用户名密码、微信小程序
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        payload  body      LoginRequest  true  "登录请求"
// @Success      200      {object}  LoginResponse
// @Failure      400      {object}  ErrorResponse  "请求参数错误"
// @Failure      401      {object}  ErrorResponse  "认证失败"
// @Router       /auth/login [post]
func (h *LoginHandler) Login(c *gin.Context) {
    // 实现代码...
}

// RefreshToken 刷新访问令牌
// @Summary      刷新访问令牌
// @Description  使用 RefreshToken 获取新的 AccessToken
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        payload  body      RefreshTokenRequest  true  "刷新令牌请求"
// @Success      200      {object}  TokenPairResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      401      {object}  ErrorResponse
// @Router       /auth/refresh_token [post]
// @Security     BearerAuth
func (h *LoginHandler) RefreshToken(c *gin.Context) {
    // 实现代码...
}
```

**常用注解说明**:

| 注解 | 说明 | 示例 |
|------|------|------|
| `@Summary` | 接口简要描述 | `@Summary 用户登录` |
| `@Description` | 接口详细描述 | `@Description 支持多种登录方式` |
| `@Tags` | 接口分组标签 | `@Tags Authentication` |
| `@Accept` | 接受的内容类型 | `@Accept json` |
| `@Produce` | 返回的内容类型 | `@Produce json` |
| `@Param` | 参数定义 | `@Param id path int true "用户ID"` |
| `@Success` | 成功响应 | `@Success 200 {object} User` |
| `@Failure` | 失败响应 | `@Failure 404 {object} Error` |
| `@Router` | 路由路径 | `@Router /users/{id} [get]` |
| `@Security` | 安全认证 | `@Security BearerAuth` |

### 步骤 5: 生成文档

```bash
# 在项目根目录执行
swag init \
  -g cmd/apiserver/apiserver.go \
  -o internal/apiserver/docs \
  --parseDependency \
  --parseInternal

# 参数说明:
# -g: 主程序入口文件
# -o: 输出目录
# --parseDependency: 解析依赖包
# --parseInternal: 解析内部包
```

生成后会在 `internal/apiserver/docs/` 目录下生成：

- `docs.go`: Go 代码
- `swagger.json`: JSON 格式的 OpenAPI 规范
- `swagger.yaml`: YAML 格式的 OpenAPI 规范

### 步骤 6: 集成到路由

编辑 `internal/apiserver/routers.go`：

```go
package apiserver

import (
    "github.com/gin-gonic/gin"
    
    // 导入生成的 docs
    _ "github.com/FangcunMount/iam-contracts/internal/apiserver/docs"
    
    ginSwagger "github.com/swaggo/gin-swagger"
    swaggerFiles "github.com/swaggo/files"
)

func SetupRouters(engine *gin.Engine) {
    // 现有路由...
    
    // Swagger UI 路由
    engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
    
    // 或者使用自定义配置
    engine.GET("/swagger/*any", ginSwagger.WrapHandler(
        swaggerFiles.Handler,
        ginSwagger.URL("/swagger/doc.json"), // 指定 OpenAPI 文件路径
        ginSwagger.DefaultModelsExpandDepth(-1), // 默认折叠 Models
    ))
}
```

### 步骤 7: 启动服务并访问

```bash
# 启动服务
go run cmd/apiserver/apiserver.go

# 访问 Swagger UI
open http://localhost:8080/swagger/index.html
```

### 步骤 8: Makefile 集成

在 `Makefile` 中添加 Swagger 相关命令：

```makefile
# Swagger 文档生成
.PHONY: swagger
swagger: ## 生成 Swagger 文档
	@echo "==> Generating Swagger docs..."
	@swag init \
		-g cmd/apiserver/apiserver.go \
		-o internal/apiserver/docs \
		--parseDependency \
		--parseInternal
	@echo "✅ Swagger docs generated at internal/apiserver/docs/"

.PHONY: swagger-fmt
swagger-fmt: ## 格式化 Swagger 注解
	@echo "==> Formatting Swagger annotations..."
	@swag fmt -g cmd/apiserver/apiserver.go

.PHONY: swagger-validate
swagger-validate: ## 验证 Swagger 文档
	@echo "==> Validating Swagger docs..."
	@npx @redocly/cli lint internal/apiserver/docs/swagger.yaml

.PHONY: swagger-serve
swagger-serve: swagger ## 生成并启动 Swagger UI 服务
	@echo "==> Starting Swagger UI..."
	@echo "📖 Swagger UI: http://localhost:8080/swagger/index.html"
	@go run cmd/apiserver/apiserver.go
```

使用方式：

```bash
# 生成文档
make swagger

# 格式化注解
make swagger-fmt

# 验证文档
make swagger-validate

# 启动服务（自动生成文档）
make swagger-serve
```

---

## 方案三：独立部署

### 使用 Docker 部署 Swagger UI

创建 `build/docker/swagger-ui/docker-compose.yml`：

```yaml
version: '3.8'

services:
  swagger-ui:
    image: swaggerapi/swagger-ui:latest
    container_name: iam-swagger-ui
    ports:
      - "8081:8080"
    environment:
      # 支持多个 API 文档
      URLS: |
        [
          { url: "/api-docs/authn.v1.yaml", name: "Authentication API" },
          { url: "/api-docs/authz.v1.yaml", name: "Authorization API" },
          { url: "/api-docs/identity.v1.yaml", name: "Identity API" },
          { url: "/api-docs/idp.v1.yaml", name: "IDP API" }
        ]
      URLS_PRIMARY_NAME: "Authentication API"
    volumes:
      # 挂载 YAML 文件
      - ../../../api/rest:/usr/share/nginx/html/api-docs:ro
    restart: unless-stopped
```

启动：

```bash
cd build/docker/swagger-ui
docker-compose up -d

# 访问
open http://localhost:8081
```

### 使用 Nginx 代理

在生产环境中，可以通过 Nginx 代理 Swagger UI：

```nginx
# configs/nginx/conf.d/iam.yangshujie.com.conf

server {
    listen 443 ssl http2;
    server_name iam.yangshujie.com;

    # SSL 配置...

    # API 服务
    location /api/ {
        proxy_pass http://localhost:8080;
        # 其他代理配置...
    }

    # Swagger UI（仅限内网或认证后访问）
    location /swagger/ {
        # IP 白名单
        allow 192.168.1.0/24;
        deny all;
        
        proxy_pass http://localhost:8080/swagger/;
        # 或者代理到独立的 Swagger UI 容器
        # proxy_pass http://localhost:8081/;
    }
}
```

---

## 最佳实践

### 1. 环境隔离

```go
// internal/apiserver/routers.go

func SetupRouters(engine *gin.Engine, config *Config) {
    // 仅在开发/测试环境开启 Swagger UI
    if config.Env != "production" {
        engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
    }
    
    // 或者使用中间件保护
    if config.EnableSwagger {
        swaggerGroup := engine.Group("/swagger")
        swaggerGroup.Use(AuthMiddleware()) // 需要认证
        swaggerGroup.GET("/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
    }
}
```

### 2. 自动化流程

在 CI/CD 中集成：

```yaml
# .github/workflows/swagger.yml
name: Update Swagger Docs

on:
  push:
    branches: [main]
    paths:
      - 'internal/apiserver/**/*.go'
      - 'cmd/apiserver/**/*.go'

jobs:
  update-docs:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Install swag
        run: go install github.com/swaggo/swag/cmd/swag@latest
      
      - name: Generate Swagger docs
        run: make swagger
      
      - name: Commit changes
        run: |
          git config --local user.email "action@github.com"
          git config --local user.name "GitHub Action"
          git add internal/apiserver/docs/
          git commit -m "docs: update Swagger documentation" || exit 0
          git push
```

### 3. 文档版本管理

```go
// cmd/apiserver/apiserver.go

// @title           IAM API Documentation
// @version         1.0.0
// @description     IAM 系统 API 文档

// 在代码中同步版本号
const APIVersion = "v1.0.0"

func main() {
    // 更新 Swagger 版本信息
    docs.SwaggerInfo.Version = APIVersion
    // ...
}
```

### 4. 双重文档策略

推荐同时维护：

1. **手写 YAML 文档**（`api/rest/*.yaml`）- 作为 API 规范的权威来源
2. **swaggo 生成文档** - 确保代码和文档同步

定期对比两者，确保一致性：

```bash
# 对比脚本
make swagger
diff -u api/rest/authn.v1.yaml internal/apiserver/docs/swagger.yaml
```

---

## 快速开始示例

**1 分钟快速体验**:

```bash
# 1. 在线预览（最快）
# 访问 https://editor.swagger.io/
# 复制 api/rest/authn.v1.yaml 内容并粘贴

# 2. 本地预览（需要 Node.js）
npm install -g @redocly/cli
redocly preview-docs api/rest/authn.v1.yaml
# 访问 http://localhost:8080

# 3. swaggo 集成（需要修改代码）
go install github.com/swaggo/swag/cmd/swag@latest
# 按照上述步骤添加注解
make swagger
# 启动服务后访问 http://localhost:8080/swagger/index.html
```

---

## 常见问题

### Q: 手写 YAML 和 swaggo 生成的文档有什么区别？

A: 

- **手写 YAML**: 完整、精确，可以包含详细的业务逻辑说明，适合作为规范
- **swaggo 生成**: 自动同步代码，确保文档不过期，但可能缺少一些细节

**建议**: 两者结合使用，手写 YAML 作为设计规范，swaggo 确保实现同步。

### Q: 生产环境要不要开启 Swagger UI？

A: 不建议在公网开放。如需使用：

- ✅ 限制 IP 白名单
- ✅ 添加认证保护
- ✅ 使用独立域名/端口
- ✅ 定期审查访问日志

### Q: 如何更新文档？

A:
```bash
# 修改代码中的注解后执行
swag init -g cmd/apiserver/apiserver.go -o internal/apiserver/docs

# 或使用 Makefile
make swagger
```

### Q: 注解太多会影响代码可读性吗？

A: 可以使用 `//go:generate` 自动生成，或者将详细文档放在 YAML 中，代码只保留简单注解。

---

## 参考链接

- [Swagger Editor](https://editor.swagger.io/)
- [Redocly CLI](https://redocly.com/docs/cli/)
- [swaggo Documentation](https://github.com/swaggo/swag)
- [Swagger UI](https://swagger.io/tools/swagger-ui/)
- [OpenAPI Specification](https://spec.openapis.org/oas/v3.1.0)
