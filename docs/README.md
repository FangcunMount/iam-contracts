# iam contracts 使用说明

本文档详细介绍了基于六边形架构的Go Web框架的设计理念、核心组件和使用方法。

## 📚 文档目录

- [框架设计总览](./framework-overview.md) - 整体架构设计理念
- [六边形架构容器设计](./hexagonal-container.md) - 依赖注入容器设计
- [数据库注册器设计](./database-registry.md) - 数据库连接管理
- [日志模块设计](./logging-system.md) - 日志系统使用指南
- [认证模块设计](./authentication.md) - 认证授权系统
- [统一异常处理](./error-handling.md) - 错误处理机制

## 🚀 快速开始

### 安装依赖

```bash
go mod download
```

### 运行示例

```bash
# 使用简化配置运行
go run cmd/apiserver/apiserver.go --config=configs/apiserver-simple.yaml

# 使用完整配置运行
go run cmd/apiserver/apiserver.go --config=configs/apiserver.yaml
```

### 测试API

```bash
# 健康检查
curl http://localhost:8080/health

# 获取服务信息
curl http://localhost:8080/api/v1/public/info
```

## 🏗️ 核心特性

- **六边形架构**: 清晰的领域驱动设计
- **依赖注入**: 灵活的模块组装
- **数据库抽象**: 支持MySQL和Redis
- **统一日志**: 结构化日志记录
- **认证授权**: JWT和Basic认证
- **错误处理**: 统一的异常处理机制

## 📖 更多信息

详细的使用说明请参考各个模块的文档。如有问题，请查看[故障排除](./troubleshooting.md)章节。
