# suggest：儿童联想搜索

本文回答：`iam-contracts` 当前的 `suggest` 补充能力到底做了什么、运行时是怎么装配和刷新的、搜索行为如何工作，以及今天已经能证明什么、还不能讲成什么。

## 30 秒结论

- `suggest` 不是新的身份主域，而是依附在 `user / guardianship` 数据上的补充查询能力，当前主要提供一个 REST 接口：`GET /api/v1/suggest/child?k=...`。
- 运行主链是：`Router -> suggest/restful.Handler -> application/suggest.Service -> 内存 search.Store`；数据由 `mysql/suggest.Loader` 从 `children + guardianships + users` 拉取，`Updater` 负责启动时全量加载和后续 cron 刷新。
- 查询行为今天已经可以证明 3 点：非数字关键词走中文/全拼/简拼前缀联想，纯数字关键词走孩子 ID / 监护手机号精确匹配，结果按 `weight` 降序并按 `child id` 去重。
- `suggest` 当前没有 gRPC 暴露面，也没有 SDK 子客户端；它更像一个 REST 读侧补充能力，而不是 IAM 的主业务中心。
- `待补证据`：`suggest.v1.yaml` 虽然已存在，也有 runtime route，但当前仍未纳入与其余 REST 合同同级别的 route/schema 比对脚本。

## 重点速查

| 想回答的问题 | 先打开哪里 |
| ---- | ---- |
| 当前 suggest 暴露了什么接口？ | [../03-接口与集成/01-REST契约与接入.md](../03-接口与集成/01-REST契约与接入.md)、[../../api/rest/suggest.v1.yaml](../../api/rest/suggest.v1.yaml) |
| 路由今天如何注册、是否挂认证？ | [../../internal/apiserver/interface/suggest/restful/handler.go](../../internal/apiserver/interface/suggest/restful/handler.go)、[../../internal/apiserver/routers.go](../../internal/apiserver/routers.go) |
| 模块何时初始化、何时启动刷新？ | [../../internal/apiserver/container/assembler/suggest.go](../../internal/apiserver/container/assembler/suggest.go) |
| 数据从哪里来？ | [../../internal/apiserver/infra/mysql/suggest/loader.go](../../internal/apiserver/infra/mysql/suggest/loader.go) |
| 搜索是怎么匹配和排序的？ | [../../internal/apiserver/infra/suggest/search/store.go](../../internal/apiserver/infra/suggest/search/store.go)、[../../internal/apiserver/infra/suggest/search/trie.go](../../internal/apiserver/infra/suggest/search/trie.go)、[../../internal/apiserver/infra/suggest/search/hash.go](../../internal/apiserver/infra/suggest/search/hash.go) |
| 配置项今天是什么？ | [../../internal/apiserver/application/suggest/config.go](../../internal/apiserver/application/suggest/config.go)、[../../configs/apiserver.dev.yaml](../../configs/apiserver.dev.yaml)、[../../configs/suggest.dev.yaml](../../configs/suggest.dev.yaml) |

## 1. 这个模块在系统里的位置

`suggest` 更适合被理解成：

- 一个**补充读侧能力**
- 一个**基于用户域数据派生的联想搜索**
- 一个**只暴露 REST 查询面、没有独立业务写模型的模块**

它今天依赖的仍是：

- `children`
- `guardianships`
- `users`

因此它不是“第四个主业务域”，也不是新的聚合体系；更准确的定位是：它是在 IAM 内部维护、对上层业务更友好的儿童联想搜索能力。

## 2. 当前对外暴露面

### 2.1 REST 接口

今天已暴露的接口是：

- `GET /api/v1/suggest/child?k=keyword`

证据：

- [../../api/rest/suggest.v1.yaml](../../api/rest/suggest.v1.yaml)
- [../../internal/apiserver/interface/suggest/restful/handler.go](../../internal/apiserver/interface/suggest/restful/handler.go)

