# 测试快速参考卡片

> 开发过程中常用的测试命令和模式速查表

## 🚀 快速开始

### 日常测试命令

```bash
# 运行所有测试
make test

# 运行单元测试（快速）
make test-unit
go test -short ./...

# 查看测试覆盖率
make test-coverage

# 竞态检测（发现并发问题）
make test-race

# 基准测试（性能分析）
make test-bench
```

### 运行特定测试

```bash
# 运行特定包的测试
go test -v ./internal/apiserver/domain/authn/... \
           ./internal/apiserver/application/authn/... \
           ./internal/apiserver/infra/... \
           ./internal/apiserver/interface/authn/...

# 运行特定测试函数
go test -v -run TestLogin ./internal/apiserver/application/authn/login/

# 运行匹配模式的测试
go test -v -run TestPassword ./...
```

## 📝 编写测试模板

### 1. 简单单元测试

```go
package domain_test

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestNewUser_Success(t *testing.T) {
    // Arrange - 准备测试数据
    username := "testuser"
    email := "test@example.com"
    
    // Act - 执行被测试的操作
    user, err := NewUser(username, email)
    
    // Assert - 验证结果
    require.NoError(t, err)
    assert.Equal(t, username, user.Username)
    assert.Equal(t, email, user.Email)
}
```

### 2. 表驱动测试（推荐）

```go
func TestPasswordValidation(t *testing.T) {
    tests := []struct {
        name     string
        password string
        wantErr  bool
        errMsg   string
    }{
        {
            name:     "有效密码",
            password: "Valid123!@#",
            wantErr:  false,
        },
        {
            name:     "密码太短",
            password: "short",
            wantErr:  true,
            errMsg:   "长度不足",
        },
        {
            name:     "缺少数字",
            password: "NoNumbers!@#",
            wantErr:  true,
            errMsg:   "必须包含数字",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidatePassword(tt.password)
            
            if tt.wantErr {
                require.Error(t, err)
                assert.Contains(t, err.Error(), tt.errMsg)
            } else {
                require.NoError(t, err)
            }
        })
    }
}
```

### 3. 使用 Mock 测试

```go
func TestUserService_GetUser(t *testing.T) {
    // 创建 Mock 控制器
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()
    
    // 创建 Mock 对象
    mockRepo := mocks.NewMockUserRepository(ctrl)
    
    // 设置期望
    expectedUser := &User{ID: "123", Username: "test"}
    mockRepo.EXPECT().
        FindByID(gomock.Any(), "123").
        Return(expectedUser, nil).
        Times(1)
    
    // 执行测试
    service := NewUserService(mockRepo)
    user, err := service.GetUser(context.Background(), "123")
    
    // 验证结果
    require.NoError(t, err)
    assert.Equal(t, expectedUser, user)
}
```

### 4. HTTP 处理器测试

```go
import (
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestUserHandler_GetUser(t *testing.T) {
    // 创建测试请求
    req := httptest.NewRequest("GET", "/api/v1/users/123", nil)
    w := httptest.NewRecorder()
    
    // 执行处理器
    handler := NewUserHandler(service)
    handler.GetUser(w, req)
    
    // 验证响应
    assert.Equal(t, http.StatusOK, w.Code)
    assert.Contains(t, w.Body.String(), "test")
}
```

### 5. 集成测试（含数据库）

```go
func TestUserRepository_Integration(t *testing.T) {
    // 跳过短模式测试
    if testing.Short() {
        t.Skip("跳过集成测试")
    }
    
    // 设置测试数据库
    db := setupTestDB(t)
    defer cleanupTestDB(t, db)
    
    // 在事务中测试（自动回滚）
    tx, err := db.Begin()
    require.NoError(t, err)
    defer tx.Rollback()
    
    // 执行测试
    repo := NewUserRepository(tx)
    user := &User{Username: "test"}
    err = repo.Save(context.Background(), user)
    
    // 验证结果
    require.NoError(t, err)
    assert.NotEmpty(t, user.ID)
}
```

## 🔧 常用断言

### Testify Assert vs Require

