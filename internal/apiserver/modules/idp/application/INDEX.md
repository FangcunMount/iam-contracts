# IDP 模块应用服务层 - 文档索引

欢迎查看 IDP（Identity Provider）模块应用服务层的完整文档。

## 📚 文档列表

### 1. [快速开始指南](./QUICKSTART.md) 🚀
**适合**: 首次使用者、快速上手

**内容**:
- 5 分钟快速开始
- 基本使用示例
- 常见问题解答
- 完整代码示例

**推荐**: ⭐⭐⭐⭐⭐ 强烈推荐先阅读此文档

---

### 2. [详细设计文档](./README.md) 📖
**适合**: 深入理解架构、开发者

**内容**:
- 六边形架构详解
- 应用服务详细说明
- DTOs 设计
- 设计原则（SOLID）
- 与其他层的关系
- 测试策略
- 最佳实践

**推荐**: ⭐⭐⭐⭐⭐ 必读文档

---

### 3. [架构图文档](./ARCHITECTURE.md) 🏗️
**适合**: 可视化学习者、架构师

**内容**:
- 整体架构图
- 模块结构图
- 依赖关系图
- 数据流图
- 时序图

**推荐**: ⭐⭐⭐⭐ 图形化理解架构

---

### 4. [完成总结](./SUMMARY.md) ✅
**适合**: 项目管理者、了解进度

**内容**:
- 已完成的工作清单
- 文件清单
- 架构特点
- 典型用例流程
- 下一步工作计划

**推荐**: ⭐⭐⭐ 了解项目状态

---

### 5. [使用示例](./examples_test.go) 💻
**适合**: 实践学习者

**内容**:
- 完整的 Go 代码示例
- 各种使用场景
- 错误处理示例
- 性能优化示例
- 安全最佳实践

**推荐**: ⭐⭐⭐⭐ 实践学习必看

---

## 🎯 阅读路径推荐

### 路径 1: 快速上手（30 分钟）
```
1. QUICKSTART.md     (10 分钟)
   ↓
2. examples_test.go  (15 分钟)
   ↓
3. 开始编码         (5 分钟)
```

### 路径 2: 深入理解（2 小时）
```
1. QUICKSTART.md     (10 分钟)
   ↓
2. README.md         (60 分钟)
   ↓
3. ARCHITECTURE.md   (30 分钟)
   ↓
4. examples_test.go  (20 分钟)
```

### 路径 3: 架构学习（1 小时）
```
1. ARCHITECTURE.md   (30 分钟)
   ↓
2. README.md         (20 分钟)
   ↓
3. SUMMARY.md        (10 分钟)
```

---

## 📁 代码结构

```
application/
├── 📄 INDEX.md                  # 本文件 - 文档索引
├── 📄 QUICKSTART.md             # 快速开始指南
├── 📄 README.md                 # 详细设计文档
├── 📄 ARCHITECTURE.md           # 架构图文档
├── 📄 SUMMARY.md                # 完成总结
├── 📄 examples_test.go          # 使用示例
├── 📄 services.go               # 应用服务聚合根
│
├── 📁 wechatapp/                # 微信应用子模块
│   ├── services.go              # 应用服务接口 + DTOs
│   └── services_impl.go         # 应用服务实现
│
└── 📁 wechatsession/            # 微信会话子模块
    ├── services.go              # 应用服务接口 + DTOs
    └── services_impl.go         # 应用服务实现
```

---

## 🔑 核心概念速查

### 应用服务

| 服务 | 职责 | 文档 |
|------|------|------|
| **WechatAppApplicationService** | 微信应用管理（创建、查询） | [README.md](./README.md#wechatappapplicationservice) |
| **WechatAppCredentialApplicationService** | 凭据管理（轮换密钥） | [README.md](./README.md#wechatappcredentialapplicationservice) |
| **WechatAppTokenApplicationService** | 令牌管理（获取、刷新） | [README.md](./README.md#wechatapptokenapplicationservice) |
| **WechatAuthApplicationService** | 微信认证（登录、解密） | [README.md](./README.md#wechatauthuapplicationservice) |

### DTOs

| DTO | 用途 | 文档 |
|-----|------|------|
| **CreateWechatAppDTO** | 创建微信应用输入 | [README.md](./README.md#dtos) |
| **WechatAppResult** | 微信应用结果输出 | [README.md](./README.md#dtos) |
| **LoginWithCodeDTO** | 微信登录输入 | [README.md](./README.md#dtos) |
| **LoginResult** | 登录结果输出 | [README.md](./README.md#dtos) |
| **DecryptPhoneDTO** | 解密手机号输入 | [README.md](./README.md#dtos) |

---

## 💡 常见任务

### 任务 1: 创建微信应用
📖 参考: [QUICKSTART.md - 创建微信应用](./QUICKSTART.md#21-创建微信应用)

### 任务 2: 获取访问令牌
📖 参考: [QUICKSTART.md - 获取访问令牌](./QUICKSTART.md#22-获取访问令牌)

### 任务 3: 微信登录
📖 参考: [QUICKSTART.md - 微信登录](./QUICKSTART.md#23-微信登录)

### 任务 4: 轮换密钥
📖 参考: [QUICKSTART.md - 轮换密钥](./QUICKSTART.md#24-轮换密钥)

### 任务 5: 理解架构
📖 参考: [ARCHITECTURE.md - 整体架构](./ARCHITECTURE.md#整体架构)

### 任务 6: 编写测试
📖 参考: [README.md - 测试策略](./README.md#测试策略)

---

## 🔗 相关链接

### 内部链接
- [领域层文档](../domain/wechatapp/README.md)
- [基础设施层文档](../infra/README.md)
- [接口层文档](../interface/README.md)

### 外部资源
- [六边形架构](https://alistair.cockburn.us/hexagonal-architecture/)
- [DDD 战术设计](https://martinfowler.com/bliki/DomainDrivenDesign.html)
- [SOLID 原则](https://en.wikipedia.org/wiki/SOLID)

---

## ❓ 帮助

### 遇到问题？

1. **查看 FAQ**: [QUICKSTART.md - 常见问题](./QUICKSTART.md#常见问题)
2. **查看示例**: [examples_test.go](./examples_test.go)
3. **阅读详细文档**: [README.md](./README.md)

### 需要更多示例？

参考 [examples_test.go](./examples_test.go)，包含：
- ✅ 基本操作示例
- ✅ 完整业务流程示例
- ✅ 性能优化示例
- ✅ 安全最佳实践示例
- ✅ 错误处理示例

---

## 📝 版本信息

- **版本**: 1.0.0
- **最后更新**: 2025-10-29
- **状态**: ✅ 已完成

---

## 👥 贡献

欢迎贡献代码和文档！

---

## 📄 许可证

参见项目根目录的 LICENSE 文件。
