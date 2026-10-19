# IAM 迁入腾讯云 serverA

> 状态：已实现 · 2026-10-19 已完成运行切换与用户业务验收；自动生产发布继续暂停。

IAM 从阿里云 serverB 迁至腾讯云 serverA（154.8.167.1 / 100.74.123.45）。沿用原线上镜像、配置、JWKS 私钥和 mTLS 证书，没有发布新业务版本、执行数据库 DDL 或迁移 MySQL/Redis。域名此前已指向新 A，本次无需修改 DNS。

## 运行事实

| 项目 | 结果 |
| --- | --- |
| 容器 / Compose project | `iam-apiserver` / `docker` |
| 二进制 revision | `c8fc1fe61a32056368148e232e96d32763c15ba5` |
| 固定镜像 ID | `sha256:592a4051a8ba2982df55d47c9b82c759f2d85f728ce4a3825fd5b083338e1945` |
| 镜像标签 | `iam-primary-isolated:c8fc1fe61a32056368148e232e96d32763c15ba5`，对应上述本地已验证镜像 |
| 资源 / 端口 | 1 GiB、1 CPU；仅 overlay 内部 9080/9090/9091，无宿主机端口发布 |
| MySQL | 经 serverB 的 `mysql-rds-proxy:3306` 连接原 RDS 私网；schema version 40、dirty=false |
| Redis / 消息 | 原阿里云 Redis DB 0、公网 16380；NSQ 继续使用原配置，未切换 provider |
| 旧实例 | serverB 容器已停止并保留；源 Compose 的 IAM 加入 `rollback-iam-serverb` profile |

先停止 B，再同步最终 JWKS 文件并启动 A，保持单一 IAM 写入进程。新机使用 Docker/宿主机 DNS；原阿里云 VPC DNS 不适用于腾讯云。TLS 只读挂载 CA chain、IAM server certificate 和 IAM server key 三个文件。仅调整新机这两个 IAM server 文件的属主至 2000:2000，内容与源端一致；未修改其他服务证书。

## 验证与验收

- 原 IAM 凭据 MySQL SELECT 1、Redis AUTH/SELECT/PING 成功，NSQ 连通。
- `/readyz` 全部组件为 ok；授权版本 lag 为 0。
- 使用 QS API 客户端证书实际请求 mTLS gRPC health，返回 SERVING；QS 授权版本证明年龄小于 10 秒。
- 公网 HTTPS `/health` 正常；切换前后发布 JWKS 内容一致（1 key），规范 JSON SHA256 为 `b7dbae4549ccbbed97ea30682fd4781eba00aa2c9c2636cdec17f2f32f05e664`。
- collection 两副本 dependency_ready、serve_ready 均为 true。
- A 原六个容器 ID/启动时间/重启次数保持，B 其他六个容器保持；原九个 Prometheus target 为 up。这九个目标不包含 IAM，未新增 IAM scrape target。
- 用户确认小程序重新登录、运营后台登录和权限页面均正常。

本次验收范围不包含新业务版本、全部授权维护操作或未来消息 provider 的切换。私有 inspect、工作 Compose 和密钥含生产凭据，保存在主机 root 专用目录，不进入 Git。

## CI 与维护落点

服务部署、健康检查、SSH 诊断改用仓库 `SVRA_*`，优先公网，部署主机名校验为 `serverA`；私钥优先复用 `SVR_MINI_SSH_KEY`。`PRODUCTION_DEPLOY_PAUSED=true` 保持，合并源码不会发布新的 IAM 业务镜像到生产。

`IAM_RUNTIME_MYSQL_HOST=mysql-rds-proxy` 只覆盖 `cd.yml` 生成的运行配置。数据库备份和 role-model 维护仍通过 `SVRB_*` 在阿里云 B 执行，继续使用原 `MYSQL_HOST` 私网 endpoint；不要把该全局 secret 改成 Docker alias。

迁移时 `/opt/iam/build/docker/docker-compose.prod.yml` 是 mode 0600 的冻结运行导出，保留原环境和固定镜像；不能直接用仓库的下一次发布模板覆盖它。后续正式业务发布仍按独立发布授权及 CI 门禁执行。

## 回退边界

新 A `/root/iam-migration-20261019/` 保存私有源 inspect、权限元数据、源 Compose、基线与核验快照；B 同名 root 目录保存基线和原 Compose。B `/opt/iam`、旧镜像、JWKS、历史日志及 `/opt/backups/iam/database` 均保留。

需要回退时，先停止 A，备份并把 A 当前 JWKS 全量同步到 B，保留期间生成/轮换的 key，再显式启动 B 的旧容器或启用 rollback profile。验证 B 就绪、JWKS 和授权版本后重载新 A 与旧阿里云 A 的 Nginx，核实 `iam-apiserver` alias 只有一个运行实例。不得直接同时启动两端，也不得用旧 JWKS 覆盖新文件。
