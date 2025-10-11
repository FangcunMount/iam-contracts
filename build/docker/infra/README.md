# Docker 基础设施

本项目使用 Docker Compose 来管理基础设施服务，包括 MySQL 和 Redis。

## 服务概览

### MySQL 服务

- **镜像**: web-mysql:latest
- **端口**: 3306
- **数据目录**: `/data/mysql/web/data`
- **日志目录**: `/data/logs/web/mysql`

### Redis 服务

- **镜像**: web-redis:latest
- **端口**: 6379
- **数据目录**: `/data/redis/web/data`
- **日志目录**: `/data/logs/web/redis`

## 快速开始

### 1. 启动所有服务

```bash
cd build/docker/infra
docker-compose up -d
```

### 2. 查看服务状态

```bash
docker-compose ps
```

### 3. 查看服务日志

```bash
# 查看所有服务日志
docker-compose logs

# 查看特定服务日志
docker-compose logs mysql
docker-compose logs redis
```

### 4. 停止所有服务

```bash
docker-compose down
```

## 环境配置

### 环境变量文件

项目使用 `configs/env/config.env` 文件来配置环境变量：

```bash
# MySQL 配置
MYSQL_ROOT_PASSWORD=web_root_2024
MYSQL_DATABASE=web_framework
MYSQL_USER=web_app_user
MYSQL_PASSWORD=web_app_password_2024
MYSQL_PORT=3306

# Redis 配置
REDIS_PASSWORD=web_redis_2024
REDIS_PORT=6379

# Docker 网络配置
DOCKER_NETWORK_NAME=web-network
```

## 数据持久化

### MySQL 数据

- **数据目录**: `/data/mysql/web/data`
- **日志目录**: `/data/logs/web/mysql`

### Redis 数据

- **数据目录**: `/data/redis/web/data`
- **日志目录**: `/data/logs/web/redis`

## 连接信息

### MySQL 连接

- **主机**: localhost
- **端口**: 3306
- **根用户**: root / web_root_2024
- **应用用户**: web_app_user / web_app_password_2024

### Redis 连接

- **主机**: localhost
- **端口**: 6379
- **密码**: web_redis_2024

## 客户端工具

### MySQL 客户端

```bash
# 使用 Docker 进入 MySQL 容器
docker exec -it web-mysql mysql -u root -p

# 或者使用本地 MySQL 客户端
mysql -h localhost -P 3306 -u web_app_user -p web_framework
```

### Redis 客户端

```bash
# 使用 Docker 进入 Redis 容器
docker exec -it web-redis redis-cli -a web_redis_2024

# 或者使用本地 Redis 客户端
redis-cli -h localhost -p 6379 -a web_redis_2024
```

## 健康检查

所有服务都配置了健康检查：

- **MySQL**: 每30秒检查一次
- **Redis**: 每30秒检查一次

## 生产环境

### 生产环境配置

生产环境使用 `configs/env/config.prod.env` 文件：

```bash
# 生产环境 MySQL 配置
MYSQL_ROOT_PASSWORD=web_root_prod_2024
MYSQL_DATABASE=web_framework_prod
MYSQL_USER=web_app_user
MYSQL_PASSWORD=web_app_password_prod_2024

# 生产环境 Redis 配置
REDIS_PASSWORD=web_redis_prod_2024
```

### 数据目录设置

```bash
# 创建数据目录
sudo mkdir -p /data/mysql/web/data /data/redis/web/data
sudo chown -R 1000:1000 /data/mysql/web/data /data/redis/web/data

# 创建日志目录
sudo mkdir -p /data/logs/web/mysql /data/logs/web/redis
sudo chown -R 1000:1000 /data/logs/web/mysql /data/logs/web/redis
```

## 故障排除

### 常见问题

1. **端口冲突**: 如果端口被占用，可以修改 `docker-compose.yml` 中的端口映射
2. **权限问题**: 确保数据目录有正确的权限
3. **内存不足**: 可以调整 `docker-compose.yml` 中的资源限制

### 日志查看

```bash
# 查看所有服务日志
docker-compose logs -f

# 查看特定服务日志
docker-compose logs -f mysql
docker-compose logs -f redis
```

### 数据备份

```bash
# MySQL 备份
docker exec web-mysql mysqldump -u root -p web_framework > backup.sql

# Redis 备份
docker exec web-redis redis-cli -a web_redis_2024 BGSAVE
docker cp web-redis:/data/dump.rdb ./backup.rdb
```
