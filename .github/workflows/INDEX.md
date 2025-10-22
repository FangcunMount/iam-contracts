# 📚 GitHub Actions CI/CD 文档索引

欢迎使用 IAM Contracts 项目的 GitHub Actions CI/CD 系统！

## 🚀 快速导航

### 新手入门（从这里开始！）

1. **[快速启动指南](QUICKSTART.md)** ⭐ 推荐首读
   - 5 分钟快速配置
   - 分步骤操作指南
   - 常见问题快速解决

2. **[实施检查清单](CHECKLIST.md)**
   - 完整的实施步骤
   - 测试验证流程
   - 故障排查指南

3. **[实施总结](SUMMARY.md)**
   - 功能特性总览
   - 工作流说明
   - 成果展示

---

### 深入了解

4. **[完整文档](README.md)**
   - 详细的工作流说明
   - 配置指南
   - 最佳实践
   - 故障排查

5. **[流程图与架构](DIAGRAMS.md)**
   - CI/CD 完整流程图
   - 部署策略图
   - 工作流关系图
   - 使用场景示例

6. **[文件清单](MANIFEST.md)**
   - 所有文件详细说明
   - 功能对照表
   - 维护建议

---

### 配置参考

7. **[Secrets 配置模板](secrets.example)**
   - 必需的 Secrets 列表
   - 配置示例
   - 安全注意事项

8. **[状态徽章配置](BADGES.md)**
   - GitHub 徽章示例
   - 自定义样式
   - README 集成方法

---

## 📂 文件结构

```
.github/workflows/
├── 工作流文件 (4个)
│   ├── cicd.yml              # 主 CI/CD 流程
│   ├── server-check.yml      # 服务器健康检查
│   ├── db-ops.yml            # 数据库操作
│   └── ping-runner.yml       # Runner 连通性测试
│
├── 文档文件 (8个)
│   ├── INDEX.md              # 本文件 - 文档索引
│   ├── QUICKSTART.md         # 快速启动（推荐首读）
│   ├── CHECKLIST.md          # 实施检查清单
│   ├── SUMMARY.md            # 实施总结
│   ├── README.md             # 完整文档
│   ├── DIAGRAMS.md           # 流程图
│   ├── MANIFEST.md           # 文件清单
│   └── BADGES.md             # 徽章配置
│
└── 配置模板 (1个)
    └── secrets.example       # Secrets 配置模板
```

---

## 🎯 根据场景选择文档

### 场景 1：我是新用户，第一次使用
**推荐阅读顺序：**
1. [QUICKSTART.md](QUICKSTART.md) - 快速上手
2. [DIAGRAMS.md](DIAGRAMS.md) - 理解流程
3. [CHECKLIST.md](CHECKLIST.md) - 逐步实施

### 场景 2：我需要配置 CI/CD
**推荐阅读顺序：**
1. [secrets.example](secrets.example) - 查看需要的配置
2. [QUICKSTART.md](QUICKSTART.md) - 快速配置步骤
3. [README.md](README.md) - 详细配置说明

### 场景 3：我遇到了问题
**推荐阅读顺序：**
1. [CHECKLIST.md](CHECKLIST.md) - 故障排查部分
2. [README.md](README.md) - 故障排查章节
3. GitHub Actions 工作流日志

### 场景 4：我需要了解工作流细节
**推荐阅读顺序：**
1. [MANIFEST.md](MANIFEST.md) - 工作流功能清单
2. [DIAGRAMS.md](DIAGRAMS.md) - 流程图
3. [README.md](README.md) - 完整说明

### 场景 5：我是团队负责人，需要整体了解
**推荐阅读顺序：**
1. [SUMMARY.md](SUMMARY.md) - 整体总结
2. [DIAGRAMS.md](DIAGRAMS.md) - 架构图
3. [CHECKLIST.md](CHECKLIST.md) - 实施计划

### 场景 6：我需要添加状态徽章
**推荐阅读：**
1. [BADGES.md](BADGES.md) - 徽章配置完整指南

---

## 📊 文档内容对比

| 文档 | 长度 | 难度 | 适合 | 主要内容 |
|------|------|------|------|----------|
| QUICKSTART.md | 5.5 KB | ⭐ 简单 | 新手 | 5分钟快速开始 |
| CHECKLIST.md | 11 KB | ⭐⭐ 中等 | 实施者 | 完整检查清单 |
| SUMMARY.md | 9.8 KB | ⭐⭐ 中等 | 管理者 | 功能总结 |
| DIAGRAMS.md | 19 KB | ⭐⭐ 中等 | 所有人 | 流程图和架构 |
| README.md | 9.6 KB | ⭐⭐⭐ 详细 | 所有人 | 完整文档 |
| MANIFEST.md | 7.0 KB | ⭐⭐⭐ 详细 | 维护者 | 文件清单 |
| BADGES.md | 4.2 KB | ⭐ 简单 | 维护者 | 徽章配置 |
| secrets.example | 3.0 KB | ⭐⭐ 中等 | 运维 | 配置模板 |

---

## 🔄 工作流快速参考

