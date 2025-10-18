# IAM Contracts 文档导航图

```
docs/
│
├── 📚 README.md                                # 文档中心首页（从这里开始）
│
├── 🏗️ architecture/                            # 架构设计
│   ├── framework-overview.md                   # 六边形架构概览
│   ├── hexagonal-container.md                  # DDD 容器化
│   ├── code-structure-apiserver.md             # 代码结构
│   └── project-structure.md                    # 项目结构
│
├── 🔐 authentication/                           # 认证系统 (AuthN)
│   │
│   ├── 📖 核心文档
│   │   ├── authentication-design.md            # DDD 设计 ⭐
│   │   ├── authentication-implementation-summary.md  # 实现总结
│   │   └── ../authn-architecture.md            # 完整架构
│   │
│   ├── 📊 分层文档
│   │   ├── authentication-service-summary.md   # Domain 层
│   │   ├── authentication-infrastructure-summary.md  # Infrastructure 层
│   │   └── authentication-application-summary.md     # Application 层
│   │
│   └── 📘 使用指南
│       ├── authentication-middleware-guide.md  # 中间件使用
│       ├── authentication-integration.md       # 集成方案
│       └── authentication-quick-reference.md   # 快速参考
│
├── 🛡️ authorization/                            # 授权系统 (AuthZ)
│   │
│   ├── 📖 入口文档
│   │   ├── authz-overview.md                   # 授权概览 ⭐ 从这里开始
│   │   ├── INDEX.md                            # 完整导航
│   │   └── REFACTORING_SUMMARY.md              # 项目现状
│   │
│   ├── 📊 架构文档
│   │   ├── README.md                           # 完整架构设计
│   │   ├── DIRECTORY_TREE.md                   # 目录结构详解
│   │   └── ARCHITECTURE_DIAGRAMS.md            # 架构图集（Mermaid）
│   │
│   └── 📦 配置文件
│       ├── resources.seed.yaml                 # 资源目录
│       └── policy_init.csv                     # 策略示例
│
├── 👥 用户中心
│   └── uc-architecture.md                      # UC 模块设计
│
├── 📐 领域设计
│   ├── architecture-overview.md                # 完整架构总览 ⭐
│   └── domain-layer-design-analysis.md         # 领域层深度分析
│
├── 🔧 系统功能
│   ├── database-registry.md                    # 数据库管理
│   ├── error-handling.md                       # 错误处理
│   ├── logging-system.md                       # 日志系统
│   ├── error-code-refactoring.md               # 错误码设计
│   ├── error-code-refactoring-summary.md
│   └── error-code-registration-fix.md
│
└── 📐 iam/                                      # IAM 规范
```

## 🎯 推荐阅读路径

### 🚀 新手入门（快速上手）

```
1. docs/README.md
   ↓
2. docs/architecture/framework-overview.md
   ↓
3. docs/authentication/authentication-design.md
   ↓
4. docs/authorization/authz-overview.md
   ↓
5. docs/authentication/authentication-integration.md
```

### 📚 深入学习（全面掌握）

#### 认证系统路径

```
1. docs/authentication/authentication-design.md          # 领域设计
   ↓
2. docs/authn-architecture.md                            # 完整架构
   ↓
3. docs/authentication/authentication-service-summary.md # Domain 层
   ↓
4. docs/authentication/authentication-infrastructure-summary.md  # Infra 层
   ↓
5. docs/authentication/authentication-application-summary.md     # App 层
   ↓
6. docs/authentication/authentication-middleware-guide.md       # 中间件
   ↓
7. docs/authentication/authentication-integration.md           # 集成使用
```

#### 授权系统路径

```
1. docs/authorization/authz-overview.md                  # 快速入口
   ↓
2. docs/authorization/REFACTORING_SUMMARY.md            # 项目现状
   ↓
3. docs/authorization/README.md                         # 完整架构
   ↓
4. docs/authorization/DIRECTORY_TREE.md                 # 目录详解
   ↓
5. docs/authorization/ARCHITECTURE_DIAGRAMS.md          # 架构图集
   ↓
6. docs/authorization/INDEX.md                          # 完整导航
```

### 🏗️ 架构研究（架构师）

```
1. docs/architecture-overview.md                         # 整体架构
   ↓
2. docs/architecture/framework-overview.md              # 框架设计
   ↓
3. docs/architecture/hexagonal-container.md             # 六边形架构
   ↓
4. docs/domain-layer-design-analysis.md                 # 领域层分析
   ↓
5. docs/uc-architecture.md                              # UC 设计
   ↓
6. docs/authn-architecture.md                           # AuthN 设计
   ↓
7. docs/authorization/README.md                         # AuthZ 设计
```

## 📖 按模块查找

### 需要了解认证？
→ 从 [authentication/authentication-design.md](./authentication/authentication-design.md) 开始

### 需要了解授权？
→ 从 [authorization/authz-overview.md](./authorization/authz-overview.md) 开始

### 需要了解架构？
→ 从 [architecture-overview.md](./architecture-overview.md) 开始

### 需要集成认证？
→ 查看 [authentication/authentication-integration.md](./authentication/authentication-integration.md)

### 需要实现权限控制？
→ 查看 [authorization/README.md](./authorization/README.md)

## 🎨 文档类型说明

- **⭐ 推荐首读**: 核心文档，必读
- **📖 核心文档**: 关键设计文档
- **📊 架构文档**: 架构设计和流程图
- **📘 使用指南**: 实践和集成指南
- **📦 配置文件**: 示例配置和数据

## 💡 提示

1. **新手**: 从 `docs/README.md` 开始，按推荐路径阅读
2. **开发者**: 直接查看对应模块的使用指南
3. **架构师**: 阅读完整的架构设计文档
4. **贡献者**: 先阅读架构文档，再查看具体模块实现

---

**最后更新**: 2025-10-18