```go
// assert - 失败后继续执行
assert.Equal(t, expected, actual)
assert.NotNil(t, object)
assert.NoError(t, err)
assert.True(t, condition)

// require - 失败后立即停止
require.NoError(t, err)  // 如果 err != nil，后续代码不执行
require.NotNil(t, obj)   // 如果 obj == nil，后续代码不执行
```

### 常用断言方法

```go
// 相等性
assert.Equal(t, expected, actual, "optional message")
assert.NotEqual(t, notExpected, actual)
assert.Same(t, expected, actual)  // 指针相同

// 布尔值
assert.True(t, condition)
assert.False(t, condition)

// Nil 检查
assert.Nil(t, object)
assert.NotNil(t, object)

// 错误检查
assert.NoError(t, err)
assert.Error(t, err)
assert.EqualError(t, err, "expected error message")
assert.ErrorIs(t, err, targetErr)
assert.ErrorAs(t, err, &targetErr)

// 集合
assert.Contains(t, "Hello World", "World")
assert.NotContains(t, slice, element)
assert.Len(t, collection, expectedLength)
assert.Empty(t, collection)
assert.NotEmpty(t, collection)
assert.ElementsMatch(t, expected, actual)  // 忽略顺序

// 数值
assert.Greater(t, actual, expected)
assert.GreaterOrEqual(t, actual, expected)
assert.Less(t, actual, expected)
assert.InDelta(t, expected, actual, delta)  // 浮点数比较

// 字符串
assert.Contains(t, haystack, needle)
assert.Regexp(t, regexp.MustCompile(`\d+`), string)

// Panic
assert.Panics(t, func() { panic("boom") })
assert.NotPanics(t, func() { /* safe code */ })
```

## 🎯 测试场景速查

### 测试正常流程（Happy Path）

```go
func TestCreateUser_Success(t *testing.T) {
    user, err := CreateUser("valid", "valid@email.com", "Valid123!@#")
    require.NoError(t, err)
    assert.NotEmpty(t, user.ID)
}
```

### 测试错误处理

```go
func TestCreateUser_InvalidEmail(t *testing.T) {
    _, err := CreateUser("valid", "invalid-email", "Valid123!@#")
    require.Error(t, err)
    assert.Contains(t, err.Error(), "邮箱格式")
}
```

### 测试边界条件

