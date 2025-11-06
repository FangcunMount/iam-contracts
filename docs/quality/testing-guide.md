# 测试方案指南

本指南介绍 IAM Contracts 项目的完整测试策略、工具和最佳实践。

## 📋 目录

- [测试金字塔](#测试金字塔)
- [测试工具](#测试工具)
- [单元测试](#单元测试)
- [集成测试](#集成测试)
- [E2E 测试](#e2e-测试)
- [API 测试](#api-测试)
- [性能测试](#性能测试)
- [测试覆盖率](#测试覆盖率)
- [CI/CD 集成](#cicd-集成)
- [最佳实践](#最佳实践)

---

## 🔺 测试金字塔

我们遵循经典的测试金字塔策略：

```
        /\
       /  \     E2E Tests (10%)
      /----\
     /      \   Integration Tests (30%)
    /--------\
   /          \ Unit Tests (60%)
  /____________\
```

### 测试层级

| 层级 | 占比 | 速度 | 成本 | 示例 |
|------|------|------|------|------|
| **单元测试** | 60% | 快 | 低 | 领域模型测试、服务层测试 |
| **集成测试** | 30% | 中 | 中 | 数据库集成、Redis 集成 |
| **E2E 测试** | 10% | 慢 | 高 | 完整业务流程测试 |

---

## 🛠️ 测试工具

### 核心工具

#### 1. **Go Testing 标准库**
```bash
# 基本用法
go test ./...                    # 运行所有测试
go test -v ./...                 # 详细输出
go test -run TestXxx ./...       # 运行特定测试
go test -short ./...             # 跳过耗时测试
```

#### 2. **Testify 测试框架**
项目使用 `github.com/stretchr/testify` 提供断言和 Mock 功能。

```go
import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/stretchr/testify/suite"
)

// 基本断言
func TestSomething(t *testing.T) {
    assert.Equal(t, expected, actual, "they should be equal")
    require.NotNil(t, object) // 失败时立即终止
}

// Suite 测试
type MySuite struct {
    suite.Suite
    db *Database
}

func (s *MySuite) SetupTest() {
    // 每个测试前执行
    s.db = setupDB()
}

func (s *MySuite) TearDownTest() {
    // 每个测试后执行
    s.db.Close()
}

func TestMySuite(t *testing.T) {
    suite.Run(t, new(MySuite))
}
```

#### 3. **gomock (已集成)**
用于生成 Mock 对象。

```bash
# 安装 mockgen
go install go.uber.org/mock/mockgen@latest

# 生成 Mock
mockgen -source=repository.go -destination=mock_repository.go -package=mocks
```

#### 4. **httptest 标准库**
用于测试 HTTP 处理器。

```go
import (
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestHTTPHandler(t *testing.T) {
    req := httptest.NewRequest("GET", "/api/v1/users", nil)
    w := httptest.NewRecorder()
    
    handler.ServeHTTP(w, req)
    
    assert.Equal(t, http.StatusOK, w.Code)
}
```

### 推荐安装的其他工具

#### 1. **golangci-lint** - 代码质量检查
```bash
# macOS 安装
brew install golangci-lint

# 使用
make lint
golangci-lint run --timeout=5m ./...
```

#### 2. **gotestsum** - 美化测试输出
```bash
# 安装
go install gotest.tools/gotestsum@latest

# 使用
gotestsum --format testname
gotestsum --format dots-v2
```

#### 3. **go-test-report** - HTML 测试报告
```bash
# 安装
go install github.com/vakenbolt/go-test-report@latest

# 生成报告
go test -v ./... 2>&1 | go-test-report -o report.html
```

---

## 🧪 单元测试

### 目录结构

```
internal/apiserver/
├── domain/authn/
│   ├── account/
│   │   ├── account.go
│   │   └── account_test.go          # 领域模型测试
│   └── authentication/
│       ├── password.go
│       └── password_test.go
├── application/authn/
│   └── login/
│       ├── service.go
│       └── service_test.go          # 应用服务测试
├── infra/                         # 按技术栈拆分的基础设施实现
│   ├── crypto/
│   ├── jwt/
│   ├── redis/
│   └── wechat/
└── interface/authn/
    └── restful/
        └── handler/
            └── auth_test.go         # 接口层测试
```

### Make 命令

```bash
# 运行所有单元测试
make test-unit

# 运行特定模块测试
go test -v ./internal/apiserver/domain/authn/... \
           ./internal/apiserver/application/authn/... \
           ./internal/apiserver/infra/... \
           ./internal/apiserver/interface/authn/...

# 运行特定测试函数
go test -v -run TestPassword ./internal/apiserver/domain/authn/authentication/

# 使用短模式（跳过集成测试）
go test -short ./...
```

### 单元测试示例

#### 领域模型测试

```go
// internal/apiserver/domain/authn/authentication/password_test.go
package authentication_test

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestNewPassword_Success(t *testing.T) {
    // Arrange
    plaintext := "SecurePassword123!"
    
    // Act
    pwd, err := NewPassword(plaintext)
    
    // Assert
    require.NoError(t, err)
    assert.NotEmpty(t, pwd.Hash())
    assert.True(t, pwd.Verify(plaintext))
}

func TestNewPassword_TooShort(t *testing.T) {
    // Arrange
    plaintext := "123"
    
    // Act
    pwd, err := NewPassword(plaintext)
    
    // Assert
    assert.Error(t, err)
    assert.Nil(t, pwd)
    assert.Contains(t, err.Error(), "密码长度")
}

func TestPassword_VerifyFailed(t *testing.T) {
    // Arrange
    pwd, _ := NewPassword("CorrectPassword123!")
    
    // Act
    result := pwd.Verify("WrongPassword")
    
    // Assert
    assert.False(t, result)
}
```

#### 表驱动测试

```go
func TestPasswordValidation(t *testing.T) {
    tests := []struct {
        name      string
        password  string
        wantErr   bool
        errMsg    string
    }{
        {
            name:     "有效密码",
            password: "SecurePass123!",
            wantErr:  false,
        },
        {
            name:     "太短",
            password: "short",
            wantErr:  true,
            errMsg:   "密码长度",
        },
        {
            name:     "无特殊字符",
            password: "OnlyLetters123",
            wantErr:  true,
            errMsg:   "特殊字符",
        },
        {
            name:     "无数字",
            password: "OnlyLetters!@#",
            wantErr:  true,
            errMsg:   "数字",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := NewPassword(tt.password)
            
            if tt.wantErr {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.errMsg)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

#### 使用 Mock 测试

```go
// 假设我们有一个 UserRepository 接口
type UserRepository interface {
    FindByID(ctx context.Context, id string) (*User, error)
    Save(ctx context.Context, user *User) error
}

// 测试 UserService
func TestUserService_GetUser(t *testing.T) {
    // Arrange
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()
    
    mockRepo := mocks.NewMockUserRepository(ctrl)
    service := NewUserService(mockRepo)
    
    expectedUser := &User{ID: "123", Username: "test"}
    mockRepo.EXPECT().
        FindByID(gomock.Any(), "123").
        Return(expectedUser, nil)
    
    // Act
    user, err := service.GetUser(context.Background(), "123")
    
    // Assert
    require.NoError(t, err)
    assert.Equal(t, expectedUser, user)
}
```

---

## 🔗 集成测试

集成测试验证组件间的交互，通常涉及数据库、Redis 等外部依赖。

### 标记集成测试

使用 build tag 或检查环境变量来区分集成测试：

```go
//go:build integration
// +build integration

package repository_test

import "testing"

func TestUserRepository_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("跳过集成测试（short 模式）")
    }
    
    // 需要真实数据库的测试
    db := setupTestDatabase(t)
    defer cleanupTestDatabase(t, db)
    
    // 测试逻辑...
}
```

### 运行集成测试

```bash
# 跳过集成测试（开发时快速反馈）
go test -short ./...

# 仅运行集成测试
go test -tags=integration ./...

# 运行所有测试
go test ./...
```

### 测试数据库设置

#### 方法 1: 使用测试数据库

```go
package repository_test

import (
    "database/sql"
    "testing"
)

func setupTestDB(t *testing.T) *sql.DB {
    dsn := "root:REDACTED_DEV_ROOT@tcp(localhost:3306)/iam_contracts_test?parseTime=true"
    db, err := sql.Open("mysql", dsn)
    if err != nil {
        t.Fatalf("无法连接测试数据库: %v", err)
    }
    
    // 运行迁移
    runMigrations(t, db)
    
    return db
}

func cleanupTestDB(t *testing.T, db *sql.DB) {
    // 清理测试数据
    db.Exec("TRUNCATE TABLE users")
    db.Close()
}
```

#### 方法 2: 使用 Docker 容器

```bash
# 启动测试数据库容器
docker run -d \
    --name iam-test-db \
    -e MYSQL_ROOT_PASSWORD=test123 \
    -e MYSQL_DATABASE=iam_test \
    -p 3307:3306 \
    mysql:8.0

# 测试完成后清理
docker rm -f iam-test-db
```

#### 方法 3: 使用事务回滚

```go
func TestWithTransaction(t *testing.T) {
    db := getDB()
    tx, err := db.Begin()
    require.NoError(t, err)
    defer tx.Rollback() // 自动回滚，不污染数据库
    
    // 在事务中执行测试
    repo := NewUserRepository(tx)
    user := &User{Username: "test"}
    err = repo.Save(context.Background(), user)
    
    assert.NoError(t, err)
}
```

---

## 🌐 E2E 测试

端到端测试验证完整的业务流程，从 HTTP 请求到数据持久化。

### E2E 测试示例

项目中已有完整的 E2E 测试案例：`internal/apiserver/interface/authn/restful/e2e_test.go`

```go
// 完整的 JWT 签名 → JWKS 发布 → JWT 验证流程
func TestCompleteJWTFlow_E2E(t *testing.T) {
    if testing.Short() {
        t.Skip("跳过 E2E 测试")
    }
    
    // 1. 设置测试环境
    keyRepo := NewInMemoryKeyRepository()
    keyManager := service.NewKeyManager(keyRepo)
    jwksService := jwks.NewJWKSService(keyManager)
    
    // 2. 生成密钥对
    ctx := context.Background()
    key, err := keyManager.GenerateKey(ctx)
    require.NoError(t, err)
    
    // 3. 签发 JWT Token
    userID := idutil.NewID()
    jwtGen := jwtGen.NewJWTGenerator(crypto.NewRSASigner())
    token, err := jwtGen.GenerateToken(userID, key)
    require.NoError(t, err)
    
    // 4. 发布 JWKS
    jwksData, err := jwksService.GetJWKS(ctx)
    require.NoError(t, err)
    assert.NotEmpty(t, jwksData)
    
    // 5. 验证 Token
    verifier := authentication.NewJWTVerifier(jwksService)
    claims, err := verifier.Verify(ctx, token)
    require.NoError(t, err)
    assert.Equal(t, userID, claims.Subject)
}
```

### 运行 E2E 测试

```bash
# 运行所有 E2E 测试
go test -v ./internal/apiserver/interface/authn/restful/e2e_test.go

# 运行特定 E2E 测试
go test -v -run TestCompleteJWTFlow_E2E ./internal/apiserver/interface/authn/restful/
```

---

## 🌍 API 测试

详细的 API 测试方案请参考：[API_TESTING_GUIDE.md](./API_TESTING_GUIDE.md)

### 快速测试命令

```bash
# 健康检查
curl http://localhost:8080/healthz

# 查看所有路由
curl http://localhost:8080/debug/routes | jq

# 查看模块状态
curl http://localhost:8080/debug/modules | jq

# 测试用户注册
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "Test123!@#",
    "phone": "13800138000"
  }'

# 测试登录
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "credential_type": "password",
    "username": "testuser",
    "password": "Test123!@#"
  }'
```

### Postman 集合

创建 Postman 集合以便团队共享测试用例：

1. 在 Postman 中导入项目 API
2. 创建环境变量：
   ```json
   {
     "base_url": "http://localhost:8080",
     "access_token": "",
     "refresh_token": ""
   }
   ```
3. 导出集合到 `tests/postman/` 目录

---

## ⚡ 性能测试

### 基准测试

Go 内置支持基准测试：

```go
func BenchmarkPasswordHashing(b *testing.B) {
    password := "SecurePassword123!"
    
    b.ResetTimer() // 重置计时器
    for i := 0; i < b.N; i++ {
        NewPassword(password)
    }
}

func BenchmarkJWTGeneration(b *testing.B) {
    signer := crypto.NewRSASigner()
    generator := jwt.NewJWTGenerator(signer)
    userID := "test-user-123"
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        generator.GenerateToken(userID, privateKey)
    }
}
```

运行基准测试：

```bash
# 运行所有基准测试
make test-bench

# 运行特定基准测试
go test -bench=BenchmarkPassword -benchmem ./internal/apiserver/domain/authn/authentication/

# 比较性能（优化前后）
go test -bench=. -benchmem ./... > old.txt
# 修改代码...
go test -bench=. -benchmem ./... > new.txt
benchcmp old.txt new.txt
```

### 压力测试工具

#### 1. **hey** - HTTP 负载生成器

```bash
# 安装
brew install hey

# 测试登录接口
hey -n 1000 -c 10 -m POST \
    -H "Content-Type: application/json" \
    -d '{"credential_type":"password","username":"test","password":"Test123!@#"}' \
    http://localhost:8080/api/v1/auth/login

# 测试结果分析
# - Requests/sec: 每秒请求数
# - Average latency: 平均延迟
# - Status code distribution: 状态码分布
```

#### 2. **wrk** - 现代 HTTP 基准测试工具

```bash
# 安装
brew install wrk

# 简单测试
wrk -t4 -c100 -d30s http://localhost:8080/healthz

# 使用 Lua 脚本测试复杂场景
wrk -t4 -c100 -d30s -s login.lua http://localhost:8080
```

#### 3. **vegeta** - 灵活的负载测试工具

```bash
# 安装
brew install vegeta

# 创建测试目标文件 targets.txt
echo "POST http://localhost:8080/api/v1/auth/login
Content-Type: application/json
@login.json" > targets.txt

# 执行测试
vegeta attack -targets=targets.txt -rate=100 -duration=30s | vegeta report
```

### 竞态检测

```bash
# 运行竞态检测
make test-race

# 或直接使用 go test
go test -race ./...

# 检测特定包
go test -race ./internal/apiserver/{domain,application,infra,interface}/authn/...
```

---

## 📊 测试覆盖率

### 生成覆盖率报告

```bash
# 使用 Make 命令（推荐）
make test-coverage

# 手动生成
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# 查看覆盖率统计
go tool cover -func=coverage.out
```

### 覆盖率报告解读

```bash
# 输出示例
github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authn/account/account.go:25:    NewAccount              100.0%
github.com/FangcunMount/iam-contracts/internal/apiserver/domain/authn/account/account.go:45:    Validate                85.7%
total:                                                                                                  (statements)            78.5%
```

### 覆盖率目标

- **整体覆盖率**: ≥ 70%
- **领域层**: ≥ 80%
- **应用服务层**: ≥ 75%
- **HTTP 处理层**: ≥ 60%
- **基础设施层**: ≥ 50%

### 持续监控

将覆盖率检查集成到 CI 流程：

```yaml
# .github/workflows/test.yml
- name: Test with Coverage
  run: |
    go test -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//' > coverage.txt
    COVERAGE=$(cat coverage.txt)
    if (( $(echo "$COVERAGE < 70" | bc -l) )); then
      echo "Coverage is below 70%: $COVERAGE%"
      exit 1
    fi
```

---

## 🚀 CI/CD 集成

### GitHub Actions 示例

创建 `.github/workflows/test.yml`:

```yaml
name: Tests

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main, develop ]

jobs:
  test:
    runs-on: ubuntu-latest
    
    services:
      mysql:
        image: mysql:8.0
        env:
          MYSQL_ROOT_PASSWORD: test_root_123
          MYSQL_DATABASE: iam_contracts_test
        ports:
          - 3306:3306
        options: >-
          --health-cmd="mysqladmin ping"
          --health-interval=10s
          --health-timeout=5s
          --health-retries=5
      
      redis-cache:
        image: redis:7-alpine
        ports:
          - 6379:6379
      
      redis-store:
        image: redis:7-alpine
        ports:
          - 6380:6379
    
    steps:
    - name: Checkout code
      uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    
    - name: Cache Go modules
      uses: actions/cache@v3
      with:
        path: ~/go/pkg/mod
        key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
        restore-keys: |
          ${{ runner.os }}-go-
    
    - name: Download dependencies
      run: go mod download
    
    - name: Run unit tests
      run: go test -short -v ./...
    
    - name: Run integration tests
      run: go test -v ./...
      env:
        DB_DSN: root:test_root_123@tcp(localhost:3306)/iam_contracts_test?parseTime=true
        REDIS_CACHE_ADDR: localhost:6379
        REDIS_STORE_ADDR: localhost:6380
    
    - name: Run race detector
      run: go test -race ./...
    
    - name: Generate coverage report
      run: |
        go test -coverprofile=coverage.out ./...
        go tool cover -html=coverage.out -o coverage.html
    
    - name: Upload coverage
      uses: codecov/codecov-action@v3
      with:
        files: ./coverage.out
    
    - name: Lint
      run: |
        go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
        golangci-lint run --timeout=5m ./...
```

### Jenkins Pipeline 示例

```groovy
pipeline {
    agent any
    
    stages {
        stage('Checkout') {
            steps {
                checkout scm
            }
        }
        
        stage('Dependencies') {
            steps {
                sh 'go mod download'
            }
        }
        
        stage('Unit Tests') {
            steps {
                sh 'go test -short -v ./...'
            }
        }
        
        stage('Integration Tests') {
            steps {
                sh 'docker-compose -f build/docker/docker-compose-test.yml up -d'
                sh 'sleep 10' // 等待服务启动
                sh 'go test -v ./...'
                sh 'docker-compose -f build/docker/docker-compose-test.yml down'
            }
        }
        
        stage('Coverage') {
            steps {
                sh 'go test -coverprofile=coverage.out ./...'
                sh 'go tool cover -html=coverage.out -o coverage.html'
                publishHTML([
                    reportDir: '.',
                    reportFiles: 'coverage.html',
                    reportName: 'Coverage Report'
                ])
            }
        }
        
        stage('Lint') {
            steps {
                sh 'golangci-lint run --timeout=5m ./...'
            }
        }
    }
    
    post {
        always {
            junit '**/test-results/*.xml'
            archiveArtifacts artifacts: 'coverage.*', fingerprint: true
        }
    }
}
```

---

## 💡 最佳实践

### 1. 测试命名规范

```go
// ✅ 好的命名
func TestUserService_CreateUser_WithValidData_Success(t *testing.T)
func TestUserService_CreateUser_WithDuplicateUsername_ReturnsError(t *testing.T)
func TestPasswordValidator_ValidatePassword_TooShort_ReturnsError(t *testing.T)

// ❌ 不好的命名
func TestUser(t *testing.T)
func Test1(t *testing.T)
func TestCreateUser(t *testing.T) // 不够具体
```

### 2. AAA 模式（Arrange-Act-Assert）

```go
func TestUserService_CreateUser(t *testing.T) {
    // Arrange（准备）
    service := NewUserService(mockRepo)
    userData := &CreateUserRequest{
        Username: "testuser",
        Password: "Test123!@#",
    }
    
    // Act（执行）
    user, err := service.CreateUser(context.Background(), userData)
    
    // Assert（断言）
    require.NoError(t, err)
    assert.NotNil(t, user)
    assert.Equal(t, "testuser", user.Username)
}
```

### 3. 表驱动测试

```go
func TestPasswordValidation(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
        errMsg  string
    }{
        {"有效密码", "Valid123!@#", false, ""},
        {"太短", "short", true, "长度"},
        {"无数字", "NoNumbers!@#", true, "数字"},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidatePassword(tt.input)
            if tt.wantErr {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.errMsg)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### 4. 测试隔离

```go
// ✅ 每个测试独立
func TestUserRepository_Save(t *testing.T) {
    db := setupTestDB(t)
    defer cleanupTestDB(t, db)
    
    // 测试逻辑...
}

// ❌ 测试间存在依赖
var globalUser *User // 避免使用全局状态

func TestCreateUser(t *testing.T) {
    globalUser = createUser() // 影响其他测试
}

func TestDeleteUser(t *testing.T) {
    deleteUser(globalUser) // 依赖上一个测试
}
```

### 5. 失败信息要清晰

```go
// ✅ 好的断言信息
assert.Equal(t, expected, actual, 
    "用户创建后 ID 应该不为空，expected: %v, actual: %v", expected, actual)

// ❌ 不够清晰
assert.Equal(t, expected, actual) // 失败时难以定位问题
```

### 6. 使用 Subtests

```go
func TestUserOperations(t *testing.T) {
    t.Run("Create", func(t *testing.T) {
        // 创建用户测试
    })
    
    t.Run("Update", func(t *testing.T) {
        // 更新用户测试
    })
    
    t.Run("Delete", func(t *testing.T) {
        // 删除用户测试
    })
}
```

### 7. Mock 最佳实践

```go
// ✅ 明确 Mock 期望
mockRepo.EXPECT().
    FindByID(gomock.Any(), "123").
    Return(expectedUser, nil).
    Times(1) // 明确调用次数

// ✅ 使用 gomock.Any() 忽略不重要的参数
mockRepo.EXPECT().
    Save(gomock.Any(), gomock.Any()).
    DoAndReturn(func(ctx context.Context, user *User) error {
        // 自定义逻辑
        return nil
    })

// ❌ 过度 Mock
// 不要 Mock 你不拥有的代码（如第三方库）
// 不要 Mock 简单的值对象
```

### 8. 测试数据管理

```go
// ✅ 使用工厂函数
func NewTestUser(overrides ...func(*User)) *User {
    user := &User{
        ID:       idutil.NewID(),
        Username: "testuser",
        Email:    "test@example.com",
    }
    
    for _, override := range overrides {
        override(user)
    }
    
    return user
}

// 使用
user := NewTestUser(func(u *User) {
    u.Username = "customname"
})
```

### 9. 并发测试

```go
func TestConcurrentAccess(t *testing.T) {
    cache := NewCache()
    
    // 使用 WaitGroup 等待所有 goroutine
    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            cache.Set(fmt.Sprintf("key-%d", id), id)
        }(i)
    }
    
    wg.Wait()
    
    assert.Equal(t, 100, cache.Len())
}
```

### 10. 测试超时控制

```go
func TestWithTimeout(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    result := make(chan error, 1)
    go func() {
        result <- doSomethingLong(ctx)
    }()
    
    select {
    case err := <-result:
        assert.NoError(t, err)
    case <-ctx.Done():
        t.Fatal("测试超时")
    }
}
```

---

## 📝 测试检查清单

开发新功能时，确保完成以下测试：

- [ ] 单元测试覆盖所有公共方法
- [ ] 测试正常路径（Happy Path）
- [ ] 测试边界条件
- [ ] 测试错误处理
- [ ] 测试并发安全（如适用）
- [ ] 集成测试验证组件交互
- [ ] E2E 测试覆盖核心业务流程
- [ ] 性能基准测试（如适用）
- [ ] 代码覆盖率 ≥ 目标值
- [ ] 所有测试在 CI 中通过

---

## 🔧 常用测试命令汇总

```bash
# 基础测试
make test              # 运行所有测试
make test-unit         # 仅单元测试
make test-coverage     # 生成覆盖率报告
make test-race         # 竞态检测
make test-bench        # 基准测试

# Go 原生命令
go test ./...                          # 所有测试
go test -v ./...                       # 详细输出
go test -short ./...                   # 跳过长时间测试
go test -run TestXxx ./...             # 运行特定测试
go test -race ./...                    # 竞态检测
go test -bench=. -benchmem ./...       # 基准测试
go test -coverprofile=coverage.out ./..# 生成覆盖率
go test -timeout 30s ./...             # 设置超时

# 代码质量
make lint              # 代码检查
make fmt               # 格式化代码
make fmt-check         # 检查格式

# API 测试
curl http://localhost:8080/healthz                 # 健康检查
curl http://localhost:8080/debug/routes | jq       # 路由列表
curl http://localhost:8080/debug/modules | jq      # 模块状态
```

---

## 📚 参考资源

### 官方文档
- [Go Testing Package](https://pkg.go.dev/testing)
- [Testify Documentation](https://github.com/stretchr/testify)
- [gomock Documentation](https://github.com/uber-go/mock)

### 最佳实践
- [Effective Go - Testing](https://go.dev/doc/effective_go#testing)
- [Go Test Comments](https://go.dev/wiki/TestComments)
- [Table Driven Tests](https://go.dev/wiki/TableDrivenTests)

### 工具链
- [golangci-lint](https://golangci-lint.run/)
- [gotestsum](https://github.com/gotestyourself/gotestsum)
- [Codecov](https://about.codecov.io/)

---

## 🆘 故障排查

### 常见问题

#### 1. 测试数据库连接失败

```bash
# 检查数据库是否运行
docker ps | grep mysql

# 检查连接字符串
mysql -h localhost -P 3306 -u root -p

# 确保测试数据库存在
mysql -u root -p -e "CREATE DATABASE IF NOT EXISTS iam_contracts_test;"
```

#### 2. Redis 连接失败

```bash
# 检查 Redis 是否运行
docker ps | grep redis

# 测试连接
redis-cli -h localhost -p 6379 ping
```

#### 3. 竞态检测报告问题

```bash
# 查看详细报告
go test -race -v ./path/to/package

# 常见原因：
# - 共享变量未加锁
# - 并发读写 map
# - 关闭已关闭的 channel
```

#### 4. 测试覆盖率低

```bash
# 查看未覆盖的代码
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# 找出覆盖率最低的文件
go tool cover -func=coverage.out | sort -k3 -n
```

---

**更新日期**: 2025-11-01  
**维护者**: IAM Contracts Team  
**反馈**: 如有问题或建议，请提交 Issue
