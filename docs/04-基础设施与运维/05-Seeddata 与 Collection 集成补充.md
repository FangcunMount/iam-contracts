# Seeddata 与 Collection 集成补充

本文回答：`iam-contracts` 的 `seed_family` 工具当前如何和 Collection 服务协作创建 testee，相关配置和代码落点在哪里，以及这部分能力今天应该如何被准确表述。

## 30 秒结论

- 这部分能力不是 IAM 主运行面的核心，而是 `cmd/tools/seeddata` 的补充能力：在 `seed_family` 过程中，创建 child 后会尝试调用 Collection API 创建对应的 testee。
- 当前真值主要在 [../../configs/seeddata.yaml](../../configs/seeddata.yaml)、[../../cmd/tools/seeddata/config.go](../../cmd/tools/seeddata/config.go) 和 [../../cmd/tools/seeddata/seed_family.go](../../cmd/tools/seeddata/seed_family.go)。
- `collection_url` 为空时会跳过 Collection 创建；Collection 调用失败会打印 warning，但不会阻断整批 seed_family 流程。
- 这篇文档应被当作“工具链补充说明”，不是对外 API 或主业务链的核心文档。

## 重点速查

| 关注点 | 当前答案 | 真实落点 |
| ---- | ---- | ---- |
| 配置入口 | `configs/seeddata.yaml` | [../../configs/seeddata.yaml](../../configs/seeddata.yaml) |
| 配置结构 | `CollectionURL` 等字段 | [../../cmd/tools/seeddata/config.go](../../cmd/tools/seeddata/config.go) |
| 真实调用链 | `seed_family.go` 里调用 `/testees` | [../../cmd/tools/seeddata/seed_family.go](../../cmd/tools/seeddata/seed_family.go) |
| Collection API 样例合同 | `cmd/tools/seeddata/collection.yaml` | [../../cmd/tools/seeddata/collection.yaml](../../cmd/tools/seeddata/collection.yaml) |
| 失败语义 | warning，不阻断整体 seed | [../../cmd/tools/seeddata/seed_family.go](../../cmd/tools/seeddata/seed_family.go) |

## 1. 当前能力边界

这部分能力今天更准确的定位是：

- 属于 seeddata 工具链
- 不是 `iam-apiserver` 的运行时主能力
- 不是认证 / 授权 / 用户域正文的核心一环

因此它更适合留在基础设施与运维层，作为补充能力说明。

## 2. 当前配置

[../../configs/seeddata.yaml](../../configs/seeddata.yaml) 当前已经包含：

- `collection_url`

[../../cmd/tools/seeddata/config.go](../../cmd/tools/seeddata/config.go) 里也能看到对应字段：

- `CollectionURL string`

这说明 Collection 集成并不是纯文档设想，而是 seeddata 工具当前的真实配置项。

## 3. 当前调用链

[../../cmd/tools/seeddata/seed_family.go](../../cmd/tools/seeddata/seed_family.go) 当前会在创建家庭数据时：

1. 创建 IAM 用户
2. 创建儿童档案
3. 建立 guardianship
4. 调 Collection API 创建 testee

调用时会向 Collection 发送包含 `iam_child_id` 等字段的请求，并使用 `collection_url + "/testees"` 作为目标地址。

## 4. 当前失败语义

当前更重要的不是“怎么配”，而是“失败会怎样”。

代码里当前采取的是：

- Collection 创建失败时打印 warning
- 不终止整个 seed_family 任务

这意味着更准确的口径是：

- Collection 集成是尽力而为的补充能力
- 不是整个 seeddata 流程的硬阻断依赖

## 5. 当前边界

### 已实现

- seeddata 配置里有 Collection URL
- seed_family 里有 testee 创建逻辑
- 仓库内有 Collection API 样例合同可参考

### 待补证据

- 旧文档里关于 `collection_auth`、`admin_token` 的说明，需要继续和当前代码核对；今天更可靠的事实仍应优先来自 `config.go` 和 `seed_family.go`

### 风险边界

- 这部分能力属于工具链补充，不应被讲成 IAM 主运行面的一部分

## 6. 继续往下读

| 文档 | 说明 |
| ---- | ---- |
| [03-命令、契约校验与开发流程.md](./03-命令、契约校验与开发流程.md) | 命令面与 seeddata 相关入口 |
| [../../configs/seeddata.yaml](../../configs/seeddata.yaml) | seeddata 配置真值 |
| [../../cmd/tools/seeddata/seed_family.go](../../cmd/tools/seeddata/seed_family.go) | 真实调用链 |