### 2.2 当前输入 / 输出语义

| 项 | 当前语义 |
| ---- | ---- |
| 查询参数 | `k` 必填 |
| 数字关键词 | 走孩子 ID / 监护手机号精确匹配 |
| 非数字关键词 | 走中文名、全拼、简拼前缀联想 |
| 返回值 | `[]suggest.Term` |
| 返回字段 | `name`、`id`、`mobile`、`weight` |

`Term` 的真实结构见：

- [../../internal/apiserver/domain/suggest/term.go](../../internal/apiserver/domain/suggest/term.go)

### 2.3 认证边界

`已实现`：路由组本身支持注入认证中间件。  
`当前运行时`：`routers.go` 在注册 `suggest` 路由时，如果 `authMiddleware` 已存在，就传入 `AuthRequired()`；否则回退成放行空中间件。

证据：

- [../../internal/apiserver/interface/suggest/restful/handler.go](../../internal/apiserver/interface/suggest/restful/handler.go)
- [../../internal/apiserver/routers.go](../../internal/apiserver/routers.go)

所以更准确的说法是：

- 不要只因为 OpenAPI 标了 `BearerAuth`，就讲成“这一组接口永远强制 JWT”
- 也不要讲成“suggest 默认公开”

今天应写成：

`当前 runtime 路由支持按依赖注入结果挂 JWT 认证；在现行 apiserver 装配下，通常会传入 AuthRequired。`

## 3. 运行主链

当前最重要的运行链是：

```text
HTTP GET /api/v1/suggest/child
  -> suggest/restful.Handler.Child
  -> application/suggest.Service.Suggest
  -> infra/suggest/search.Current().Suggest
  -> 返回 []suggest.Term
```

这一段可以直接回链到：

- [../../internal/apiserver/interface/suggest/restful/handler.go](../../internal/apiserver/interface/suggest/restful/handler.go)
- [../../internal/apiserver/application/suggest/service.go](../../internal/apiserver/application/suggest/service.go)
- [../../internal/apiserver/infra/suggest/search/store.go](../../internal/apiserver/infra/suggest/search/store.go)

有两个值得特别注意的点：

1. 查询不直接打数据库，而是依赖当前活跃的内存 `Store`
2. `Service` 本身很薄，真正的查询行为主要落在 `search.Store`

## 4. 模块初始化与刷新

### 4.1 什么时候初始化

模块初始化由 `SuggestModule.Initialize(...)` 完成：

- 先读 `suggest.*` 配置
- `enable=false` 时直接跳过
- 需要 MySQL 连接
- 创建 `Service`
- 创建 `Loader`
- 创建 `Updater`
- 启动后台刷新

证据：

- [../../internal/apiserver/container/assembler/suggest.go](../../internal/apiserver/container/assembler/suggest.go)

### 4.2 刷新策略

`Updater` 当前做的事情是：

- 启动时先跑一次全量
- 注册全量 cron
- 如果配置了 `delta_sync_cron`，再注册增量 cron
- `ctx.Done()` 或关机时停止调度

证据：

- [../../internal/apiserver/application/suggest/updater.go](../../internal/apiserver/application/suggest/updater.go)

### 4.3 快照持久化

`已实现`：当 `snapshot=true` 且 `data_dir` 非空时，刷新结果会写到 `snapshot.txt`。  
`当前边界`：这只是当前加载数据的快照持久化，不是新的权威存储，也不是增量事件日志。

证据：

- [../../internal/apiserver/application/suggest/updater.go](../../internal/apiserver/application/suggest/updater.go)
- [../../configs/apiserver.dev.yaml](../../configs/apiserver.dev.yaml)

## 5. 数据来源与配置

### 5.1 默认数据源

默认 SQL 来自：

- [../../internal/apiserver/infra/mysql/suggest/loader.go](../../internal/apiserver/infra/mysql/suggest/loader.go)

它当前会把这些表 join 在一起：

