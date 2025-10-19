# Docker 构建文件

本目录包含 Docker 相关的构建和部署文件。

## 文件说明

### Dockerfile
多阶段构建的 Docker 镜像配置：
- **构建阶段**：使用 golang:1.24-alpine 编译 Go 应用
- **运行阶段**：使用 alpine:3.19 最小化镜像
- **特性**：
  - 非 root 用户运行（安全）
  - 健康检查配置
  - 时区设置（Asia/Shanghai）
  - 多架构支持（linux/amd64）

### docker-compose.yml
Docker Compose 编排配置，包含：
- **iam-apiserver**：IAM API 服务器
- **mysql**：MySQL 8.0 数据库（可选，如果 infra 项目未提供）
- **redis**：Redis 7 缓存（可选，如果 infra 项目未提供）

适合本地开发和测试环境使用。

### .dockerignore
Docker 构建忽略文件，减小构建上下文大小：
- 排除构建产物（bin/, tmp/, logs/）
- 排除 IDE 文件（.vscode/, .idea/）
- 排除文档和配置文件
- 排除测试覆盖率文件

## 使用方法

### 方式 1：使用 Makefile（推荐）

```bash
# 构建镜像
make docker-build

# 运行容器
make docker-run

# 使用 Docker Compose 启动所有服务
make docker-compose-up

# 查看日志
make docker-compose-logs

# 停止服务
make docker-compose-down

# 清理镜像和容器
make docker-clean
```

### 方式 2：直接使用 Docker 命令

```bash
# 构建镜像
docker build \
  --build-arg VERSION=v1.0.0 \
  --build-arg BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S') \
  --build-arg GIT_COMMIT=$(git rev-parse --short HEAD) \
  -f build/docker/Dockerfile \
  -t iam-contracts:latest \
  .

# 运行容器
docker run -d \
  --name iam-apiserver \
  -p 8080:8080 \
  -v $(pwd)/configs:/app/configs:ro \
  -v $(pwd)/logs:/app/logs \
  iam-contracts:latest

# 查看日志
docker logs -f iam-apiserver

# 停止容器
docker stop iam-apiserver
docker rm iam-apiserver
```

### 方式 3：使用 Docker Compose

```bash
# 启动所有服务
docker-compose -f build/docker/docker-compose.yml up -d

# 查看状态
docker-compose -f build/docker/docker-compose.yml ps

# 查看日志
docker-compose -f build/docker/docker-compose.yml logs -f iam-apiserver

# 停止服务
docker-compose -f build/docker/docker-compose.yml down

# 重启服务
docker-compose -f build/docker/docker-compose.yml restart iam-apiserver
```

## 环境变量配置

可以通过环境变量覆盖配置：

```bash
# 方式 1：命令行传递
docker run -d \
  -e MYSQL_HOST=your-db-host \
  -e MYSQL_PORT=3306 \
  -e REDIS_HOST=your-redis-host \
  -e REDIS_PORT=6379 \
  iam-contracts:latest

# 方式 2：使用 .env 文件（Docker Compose）
# 创建 .env 文件
cat > .env <<EOF
MYSQL_HOST=mysql
MYSQL_PORT=3306
MYSQL_DATABASE=iam
MYSQL_USER=iam
MYSQL_PASSWORD=iam123
REDIS_HOST=redis
REDIS_PORT=6379
EOF

# 启动（自动读取 .env 文件）
docker-compose -f build/docker/docker-compose.yml up -d
```

## 健康检查

容器包含健康检查配置，会定期检查服务状态：

```bash
# 查看容器健康状态
docker inspect --format='{{.State.Health.Status}}' iam-apiserver

# 手动执行健康检查
docker exec iam-apiserver curl -f http://localhost:8080/healthz
```

## 日志管理

### 容器日志

```bash
# 查看实时日志
docker logs -f iam-apiserver

# 查看最近 100 行
docker logs --tail 100 iam-apiserver

# 查看带时间戳的日志
docker logs -t iam-apiserver
```

### 应用日志

应用日志会输出到挂载的日志目录：

```bash
# 查看应用日志
tail -f logs/apiserver.log
```

## 数据持久化

### 使用 Docker Volume

```bash
# 创建数据卷
docker volume create iam-data

# 运行容器并挂载数据卷
docker run -d \
  --name iam-apiserver \
  -v iam-data:/app/data \
  -v $(pwd)/configs:/app/configs:ro \
  iam-contracts:latest
```

### 使用主机目录

```bash
# 挂载主机目录
docker run -d \
  --name iam-apiserver \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/configs:/app/configs:ro \
  -v $(pwd)/logs:/app/logs \
  iam-contracts:latest
```

## 网络配置

### 使用自定义网络

```bash
# 创建网络
docker network create iam-network

# 运行容器并连接到网络
docker run -d \
  --name iam-apiserver \
  --network iam-network \
  -p 8080:8080 \
  iam-contracts:latest
```

Docker Compose 会自动创建网络（已在 docker-compose.yml 中配置）。

## 故障排查

### 容器无法启动

```bash
# 查看容器日志
docker logs iam-apiserver

# 查看容器详细信息
docker inspect iam-apiserver

# 进入容器调试
docker exec -it iam-apiserver sh
```

### 端口冲突

```bash
# 检查端口占用
lsof -i :8080
# 或
netstat -tlnp | grep 8080

# 使用不同端口
docker run -d -p 8081:8080 iam-contracts:latest
```

### 配置文件问题

```bash
# 验证配置文件挂载
docker exec iam-apiserver ls -la /app/configs

# 查看配置文件内容
docker exec iam-apiserver cat /app/configs/apiserver.yaml

# 手动测试启动
docker exec -it iam-apiserver /app/apiserver --config=/app/configs/apiserver.yaml
```

## 生产环境注意事项

1. **安全配置**：
   - 使用非 root 用户运行（已配置）
   - 限制容器资源（CPU、内存）
   - 定期更新基础镜像

2. **资源限制**：
   ```bash
   docker run -d \
     --cpus=2 \
     --memory=2g \
     --memory-swap=2g \
     iam-contracts:latest
   ```

3. **重启策略**：
   ```bash
   docker run -d \
     --restart=unless-stopped \
     iam-contracts:latest
   ```

4. **日志轮转**：
   ```bash
   docker run -d \
     --log-driver json-file \
     --log-opt max-size=10m \
     --log-opt max-file=3 \
     iam-contracts:latest
   ```

5. **监控和告警**：
   - 集成 Prometheus 监控
   - 配置健康检查告警
   - 监控容器资源使用

## 多架构支持

如需构建多架构镜像（如 ARM64）：

```bash
# 创建构建器
docker buildx create --use

# 构建多架构镜像
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --build-arg VERSION=v1.0.0 \
  -f build/docker/Dockerfile \
  -t iam-contracts:latest \
  --push \
  .
```

## 相关文档

- [部署总览](../../docs/DEPLOYMENT.md) - 所有部署方式说明
- [Jenkins 部署](../../docs/JENKINS_QUICKSTART.md) - CI/CD 自动化部署
- [主 README](../../README.md) - 项目概述

## 技术支持

如有问题，请参考：
- [故障排查指南](../../docs/DEPLOYMENT.md#故障排查)
- [GitHub Issues](https://github.com/FangcunMount/iam-contracts/issues)
