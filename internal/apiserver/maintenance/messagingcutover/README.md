# IAM M3 离线安装入口（验证中）

M3唯一总表由infra的`docs/server/rocketmq-m3-execution.md`维护。本组件属于T25-C2，不代表C2完成或生产批准；不修改业务写链、SDK版本和worker预算。

`iam-maintenance m3-messaging`不会启动API、SDK、Redis、MQ、Relay或业务自主任务。默认是只读预查，直接SELECT/SHOW，不构造可能创建`schema_migrations`的迁移驱动；它将私有原主库身份、真实版本表DDL摘要、已有新增对象DDL/是否非空、当前绑定/协议及两套旧Outbox责任写入新0600计划。不能覆盖既有计划；公开输出只有计划摘要和`cutover_authorized=false`。

两个独立步骤：

1. `--action schema --store-id <冻结store> --writer-epoch <冻结epoch> --export-plan <新0600文件>`仅预览。独立审核原文件摘要后，`--apply --writers-excluded --plan <原文件> --reviewed-sha256 <独立摘要>`显式运行42–45到固定45；不自动绑定/选新协议。初始41及清洁的42–44可重新冻结各自计划，dirty拒绝，不force或向后迁移。
2. 安装后重新生成`--action select`计划并独立审核，再使用同样apply形式。要求两套旧Outbox无非published行，七张新增表除唯一初始outbox协议行外为空。在一个原MySQL事务写`iam.mysql`绑定和CAS选择`transaction/m3-1/generation=2`，不覆盖原绑定，不重新选择/重置已有事务库。提交回复未知保持Unknown，不自动重试。

连接复用原maintenance的`IAM_APISERVER_MYSQL_*`和TLS校验；本命令独占并关闭自有连接池，迁移驱动关闭自己的池不会关闭持有具名互斥的独立连接。GET_LOCK只串行化此离线入口，**不排除业务写者**。`--writers-excluded`只是操作者声明，不是停写证明、启动屏障或生产授权。实际调用必须放在唯一协调目录的受审pre-start-maintenance步骤中，实际目标持有原部署锁，按T29/T30冻结清单暂停授权写HTTP/全部维护工具/自动发布并排空旧链路。不能运行普通部署脚本来代替本入口。

保护候选范围是七个新增协议/业务证据对象和原真实`schema_migrations`，保留AuthN/login/session revocation事实在该范围外。当前入口**不创建保护备份、不执行删除/恢复**，不证明原库备份可恢复；保护/恢复附件和全体旧写者排除仍由C2/T29组合门禁完成。生产实际版本41未核验，前置不符时阻断，不能用夹具或源码head替代T30只读核验。首个新runtime启动命令之后仍禁止旧备份覆盖，遵守infra不可逆启动屏障。

本地仅运行纯/race/编译/lint。`TestM3MessagingOfflineNative`仅GitHub隔离服务运行，缺依赖失败、不skip：使用随机临时schema及最小41夹具验证实际CLI只读预览、拒绝脏版本/错误主库/摘要/新责任、离线DDL、双旧账本排空及绑定/协议共同提交。AuthN保留行在预查后变化且保持，不能据此声称完整生产AuthN/全量恢复验收。
