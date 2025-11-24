# 端口配置总览

汇总当前项目在开发/生产环境下主要服务的端口分配及来源。

## 生产环境（Docker / 容器内）

| 服务 | 容器端口 | 宿主机映射 | 配置来源 |
| --- | --- | --- | --- |
| HTTP | 8080 | 8080 | configs/apiserver.prod.yaml (`server.port`); build/docker/docker-compose.prod.yml |
| HTTPS | 8443 | 8443（经 nginx 反代） | configs/apiserver.prod.yaml (`server.port-ssl`); nginx 暴露 443 |
| gRPC | 9090 | 9090 | configs/apiserver.prod.yaml (`grpc.bind-port`); build/docker/docker-compose.prod.yml |
| gRPC Health | 9091 | 9091（同容器内） | configs/apiserver.prod.yaml (`grpc.healthz-port`) |
| MySQL | 3306 | 3306 | build/docker/docker-compose.prod.yml（临时内置 MySQL） |
| Redis | 6379 | 6379 | build/docker/docker-compose.prod.yml（临时内置 Redis） |
| Nginx | 80 / 443 | 80 / 443 | build/docker/docker-compose.prod.yml |

说明：gRPC 端口在配置中默认开放，Compose 已映射到宿主机；HTTPS 由容器内 TLS 或 nginx 终结。

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
