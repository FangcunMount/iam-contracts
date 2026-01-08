IAM Suggest 模块接入说明
========================

本说明基于当前 IAM 代码结构（domain/application/infra/interface），提供儿童联想搜索的落地指引与示例代码。

## 能力与接口

- 能力：儿童姓名联想（中文/全拼/简拼前缀），手机号/ID 精确匹配。
- 接口：`GET /api/v1/suggest/child?k=keyword`
  - Header：`Authorization: Bearer <jwt>`（按需开启）
  - Query：`k` 关键词；纯数字走精确匹配；否则前缀联想。
  - Response 示例：
    ```json
    [
      {
        "name": "张三丰",
        "id": 123,
        "mobile": "13800138000,13900139000",
        "weight": 1
      }
    ]
    ```
  - 行为：按 weight 降序，去重（同 id），默认取前 20。

## 配置（viper）

```yaml
suggest:
  enable: true
  data_dir: ./data/suggest        # 可选：快照落盘
  full_sync_cron: "@every 1h"     # 全量周期
  delta_sync_cron: "@every 5m"    # 增量周期（可选）
  max_results: 20
  key_pad_len: 25                 # 通配填充长度
  full_sql: ""                    # 留空使用默认 SQL
  delta_sql: ""                   # 留空使用默认 SQL
```

示例文件：`configs/suggest.dev.yaml`、`configs/suggest.prod.yaml`。

## 目录结构（与 IAM 架构对齐）

```
internal/apiserver/domain/suggest/term.go
internal/apiserver/application/suggest/{config.go,service.go,updater.go}
internal/apiserver/infra/suggest/search/{trie.go,hash.go,store.go}
internal/apiserver/infra/mysql/suggest/loader.go
internal/apiserver/interface/suggest/restful/handler.go
internal/apiserver/container/assembler/suggest.go
```

## 数据模型

```go
// internal/apiserver/domain/suggest/term.go
type Term struct {
    Name   string `json:"name"`
    ID     int64  `json:"id"`
    Mobile string `json:"mobile"`
    Weight int    `json:"weight"`
}
```

## 默认 SQL（按 IAM 表）

全量：
```sql
SELECT
  c.id,
  c.name,
  GROUP_CONCAT(DISTINCT u.phone) AS mobiles,
  1 AS weight
FROM children c
INNER JOIN guardianships g ON g.child_id = c.id AND g.deleted_at IS NULL
INNER JOIN users u ON u.id = g.user_id AND u.deleted_at IS NULL
WHERE c.deleted_at IS NULL
GROUP BY c.id;
```

增量（基于更新时间）：
```sql
SELECT
  c.id,
  c.name,
  GROUP_CONCAT(DISTINCT u.phone) AS mobiles,
  1 AS weight
FROM children c
INNER JOIN guardianships g ON g.child_id = c.id AND g.deleted_at IS NULL
INNER JOIN users u ON u.id = g.user_id AND u.deleted_at IS NULL
WHERE c.deleted_at IS NULL AND GREATEST(c.updated_at, g.updated_at, u.updated_at) > ?
GROUP BY c.id;
```

> 如需区分权重，可在 SQL 中计算 `weight` 字段（如按关系类型或最近活跃时间）。

## 搜索核心

Trie + Hash（精确数字匹配），并发安全通过原子切换 Store：

```go
// internal/apiserver/infra/suggest/search/store.go
store := search.Load(lines)
search.Swap(store)
results := store.Suggest(keyword, maxResults, keyPadLen)
```

- 数字：走 Hash，按权重排序后返回。
- 非数字：前缀通配（'*' 填充至 keyPadLen），支持中文、全拼、简拼。
- 去重：按 ID。

## 应用层

- `application/suggest/config.go`：读取 viper 配置。
- `application/suggest/service.go`：对外查询接口。
- `application/suggest/updater.go`：基于 cron 的全量/增量刷新，支持快照持久化。

## 接口层

`interface/suggest/restful/handler.go` 注册路由：
```go
group := engine.Group("/api/v1/suggest", authMiddleware)
group.GET("/child", h.Child)
```

## 装配与启动

1. 读取 `suggest.*` 配置；`enable=false` 时跳过。
2. `assembler/suggest.go` 初始化 Loader、Service、Updater 并启动 cron。
3. `routers.go` 检查模块后注册 REST 路由，沿用 JWT 中间件（若可用）。
4. 优雅退出：`server.go` 在 Shutdown 时调用 Suggest Cleanup。

## 测试

- 单测示例：`infra/suggest/search/store_test.go`（前缀/拼音/去重/数字排序）。
- 如需集成测试，可加载模拟数据行后调用 HTTP handler。

## 迁移注意

- Disease 字段已移除，输出中不再包含 `disease_name`。
- 目录已与 IAM 统一，不再使用 `modules/suggest` 前缀。
- 运行测试时确保 Go 工具链与 GOROOT 版本一致（此前曾因 1.24.0 vs 1.24.3 不匹配导致失败）。
