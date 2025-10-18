# IAM Contracts 文档中心

欢迎来到 IAM Contracts 项目文档中心！本项目是一个基于 **六边形架构 + DDD + CQRS** 的 IAM（身份与访问管理）系统。

---

## 🎯 快速导航

### 三大核心模块

| 模块 | 说明 | 核心功能 | 文档 |
|------|------|----------|------|
| 👥 **[用户中心](./uc/)** | 用户信息与关系管理 | 用户管理、儿童档案、监护关系 | [架构设计](./uc/README.md) |
| 🔐 **[认证中心](./authn/)** | 身份认证与 Token 管理 | 多渠道登录、JWT、JWKS | [架构设计](./authn/README.md) |
| 🛡️ **[授权中心](./authorization/)** | 权限控制与策略管理 | RBAC、Casbin、策略版本 | [架构设计](./authorization/README.md) |

---

## 🏗️ 系统架构

### 核心文档

**[系统架构总览](./architecture-overview.md)** ⭐

完整的架构设计文档，包括：

- 六边形架构 + DDD + CQRS 详解
- 技术栈选型（Go、Gin、gRPC、GORM、Casbin、Redis）
- 三大模块交互关系
- 目录结构与分层设计
- 部署架构方案

---

## 🚀 快速开始

### 新人入门路径

1. **理解架构** → [系统架构总览](./architecture-overview.md)
   - 了解六边形架构、DDD、CQRS 核心概念
   - 理解项目整体结构

2. **选择模块** → 根据兴趣进入具体模块
   - [用户中心](./uc/README.md) - 用户和监护关系管理
   - [认证中心](./authn/README.md) - JWT 认证与登录
   - [授权中心](./authorization/README.md) - RBAC 权限控制

### 开发环境搭建

```bash
# 1. 克隆项目
git clone https://github.com/fangcun-mount/iam-contracts.git
cd iam-contracts

# 2. 启动基础设施（MySQL + Redis）
cd build/docker/infra
docker-compose up -d

# 3. 安装依赖
go mod download

# 4. 运行测试
go test ./...

# 5. 启动 API Server
go run cmd/apiserver/apiserver.go
```

---

## 💡 设计原则

本项目遵循以下核心设计原则：

1. **六边形架构**: Domain 核心，Interface/Infrastructure 为适配器
2. **DDD 领域驱动**: 聚合根、值对象、领域服务、仓储模式
3. **CQRS 读写分离**: Command 修改状态，Query 只读
4. **依赖倒置**: Domain 定义接口，Infrastructure 实现
5. **租户隔离**: 所有操作基于 TenantID 进行数据隔离

---

## 📖 文档结构

```text
docs/
├── README.md                    # 📚 文档中心首页（本文档）
├── architecture-overview.md     # 🏗️ 系统总体架构 ⭐
│
├── uc/                          # 👥 用户中心
│   ├── README.md                # UC 模块概述
│   ├── DOMAIN_MODELS.md         # 领域模型设计（聚合根、实体、值对象）
│   ├── ARCHITECTURE.md          # 分层架构与 CQRS 实现
│   ├── API_DESIGN.md            # RESTful 和 gRPC API 设计
│   ├── DATA_MODELS.md           # 数据库设计（ER图、表结构）
│   └── BUSINESS_FLOWS.md        # 业务流程详解
│
├── authn/                       # 🔐 认证中心  
│   ├── README.md                # AuthN 架构概述
│   ├── DIRECTORY_STRUCTURE.md   # 目录结构与分层
│   ├── AUTHENTICATION_FLOWS.md  # 认证流程详解
│   ├── TOKEN_MANAGEMENT.md      # Token 生命周期管理
│   ├── JWKS_GUIDE.md            # JWKS 公钥集发布指南 ⭐
│   ├── SECURITY_DESIGN.md       # 安全设计
│   └── API_REFERENCE.md         # API 参考文档
│
└── authorization/               # 🛡️ 授权中心
    ├── README.md                # AuthZ 架构设计 ⭐
    ├── ARCHITECTURE_DIAGRAMS.md # 架构图集
    ├── REDIS_PUBSUB_GUIDE.md    # Redis 策略同步指南
    ├── resources.seed.yaml      # 资源配置数据
    └── policy_init.csv          # 策略初始化数据
```

---

## 🤝 贡献指南

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 创建 Pull Request

## 📞 联系方式

- **项目仓库**: <https://github.com/fangcun-mount/iam-contracts>
- **问题反馈**: <https://github.com/fangcun-mount/iam-contracts/issues>

## 📄 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](../LICENSE) 文件

---

**最后更新**: 2025-10-18  
**版本**: V2.0

欢迎贡献文档和代码！🎉
