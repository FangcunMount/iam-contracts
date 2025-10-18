# 用户中心（UC Module）架构设计

> 负责用户生命周期管理、儿童档案管理以及监护关系管理

---

## 📚 文档导航

| 文档 | 说明 | 内容 |
|------|------|------|
| **本文档** | 模块概述 | 设计目标、核心职责、技术特性 |
| [领域模型](./DOMAIN_MODELS.md) | DDD 设计 | 聚合根、实体、值对象详解 |
| [架构设计](./ARCHITECTURE.md) | 分层架构 | 六边形架构、CQRS 实现 |
| [API 设计](./API_DESIGN.md) | 接口文档 | RESTful API、gRPC API |
| [数据模型](./DATA_MODELS.md) | 数据库设计 | ER 图、表结构 |
| [业务流程](./BUSINESS_FLOWS.md) | 流程详解 | 注册、绑定、查询流程 |

---

## 1. 模块概述

用户中心（User Center, UC）是 IAM 平台的核心模块之一，负责管理用户生命周期、儿童档案以及监护关系。

### 1.1 设计目标

- ✅ **领域驱动**: 基于 DDD 战术设计，清晰的领域边界
- ✅ **六边形架构**: 业务逻辑与基础设施完全解耦
- ✅ **CQRS 模式**: 命令与查询职责分离
- ✅ **高内聚低耦合**: 通过端口适配器实现依赖倒置

### 1.2 技术特性

| 特性 | 实现方式 |
|------|---------|
| **事务管理** | Unit of Work (UoW) 模式 |
| **并发控制** | 乐观锁（GORM 版本字段） |
| **数据验证** | 值对象自包含验证 |
| **错误处理** | 统一错误码 + 错误包装 |
| **日志追踪** | 结构化日志 + 请求 ID |

---

## 2. 核心职责

### 2.1 用户管理

- **注册**: 创建新用户账号
- **资料维护**: 更新姓名、联系方式、身份证
- **状态管理**: 激活、停用、封禁

### 2.2 儿童档案管理

- **档案创建**: 注册儿童基本信息
- **信息维护**: 更新姓名、性别、生日、身高体重
- **查重检测**: 基于姓名+生日查找相似儿童

### 2.3 监护关系管理

- **关系授予**: 建立用户与儿童的监护关系
- **关系撤销**: 解除监护权限
- **关系查询**: 查询监护人的所有儿童、儿童的所有监护人

---

## 3. 架构亮点

### 3.1 领域模型

```text
User (聚合根)        Child (聚合根)
     │                    │
     │                    │
     └──► Guardianship ◄──┘
          (聚合根)
```

**三个聚合根**:
- **User**: 用户信息，包含联系方式、身份证等值对象
- **Child**: 儿童档案，包含生日、身高体重等值对象
- **Guardianship**: 监护关系，连接用户和儿童

详见：[领域模型文档](./DOMAIN_MODELS.md)

### 3.2 分层架构

```text
HTTP/gRPC (适配器)
    ↓
Application (应用服务)
    ↓
Domain (领域层)
    ↓
Infrastructure (基础设施)
```

**CQRS 实现**:
- **Command**: 修改状态，使用聚合根和仓储
- **Query**: 只读查询，直接使用查询服务

详见：[架构设计文档](./ARCHITECTURE.md)

### 3.3 API 设计

**RESTful API**:
- `POST /users` - 注册用户
- `GET /users/:id` - 查询用户
- `POST /children` - 注册儿童
- `POST /guardianships` - 授予监护权

**gRPC API**:
- `CreateUser` - 创建用户
- `GetUser` - 获取用户信息
- `ListUserChildren` - 查询用户的所有儿童

详见：[API 设计文档](./API_DESIGN.md)

---

## 4. 快速开始

### 4.1 注册用户

```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{
    "name": "张三",
    "phone": "13800138000",
    "email": "zhangsan@example.com"
  }'
```

### 4.2 注册儿童

```bash
curl -X POST http://localhost:8080/api/v1/children \
  -H "Content-Type: application/json" \
  -d '{
    "name": "张小明",
    "gender": "male",
    "birthday": "2018-05-20"
  }'
```

### 4.3 授予监护权

```bash
curl -X POST http://localhost:8080/api/v1/guardianships \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "usr_123",
    "child_id": "chd_456",
    "relation": "father"
  }'
```

### 4.4 查询用户的所有儿童

```bash
curl http://localhost:8080/api/v1/users/usr_123/children
```

---

## 5. 核心优势

### 5.1 DDD 战术设计

- ✅ **聚合根明确**: User、Child、Guardianship 各自独立
- ✅ **值对象封装**: Phone、Email、IDCard 自包含验证
- ✅ **领域服务**: 查重检测、关系验证等业务逻辑

### 5.2 六边形架构

- ✅ **业务独立**: Domain 层不依赖任何外部框架
- ✅ **端口适配器**: HTTP、gRPC 均可接入
- ✅ **可测试性**: 业务逻辑可独立测试

### 5.3 CQRS 模式

- ✅ **读写分离**: Command 修改，Query 查询
- ✅ **性能优化**: 查询可直接访问数据库，无需加载聚合根
- ✅ **扩展性**: 可独立优化读写路径

---

## 6. 最佳实践

### 6.1 领域层

- ✅ 聚合根负责维护业务不变性
- ✅ 值对象自包含验证逻辑
- ✅ 领域事件用于解耦聚合根

### 6.2 应用层

- ✅ 应用服务编排业务流程
- ✅ 使用 UoW 管理事务边界
- ✅ Command 和 Query 职责分离

### 6.3 适配器层

- ✅ DTO 与领域对象分离
- ✅ HTTP/gRPC 适配器薄层化
- ✅ 统一错误处理和日志记录

---

## 7. 相关文档

- [系统架构总览](../architecture-overview.md) - 整体架构设计
- [认证中心](../authn/README.md) - JWT 认证与登录
- [授权中心](../authorization/README.md) - RBAC 权限控制

---

**最后更新**: 2025-10-18  
**维护团队**: IAM Team
