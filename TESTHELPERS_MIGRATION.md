# Testhelpers 迁移报告（自动生成）

日期: 2025-11-07

目标：将域层测试中重复出现的本地 stub 抽取到 `internal/apiserver/testhelpers`，并尽量把测试迁移为外部测试包（`package foo_test`），以便复用 stub 并避免导入循环。

概览

- 已迁移并使用共享 stub：
  - `internal/apiserver/domain/authz/assignment` — 测试改为外部包并使用 `testhelpers.AssignmentRepoStub`、`testhelpers.RoleRepoStub` 等。
  - `internal/apiserver/domain/uc/user` — 测试已改用 `testhelpers.NewUserRepoStub()` 等；旧的包内本地 helper 已删除。
  - 其它包（如 `uc/child`）无本地重复 stub，因此无需迁移。

- 保留包内本地 stub（但已规范化为 helpers 文件）：
  - `internal/apiserver/domain/uc/guardianship`
    - 原因：该包包含对仓储返回值按调用序列做不同响应的并发场景测试（`seqGuardRepo`），以及一些包内特化行为（例如序列化/并发冲突检测）。这些测试依赖包内特化 stub 的内部细节，简单通用的共享 stub 无法安全替代。
    - 已做的改进：将这些包内专用 stub 及测试辅助函数抽取到 `internal/apiserver/domain/uc/guardianship/guardianship_test_helpers.go`（同包、非导出），以提高可读性并集中维护。

变更验证

- 我在本次迁移后运行了回归测试：

  ```bash
  go test ./... 
  ```

  结果：所有可测试的包均通过（cached/ok），没有发现回归。

建议与下一步

- 对于已保留包内 stub 的包（当前仅 guardianship）：保持其 helpers 文件并在注释中明确说明为何不能迁移到共享 `testhelpers`（并发序列化、顺序响应等）。
- 若未来需要把这些复杂场景也统一管理，可以考虑为共享 stub 增加可配置的行为（例如可注入的响应序列），但这需要仔细设计以避免引入复杂性或耦合。
- 如需，我可以：
  1. 把本次改动打成一个 feature 分支并生成单个 PR；
  2. 进一步运行 `golangci-lint` 并修正风格问题；
  3. 将迁移策略写进项目文档（迁移准则、何时保留包内 stub）。

记录（变更清单）

- 新增： `internal/apiserver/domain/uc/guardianship/guardianship_test_helpers.go`（提取的包内 helpers）
- 删除： `internal/apiserver/domain/uc/user/user_test_helpers_test.go`（已用共享 stubs 替代）
- 修改： `internal/apiserver/domain/authz/assignment/validator_test.go`、`internal/apiserver/domain/authz/assignment/assignment_test.go`（改为外部测试并使用 `testhelpers`）

如果需要更详细的变更 diff 或把改动包装为 PR，我可以继续执行。

TESTHELPERS 迁移说明

=====================

概述
----

为减少重复测试桩（stub）代码，我将多个包中使用的测试桩抽取到 `internal/apiserver/testhelpers`，并在测试中引用该共享包。但直接引用共享包会在某些情况下造成包间的导入循环（import cycle），以及 stub 的方法签名与领域接口不一致的问题。为保证仓库能编译并通过测试，我做了如下修复与迁移工作。

已做变更（最小化修改集）
-------------------------

1. 修正 shared stub 的方法签名以匹配领域接口
   - 文件：`internal/apiserver/testhelpers/stubs.go`
   - 变更摘要：把 `RoleRepoStub.Delete` / `FindByID` 的 id 参数类型从 `idutil.ID` 改为领域层使用的 `meta.ID`，确保实现满足 `role.Repository` 接口。

2. 避免测试导入循环：将受影响测试改为外部测试包
   - 文件：`internal/apiserver/domain/authz/assignment/validator_test.go`
   - 变更摘要：把 `package assignment` 改为 `package assignment_test`，并通过导入 `assignment` 包（`assignment.` 前缀）来引用导出类型/构造函数（例如 `assignment.GrantCommand`, `assignment.NewValidator`）。同时把测试中原来依赖包内未限定符号的部分改为使用 `assignment.` 前缀或使用 `testhelpers` 中的 stub。这样可以让测试安全地导入 `internal/apiserver/testhelpers` 而不会造成循环导入。

为什么要这么改
----------------

- 共享 stub 能减少重复并提高可维护性，但共享包不能和领域包互相导入（会出现 import cycle）。
- 领域接口使用特定类型（例如 `meta.ID`），如果 stub 签名不匹配则会导致编译错误（stub 未实现接口）。
- 使用外部测试包（`xxx_test`）是一种常见做法，既能以"黑盒"方式测试导出的 API，又能避免循环依赖。

影响范围（文件列表）
---------------------

- 新/改文件：
  - `internal/apiserver/testhelpers/stubs.go`（已修改）
  - `internal/apiserver/domain/authz/assignment/validator_test.go`（已修改）

验证
----

- 我在仓库根目录运行了 `go test ./...`，所有包能成功构建并运行测试（输出显示包为 ok / cached，没有新的失败）。

后续建议
--------

1. 审查其它测试对 `internal/apiserver/testhelpers` 的引用（当前仓库仅发现 assignment 的测试需要修改）。如果将来你逐步把更多 stubs 迁移到 `testhelpers`，建议在变更前先检视：
   - 目标 stub 是否依赖领域包的内部类型或导出类型？（尽量只使用导出类型 meta.*）
   - 是否会造成 import cycle？若会，优先把测试改为外部测试包 `pkgname_test` 或把 stub 保留在包内 `_test.go`。

2. 设计建议
   - 将 `testhelpers` 限制为只依赖 `internal/pkg` 之类的基础数据类型（例如 `meta`），不要直接 import 业务逻辑包。若某些 stub 必须依赖领域实体，考虑把该 stub 保留在对应领域包的 `_test.go` 中。
   - 编写一份 `CONTRIBUTING.md` 或在 README 添加一节关于“如何添加/共享测试桩”的说明，规定放置位置和依赖规则，避免未来再次引入循环依赖。

3. 自动化检查（可选）
   - 添加 CI Lint 步骤：运行 `go vet` / `go test`，并在 PR 中强制执行，以便在代码合并前发现导入循环或接口不匹配的问题。

我能为你做的后续工作
----------------------

- 如果你同意，我可以：
  1. 扫描并列出所有可能受影响的测试文件（已完成，当前仅 assignment 一处）;
  2. 把上述建议写入仓库文档（例如补充到 `README.md` 或新增 `TESTHELPERS_MIGRATION.md` — 已创建本文件）；
  3. 按需把 `testhelpers` 的其余 stub 也做严格的签名校验或移动到更合适的位置。

结语
----

已完成最小修复以恢复构建与测试绿灯。若你希望我继续把共享 stub 的责任边界做成更严格的规范（或把 stubs 迁移回每个包以减少耦合），请告诉我优先级，我会继续推进并在每一步运行测试验证。