### CI/CD Pipeline (`cicd.yml`)
- **触发：** Push, PR, Tag
- **功能：** Lint → Test → Build → Docker → Deploy
- **环境：** Dev / Staging / Prod

### Server Health Check (`server-check.yml`)
- **触发：** 定时（每小时）+ 手动
- **功能：** API + DB + SSL 检查
- **类型：** Full / Quick / API-only / DB-only

### Database Operations (`db-ops.yml`)
- **触发：** 手动
- **功能：** Health-check / Backup / Migrate / Seed
- **环境：** Dev / Staging / Prod

### Ping Runner (`ping-runner.yml`)
- **触发：** 定时（每天）+ 手动
- **功能：** Runner 系统检查
- **报告：** 系统信息 + 资源状态

---

## 💡 使用技巧

### 快速查找信息

1. **需要配置清单？** → [secrets.example](secrets.example)
2. **需要操作步骤？** → [QUICKSTART.md](QUICKSTART.md) 或 [CHECKLIST.md](CHECKLIST.md)
3. **需要理解流程？** → [DIAGRAMS.md](DIAGRAMS.md)
4. **需要详细说明？** → [README.md](README.md)
5. **需要功能总结？** → [SUMMARY.md](SUMMARY.md) 或 [MANIFEST.md](MANIFEST.md)
6. **需要添加徽章？** → [BADGES.md](BADGES.md)

### 搜索关键词

在文档中搜索以下关键词快速定位：

- `必需` - 找到必需的配置项
- `可选` - 找到可选的配置项
- `触发` - 了解工作流如何触发
- `部署` - 了解部署相关内容
- `故障` 或 `失败` - 找到故障排查内容
- `手动` - 找到手动操作相关内容

---

## 📞 获取帮助

### 按优先级尝试以下方法：

1. **查阅文档**
   - 首先查看 [QUICKSTART.md](QUICKSTART.md)
   - 然后查看 [CHECKLIST.md](CHECKLIST.md) 的故障排查部分

2. **查看日志**
   - GitHub Actions 页面查看工作流运行日志
   - 查找红色 ❌ 标记的失败步骤

3. **参考流程图**
   - 查看 [DIAGRAMS.md](DIAGRAMS.md) 理解流程
   - 确认当前所处的阶段

4. **创建 Issue**
   - 如果以上都不能解决，在仓库创建 Issue
   - 附上详细的错误信息和日志截图

---

## 🎓 学习路径

### 初级（1-2 小时）
1. 阅读 [QUICKSTART.md](QUICKSTART.md)
2. 配置基础 Secrets
3. 运行第一个工作流（Ping Runner）
4. 完成一次 CI 测试

### 中级（3-4 小时）
1. 阅读 [README.md](README.md)
2. 理解 [DIAGRAMS.md](DIAGRAMS.md) 中的流程
3. 完成开发环境部署
4. 运行健康检查工作流

### 高级（1 天）
1. 阅读 [MANIFEST.md](MANIFEST.md)
2. 完成三个环境的部署
3. 配置定时任务和通知
4. 优化工作流性能

### 专家级（持续）
1. 自定义工作流
2. 集成第三方工具
3. 性能优化
4. 安全加固

---

## 🔖 常用链接

### 项目内链接
- [工作流目录](./)
- [项目根目录](../../)
- [API 文档](../../api/)
- [配置文件](../../configs/)

### 外部资源
- [GitHub Actions 官方文档](https://docs.github.com/en/actions)
- [工作流语法参考](https://docs.github.com/en/actions/reference/workflow-syntax-for-github-actions)
- [Docker 文档](https://docs.docker.com/)
- [Go 官方文档](https://golang.org/doc/)

---

## 📝 文档维护

### 更新频率
- **工作流文件：** 根据需求更新
- **文档文件：** 每次重大变更后更新
- **配置模板：** 新增 Secret 时更新

### 版本管理
当前版本：**1.0.0**
最后更新：**2025-10-21**

### 贡献指南
如发现文档问题或有改进建议：
1. 创建 Issue 描述问题
2. 或直接提交 PR 修复
3. 确保文档保持清晰和最新

---

## ✅ 下一步行动

根据你的角色选择下一步：

### 👨‍💻 开发人员
1. 阅读 [QUICKSTART.md](QUICKSTART.md)
2. 了解分支策略和提交规范
3. 尝试创建第一个 PR

### 🔧 运维人员
1. 阅读 [secrets.example](secrets.example)
2. 配置所有必需的 Secrets
3. 完成 [CHECKLIST.md](CHECKLIST.md) 中的测试

### 👔 项目经理
1. 阅读 [SUMMARY.md](SUMMARY.md)
2. 了解 [DIAGRAMS.md](DIAGRAMS.md) 中的流程
3. 制定团队培训计划

### 🎨 文档维护者
1. 阅读 [MANIFEST.md](MANIFEST.md)
2. 了解所有文档的用途
3. 建立文档更新流程

---

**祝你使用愉快！如有任何问题，欢迎随时查阅相关文档或创建 Issue。** 🚀

---

**文档索引版本：** 1.0.0  
**最后更新：** 2025-10-21  
**维护者：** DevOps Team