```go
func TestCreateUser_Boundaries(t *testing.T) {
    tests := []struct {
        name     string
        username string
        wantErr  bool
    }{
        {"最小长度", "abc", false},
        {"太短", "ab", true},
        {"最大长度", strings.Repeat("a", 32), false},
        {"太长", strings.Repeat("a", 33), true},
        {"空字符串", "", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := CreateUser(tt.username, "test@test.com", "Valid123!@#")
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### 测试并发安全

```go
func TestCache_ConcurrentAccess(t *testing.T) {
    cache := NewCache()
    var wg sync.WaitGroup
    
    // 并发写入
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

### 测试超时控制

```go
func TestLongOperation_Timeout(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
    defer cancel()
    
    err := LongOperation(ctx)
    assert.ErrorIs(t, err, context.DeadlineExceeded)
}
```

## 🐛 调试测试

### 打印调试信息

```go
func TestDebug(t *testing.T) {
    user := getUser()
    
    // 使用 t.Log（仅在 -v 时显示）
    t.Logf("User: %+v", user)
    
    // 使用 fmt.Printf（始终显示）
    fmt.Printf("Debug: user=%+v\n", user)
    
    // 临时打印 JSON
    data, _ := json.MarshalIndent(user, "", "  ")
    t.Logf("User JSON:\n%s", data)
}
```

### 只运行一个测试

```bash
# 运行特定测试
go test -v -run TestSpecificTest ./path/to/package

# 更详细的输出
go test -v -run TestSpecificTest ./path/to/package 2>&1 | tee test.log
```

### 使用 Delve 调试器

```bash
# 安装 Delve
go install github.com/go-delve/delve/cmd/dlv@latest

# 调试特定测试
dlv test ./internal/apiserver/domain/authn/authentication -- -test.run TestPassword

# 在代码中设置断点
# import "runtime/debug"
# debug.PrintStack()
```

## 📊 测试覆盖率

### 生成并查看覆盖率

```bash
# 生成覆盖率报告
go test -coverprofile=coverage.out ./...

# 查看总体覆盖率
go tool cover -func=coverage.out | grep total

# 生成 HTML 报告
go tool cover -html=coverage.out -o coverage.html
open coverage.html  # macOS

# 查看特定包的覆盖率
go test -cover ./internal/apiserver/{domain,application,infra,interface}/authn/...
```

### 覆盖率目标

- 🎯 **整体**: ≥ 70%
- 🎯 **领域层**: ≥ 80%
- 🎯 **应用服务**: ≥ 75%
- 🎯 **HTTP 层**: ≥ 60%

## 🔥 性能测试

### 基准测试

```go
func BenchmarkPasswordHashing(b *testing.B) {
    password := "SecurePassword123!"
    
    // 重置计时器（排除准备时间）
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        HashPassword(password)
    }
}

func BenchmarkJWTGeneration(b *testing.B) {
    signer := setupSigner()
    userID := "test-123"
    
    b.ResetTimer()
    b.ReportAllocs()  // 报告内存分配
    
    for i := 0; i < b.N; i++ {
        GenerateJWT(signer, userID)
    }
}
```

### 运行基准测试

```bash
# 运行所有基准测试
go test -bench=. -benchmem ./...

# 运行特定基准测试
go test -bench=BenchmarkPassword -benchmem ./internal/apiserver/domain/authn/authentication/

# 运行多次取平均值
go test -bench=. -benchtime=10s -count=5 ./...

# 对比优化前后
go test -bench=. -benchmem ./... > old.txt
# 修改代码...
go test -bench=. -benchmem ./... > new.txt
benchcmp old.txt new.txt
```

### 压力测试

```bash
# 使用 hey 测试 API
hey -n 1000 -c 10 http://localhost:8080/healthz

# 使用 wrk
wrk -t4 -c100 -d30s http://localhost:8080/api/v1/users

# 使用 vegeta
echo "GET http://localhost:8080/healthz" | vegeta attack -rate=100 -duration=30s | vegeta report
```

## 🐳 Docker 测试环境

### 启动测试数据库

```bash
# 启动开发环境
make docker-dev-up

# 或手动启动
docker run -d \
  --name iam-test-mysql \
  -e MYSQL_ROOT_PASSWORD=REDACTED_DEV_ROOT \
  -e MYSQL_DATABASE=iam_contracts_test \
  -p 3307:3306 \
  mysql:8.0

docker run -d \
  --name iam-test-redis \
  -p 6380:6379 \
  redis:7-alpine redis-server --requirepass REDACTED_REDIS_CACHE
```

### 清理测试环境

```bash
# 停止并删除容器
docker stop iam-test-mysql iam-test-redis
docker rm iam-test-mysql iam-test-redis

# 或使用 make
make docker-dev-down
```

## 🚨 常见问题

### 测试失败排查

```bash
# 1. 查看详细日志
go test -v ./...

# 2. 只运行失败的测试
go test -v -run TestFailedTest ./path/to/package

# 3. 检查数据库连接
mysql -h localhost -P 3306 -u root -p
redis-cli -h localhost -p 6379 ping

# 4. 清理缓存重新测试
go clean -testcache
go test ./...
```

### 测试挂起

```bash
# 设置超时
go test -timeout 30s ./...

# 查看运行中的测试
ps aux | grep "go test"

# 强制终止
killall "go test"
```

### Mock 相关问题

```bash
# 重新生成 Mock
go install go.uber.org/mock/mockgen@latest
go generate ./...

# 检查 Mock 期望
# 在测试中使用 ctrl.Finish() 确保所有期望都被满足
```

## 📚 测试资源

### 项目文档

- [完整测试指南](./TESTING_GUIDE.md)
- [API 测试指南](./API_TESTING_GUIDE.md)
- [开发环境设置](./DEV_ENVIRONMENT_SETUP.md)

### 外部资源

- [Go Testing 官方文档](https://pkg.go.dev/testing)
- [Testify 文档](https://github.com/stretchr/testify)
- [Table Driven Tests](https://go.dev/wiki/TableDrivenTests)

---

**提示**: 将此文档添加到书签，开发时随时查阅！

**快捷键**: `Cmd/Ctrl + F` 搜索你需要的测试模式
