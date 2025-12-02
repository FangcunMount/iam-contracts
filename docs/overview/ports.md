# 端口配置总览

汇总当前项目在开发/生产环境下主要服务的端口分配及来源。

## 生产环境（Swarm / Overlay）

| 服务 | 容器端口 | 宿主机映射 | 配置来源 |
| --- | --- | --- | --- |
| HTTP | 8080 | 不暴露，ServerA 网关容器通过 overlay 访问 | configs/apiserver.prod.yaml (`server.port`); build/docker/docker-compose.prod.yml（expose） |
| HTTPS | 8443 | 不暴露，TLS 由网关终结 | configs/apiserver.prod.yaml (`server.port-ssl`) |
| gRPC | 9090 | 不暴露，内部调用 | configs/apiserver.prod.yaml (`grpc.bind-port`); build/docker/docker-compose.prod.yml（expose） |
| gRPC Health | 9091 | 同容器内 | configs/apiserver.prod.yaml (`grpc.healthz-port`) |
| Nginx 网关 | 80 / 443 | 80 / 443（ServerA 公网） | 由基础设施网关栈提供 |
| MySQL (RDS) | 3306 | 云 RDS 内网地址 | 阿里云 RDS |
| Redis Cache | 6379 | `redis-cache`（ServerA 容器名） | 基础设施缓存实例 |
| Redis Store | 6379 | 云 Redis 内网地址 | 阿里云 Redis-Store |

说明：生产环境不再在 Compose 内启动 MySQL/Redis/Nginx；IAM 仅加入 `infra-network`，由外部网关代理到容器端口。

## 开发环境

### 本地（Air/Make dev）

| 服务 | 端口 | 配置来源 |
| --- | --- | --- |
| HTTP | 18081 | configs/apiserver.dev.yaml (`server.port`/`insecure.bind-port`) |
| HTTPS | 18441 | configs/apiserver.dev.yaml (`server.port-ssl`/`secure.bind-port`) |
| gRPC | 19091 | configs/apiserver.dev.yaml (`grpc.bind-port`) |
| gRPC Health | 19092 | configs/apiserver.dev.yaml (`grpc.healthz-port`) |

### Docker 开发编排

| 服务 | 容器端口 | 宿主机映射 | 配置来源 |
| --- | --- | --- | --- |
| HTTP | 18081 | 18081 | build/docker/docker-compose.dev.yml |
| HTTPS | 18441 | 18441 | build/docker/docker-compose.dev.yml |
| gRPC | 19091 | 19091 | build/docker/docker-compose.dev.yml |
| MySQL | 3306 | 3306 | build/docker/docker-compose.dev.yml |
| Redis | 6379 | 6379 | build/docker/docker-compose.dev.yml |

说明：开发配置文件期望 Redis Store 在 6380，但开发 compose 仅启动一个 Redis（6379）；若需要分离存储实例，可增加一个 6380 映射的 Redis 容器。
