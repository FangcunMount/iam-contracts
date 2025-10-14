# 项目目录结构（当前快照）

以下是仓库的树状结构（截取到第二层目录）。把本文件作为快速定位参考；如需更细的视图，可将深度提高到 3 或 4 层并按模块拆分。

```text
iam-contracts/
├── .air.toml
├── .git/
├── .gitignore
├── LICENSE
├── Makefile
├── README.md
├── api/
│   ├── README.md
│   ├── grpc/
│   └── rest/
├── build/
│   └── docker/
│       └── infra/
│           ├── .env
│           └── README.md
├── cmd/
│   └── apiserver/
├── configs/
│   ├── apiserver-simple.yaml
# 项目目录结构（当前快照）

下面是仓库的树状结构（截取到第二层）。把本文件作为快速定位参考；如需更细的视图，可将深度提高到 3 或 4 层并按模块拆分。

```text
iam-contracts/
├── .air.toml
├── .git/
├── .gitignore
├── LICENSE
├── Makefile
├── README.md
├── api/
│   ├── README.md
│   ├── grpc/
│   └── rest/
├── build/
│   └── docker/
│       └── infra/
│           ├── .env
│           └── README.md
├── cmd/
│   └── apiserver/
├── configs/
│   ├── apiserver-simple.yaml
│   ├── apiserver.yaml
│   ├── cert/
│   ├── env/
│   ├── mysql/
│   └── redis/
├── docs/
│   ├── README.md
│   ├── authentication.md
│   ├── code-structure-apiserver.md
│   ├── database-registry.md
│   ├── error-handling.md
│   ├── framework-overview.md
│   ├── hexagonal-container.md
│   ├── iam/
│   └── logging-system.md
├── go.mod
├── go.sum
├── internal/
│   ├── apiserver/
│   │   ├── app.go
│   │   ├── auth.go
│   │   ├── config/
│   │   ├── container/
│   │   │   ├── assembler/
│   │   │   │   ├── auth.go
│   │   │   │   ├── module.go
│   │   │   │   └── user.go
│   │   │   └── container.go
│   │   ├── database.go
│   │   ├── modules/
│   │   │   ├── authn/
│   │   │   │   ├── application/
│   │   │   │   │   ├── account/
│   │   │   │   │   └── uow/
│   │   │   │   ├── domain/
│   │   │   │   │   ├── account/
│   │   │   │   │   └── token/
│   │   │   │   ├── infra/
│   │   │   │   │   └── mysql/
│   │   │   │   └── interface/
│   │   │   │       └── restful/
│   │   │   ├── authz/
│   │   │   └── uc/
│   │   │       ├── application/
│   │   │       ├── domain/
│   │   │       ├── infra/
│   │   │       └── interface/
│   │   ├── options/
│   │   ├── routers.go
│   │   ├── run.go
│   │   └── server.go
│   └── pkg/
│       ├── code/
│       ├── grpcserver/
│       ├── logger/
│       ├── middleware/
│       ├── options/
│       ├── pubsub/ (removed earlier)
│       └── server/
├── pkg/
│   ├── app/
│   ├── auth/
│   ├── core/
│   ├── database/
│   ├── errors/
│   ├── flag/
│   ├── json/
│   ├── log/
│   ├── meta/
│   ├── shutdown/
│   ├── term/
│   ├── util/
│   └── version/
└── scripts/

```

## Domain 层：职责与约定（更清晰的说明）

Domain 层位于各模块的 `domain/` 目录下（例如 `internal/apiserver/modules/authn/domain`）。它表达领域本身的语言与约束，下面把常见概念与工程约定做更明确的说明：

- 聚合根（Aggregate Root）
  - 定义：聚合根是聚合的一颗根实体，对外暴露一致性边界。聚合内部可以包含多个实体和值对象。
  - 放置：在聚合对应的包中（例如 `domain/account` 内的 `account.go`），名字通常为 `Account`、`Order` 等。
  - 职责：拥有修改聚合内部状态的行为方法（比如 `Account.ChangeEmail(...)`、`Account.Lock()`），并维护聚合不变式。

- 实体（Entity）与值对象（Value Object）
  - 实体：有唯一标识（ID），通常映射到数据库行，包含行为或可变状态（例如 `OperationAccount`）。
  - 值对象：按值比较、不可变，如密码策略参数、地址等，通常以小型 struct 表示并放在同聚合包下。

- 领域服务（Domain Service）
  - 使用场景：当某条业务规则跨越多个聚合或不适合放在某个实体/聚合上时，使用领域服务放置纯业务规则。
  - 位置：`domain/service/` 或 `domain/<aggregate>/service/`，命名要能表达业务意图（如 `AccountReconciliationService`）。

- 端口（Ports）与仓储（Repository）
  - 约定：在 `domain/.../port` 下定义抽象接口（仓储、外部系统依赖、事件发布接口等），接口只包含业务所需的方法。
  - 实现：infra 层实现这些接口（例如 `infra/mysql` 提供 `NewRepository(db *gorm.DB)`），应用层通过端口调用仓储。

- 事务与边界
  - 领域对象包含行为与不变式；应用层负责决定事务边界（什么时候启动/提交回滚）并协调多个聚合的交互。

- 代码组织建议（工程实践）
  - 每个聚合一个包（`domain/account`），包内放聚合根文件、实体、VO、以及 `port` 子包或单独文件定义仓储接口。
  - 保持领域对象尽量纯粹：不要引入 infra 依赖（如数据库、http 客户端）到 domain 包。
  - 复杂查询或跨表/跨聚合报表放在应用层或专门的查询服务，不要污染聚合接口。

## 如何保持目录文档同步（建议）

- 在 CI 中添加自动生成脚本：例如 `scripts/gen-structure.sh`，用 `tree` 或 `find` 生成结构并写入 `docs/project-structure.md`。示例：

```sh
# scripts/gen-structure.sh (示例)
#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR=$(dirname "$(dirname "$0")")
cd "$ROOT_DIR"
echo "# Project structure" > docs/project-structure.md
echo "" >> docs/project-structure.md
tree -L 2 -a --noreport | sed 's/^/  /' >> docs/project-structure.md
```

- 我可以帮你：
  - 把上述脚本加入 `scripts/`，并在 README 中增加使用方法；
  - 或者把 CI job（GitHub Actions / GitLab CI）加入一个阶段，在 PR 时自动更新并提交目录快照（需开权限）。

---

需要我现在把 `scripts/gen-structure.sh` 添加到仓库并在 README 中写入使用说明吗？
