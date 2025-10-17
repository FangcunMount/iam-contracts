// Package driving 定义驱动端口（Driving Ports）
//
// 驱动端口是应用层对领域层的接口，定义了业务用例（Use Cases）。
// 领域服务实现这些接口，应用层通过接口调用领域逻辑，实现应用层与领域层的解耦。
//
// # 端口分类
//
// 1. KeyManagementService - 密钥管理服务
//   - 负责密钥的生命周期管理（创建、激活、宽限、退役、清理）
//   - 用例：管理员手动创建密钥、查询密钥状态、强制退役密钥
//
// 2. KeyRotationService - 密钥轮换服务
//   - 负责密钥的自动轮换（生成新密钥、旧密钥进入宽限期、清理过期密钥）
//   - 用例：定时任务自动轮换、手动触发轮换、查询轮换状态
//
// 3. KeySetPublishService - JWKS 发布服务
//   - 负责构建和发布 JWKS（/.well-known/jwks.json）
//   - 用例：客户端获取公钥、验证缓存、强制刷新缓存
//
// # 调用关系
//
// 应用层（Application Layer）调用这些端口接口：
//   - KeyManagementAppService → KeyManagementService
//   - KeyRotationAppService → KeyRotationService
//   - KeyPublishAppService → KeySetPublishService
//
// # 实现位置
//
// 具体实现位于领域服务层：
//   - domain/jwks/service/key_manager.go
//   - domain/jwks/service/key_rotator.go
//   - domain/jwks/service/keyset_builder.go
//
// # 六边形架构
//
//	┌─────────────────────────────────────────┐
//	│         Application Layer               │
//	│  (Orchestration, Transaction, DTOs)     │
//	└──────────────┬──────────────────────────┘
//	               │ 调用
//	               ↓
//	┌─────────────────────────────────────────┐
//	│         Driving Ports                   │
//	│  (KeyManagementService, etc.)           │
//	└──────────────┬──────────────────────────┘
//	               │ 实现
//	               ↓
//	┌─────────────────────────────────────────┐
//	│         Domain Layer                    │
//	│  (Entities, Value Objects, Services)    │
//	└──────────────┬──────────────────────────┘
//	               │ 依赖
//	               ↓
//	┌─────────────────────────────────────────┐
//	│         Driven Ports                    │
//	│  (KeyRepository, KeyGenerator, etc.)    │
//	└──────────────┬──────────────────────────┘
//	               │ 实现
//	               ↓
//	┌─────────────────────────────────────────┐
//	│      Infrastructure Layer               │
//	│  (MySQL, Redis, Crypto, KMS, etc.)      │
//	└─────────────────────────────────────────┘
package driving