- `children`
- `guardianships`
- `users`

默认全量 SQL 的核心意图是：

- 以孩子为结果主体
- 把监护人的手机号聚合成 `mobiles`
- 产出默认 `weight = 1`

### 5.2 增量同步条件

默认增量 SQL 会基于：

- `c.updated_at`
- `g.updated_at`
- `u.updated_at`

做 `GREATEST(...) > ?` 过滤。

这意味着当前 suggest 的“增量刷新”不是基于事件总线，而是基于数据库时间戳轮询。

### 5.3 当前可配置项

今天配置层真正生效的主要是：

| 配置项 | 作用 |
| ---- | ---- |
| `enable` | 是否启用模块 |
| `data_dir` | 快照目录 |
| `full_sync_cron` | 全量刷新周期 |
| `delta_sync_cron` | 增量刷新周期 |
| `max_results` | 返回结果上限 |
| `key_pad_len` | 前缀通配填充长度 |
| `full_sql` / `delta_sql` | 自定义 SQL |
| `snapshot` | 是否落盘快照 |

证据：

- [../../internal/apiserver/application/suggest/config.go](../../internal/apiserver/application/suggest/config.go)
- [../../configs/suggest.dev.yaml](../../configs/suggest.dev.yaml)
- [../../configs/apiserver.dev.yaml](../../configs/apiserver.dev.yaml)

## 6. 搜索行为今天到底怎么工作

### 6.1 数字关键词

`已实现`：纯数字关键词走 `Hash` 精确匹配。

它当前会索引：

- `child id`
- 监护手机号

证据：

- [../../internal/apiserver/infra/suggest/search/hash.go](../../internal/apiserver/infra/suggest/search/hash.go)

### 6.2 非数字关键词

`已实现`：非数字关键词走 `Trie` 前缀 / 通配匹配。

当前会导入三类 key：

- 原始中文名
- 全拼
- 简拼

证据：

- [../../internal/apiserver/infra/suggest/search/trie.go](../../internal/apiserver/infra/suggest/search/trie.go)

### 6.3 排序与去重

今天结果处理链是：

1. 取候选结果
2. 按 `child id` 去重
3. 按 `weight` 降序排序
4. 截断到 `max_results`

证据：

- [../../internal/apiserver/infra/suggest/search/store.go](../../internal/apiserver/infra/suggest/search/store.go)
- [../../internal/apiserver/infra/suggest/search/store_test.go](../../internal/apiserver/infra/suggest/search/store_test.go)

### 6.4 一个容易讲错的点

不要把它讲成“全文检索”或“搜索引擎”。

更准确的说法是：

`当前它是基于内存 Trie + Hash 的前缀联想和精确数字匹配。`

## 7. 当前边界与风险

### 7.1 它不是独立对外协议面

今天 `suggest` 只有 REST 暴露面：

- 没有 gRPC
- 没有 SDK 子客户端
- 没有单独的业务域专题主线

因此它更像补充能力，而不是主集成协议层。

### 7.2 默认权重并不丰富

默认 SQL 里 `weight` 当前固定为 `1`。  
如果业务方希望按关系类型、最近活跃度或别的信号排序，需要通过自定义 SQL 明确补进去。

### 7.3 合同校验链还没完全纳入

`待补证据`：`suggest.v1.yaml` 虽然已存在，也有 runtime route，但当前 `REST` 校验链还没像 `authn / authz / identity / idp` 那样把它纳入同级比对。

更完整的说明见：

- [../03-接口与集成/01-REST契约与接入.md](../03-接口与集成/01-REST契约与接入.md)

## 8. 一句话总结

`suggest` 今天是一项建立在用户域数据之上的补充读侧能力：启动时全量加载、运行时按 cron 刷新、查询时走内存 Trie + Hash，主要解决“按中文名 / 拼音 / 手机号 / 孩子 ID 快速联想儿童”的问题。
