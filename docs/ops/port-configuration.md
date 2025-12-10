# IAM Contracts 端口配置说明

## 端口分配规则

### 生产环境端口 (90xx)

| 服务类型 | 端口 | 协议 | 说明 |
|---------|------|------|------|
| HTTP REST API | 9080 | HTTP | 不安全的 HTTP 接口（内网使用） |
| HTTPS REST API | 9444 | HTTPS | 安全的 HTTPS 接口（外网访问） |
| gRPC 服务 | 9090 | gRPC/HTTP2 | gRPC 服务端口（mTLS 保护） |
| gRPC 健康检查 | 9091 | HTTP | gRPC 独立健康检查端口 |

### 开发环境端口 (180xx/190xx)

| 服务类型 | 端口 | 协议 | 说明 |
|---------|------|------|------|
| HTTP REST API | 18081 | HTTP | 开发环境 HTTP 接口 |
| HTTPS REST API | 18441 | HTTPS | 开发环境 HTTPS 接口 |
| gRPC 服务 | 19091 | gRPC/HTTP2 | 开发环境 gRPC 服务 |
| gRPC 健康检查 | 19092 | HTTP | 开发环境 gRPC 健康检查 |

## 配置文件对应关系

### 生产环境 (`configs/apiserver.prod.yaml`)

```yaml
insecure:
  bind-address: 0.0.0.0
  bind-port: 9080        # HTTP REST API

secure:
  bind-address: 0.0.0.0
  bind-port: 9444        # HTTPS REST API
  tls:
    cert-file: /etc/iam-contracts/ssl/yangshujie.com.crt
    private-key-file: /etc/iam-contracts/ssl/yangshujie.com.key

grpc:
  bind-address: 0.0.0.0
  bind-port: 9090        # gRPC 服务
  healthz-port: 9091     # gRPC 健康检查
```

### 开发环境 (`configs/apiserver.dev.yaml`)

```yaml
insecure:
  bind-address: 0.0.0.0
  bind-port: 18081       # HTTP REST API

secure:
  bind-address: 0.0.0.0
  bind-port: 18441       # HTTPS REST API

grpc:
  bind-address: 0.0.0.0
  bind-port: 19091       # gRPC 服务
  healthz-port: 19092    # gRPC 健康检查
```

## Docker 部署

### Dockerfile 暴露端口

```dockerfile
# 生产环境端口
EXPOSE 9080 9444
```

### Docker Compose 端口映射

#### 开发环境 (`build/docker/docker-compose.dev.yml`)

```yaml
services:
  iam-apiserver:
    ports:
      - "18081:18081"   # HTTP REST API
      - "18441:18441"   # HTTPS REST API
      - "19091:19091"   # gRPC 服务
      - "19092:19092"   # gRPC 健康检查
```

#### 生产环境 (Docker Swarm)

```yaml
services:
  iam-apiserver:
    ports:
      - "9080:9080"     # HTTP REST API (内网)
      - "9444:9444"     # HTTPS REST API (外网)
      - "9090:9090"     # gRPC 服务 (内网)
      - "9091:9091"     # gRPC 健康检查 (内网)
```

## 网络访问策略与安全设计

### 🌐 外网访问 (通过 Nginx 反向代理)

```text
外部客户端 (443/HTTPS) 
    ↓
Nginx 反向代理 
    ↓
iam-apiserver:9444 (HTTPS REST API)
```

**安全措施**：

- ✅ TLS 1.2+ 加密传输
- ✅ Nginx 防火墙规则
- ✅ Rate Limiting
- ✅ 只暴露必要的 API 端点

**配置示例** (Nginx):

```nginx
upstream iam_backend {
    server iam-apiserver:9444;
}

server {
    listen 443 ssl http2;
    server_name api.yangshujie.com;
    
    ssl_certificate /data/ssl/certs/yangshujie.com.crt;
    ssl_certificate_key /data/ssl/private/yangshujie.com.key;
    
    location /api/v1/ {
        proxy_pass https://iam_backend;
        proxy_ssl_verify off;
    }
}
```

---

### 🔒 内网访问 (Docker 网络隔离)

#### 1. HTTP REST API (9080)

**用途**：内网服务间高性能调用  
**协议**：HTTP (无 TLS)  
**访问控制**：

- ✅ Docker 网络隔离 (`infra-network`)
- ✅ 不映射到宿主机端口
- ✅ 仅限可信内网服务访问

**适用场景**：

```text
内网服务 A (同 Docker 网络) 
    ↓
iam-apiserver:9080 (HTTP)
```

---

#### 2. gRPC 服务 (9090 - mTLS 保护)

**用途**：服务间 gRPC 调用  
**协议**：gRPC over HTTP/2 (mTLS)  
**安全级别**：🔐 **最高** (双向 TLS 认证)

**mTLS 配置**：

```yaml
grpc:
  mtls:
    enabled: true                    # 启用 mTLS
    require-client-cert: true        # 强制客户端证书
    ca-file: /etc/iam-contracts/grpc/ca/ca-chain.crt
    allowed-ous:                     # 白名单：仅允许特定 OU
      - QS                           # 前端服务
      - Admin                        # 管理服务
      - Ops                          # 运维服务
```

**证书验证流程**：

```text
客户端服务
  ├─ 提供客户端证书 (需包含 CN 和 OU)
  ├─ 服务端验证证书链
  ├─ 检查 OU 是否在白名单
  └─ 验证通过 → 建立连接
```

**访问示例**：

```go
// 客户端需要提供证书
creds, _ := credentials.NewClientTLSFromFile(
    "/path/to/ca.crt",
    "",
)
conn, _ := grpc.Dial(
    "iam-apiserver:9090",
    grpc.WithTransportCredentials(creds),
)
```

**拒绝访问场景**：

- ❌ 无客户端证书
- ❌ 证书过期或无效
- ❌ OU 不在白名单 (`allowed-ous`)
- ❌ 证书未由信任的 CA 签发

---

#### 3. gRPC 健康检查 (9091 - HTTP)

**用途**：监控系统健康检查  
**协议**：HTTP (简单 GET 请求)  
**无需认证**：方便监控系统集成

**访问示例**：

```bash
# Kubernetes Liveness Probe
curl http://iam-apiserver:9091/healthz

# Docker Healthcheck
HEALTHCHECK CMD curl -f http://localhost:9091/healthz
```

---

## 🛡️ 端口安全策略总结

| 端口 | 协议 | 安全级别 | 认证方式 | 访问范围 | 用途 |
|-----|------|---------|---------|---------|------|
| 9080 | HTTP | ⚠️ 低 | 无 | 内网 | 高性能 API 调用 |
| 9444 | HTTPS | 🔒 中 | TLS | 外网 | 客户端访问 |
| 9090 | gRPC | 🔐 高 | mTLS + OU 白名单 | 内网 | 服务间通信 |
| 9091 | HTTP | ⚠️ 低 | 无 | 内网 | 健康检查 |

**安全建议**：

1. ✅ 9080 和 9091 **仅用于内网**，通过 Docker 网络隔离
2. ✅ 9090 **必须启用 mTLS**，严格控制客户端白名单
3. ✅ 9444 通过 **Nginx 代理暴露**，不直接映射宿主机端口
4. ✅ 使用防火墙规则 **禁止外网直接访问** 9080/9090/9091

### 开发环境本地访问

- `18081`: HTTP REST API (开发测试)
- `18441`: HTTPS REST API (开发测试)
- `19091`: gRPC 服务 (开发测试)
- `19092`: gRPC 健康检查 (开发测试)

## 健康检查端点

### REST API 健康检查

```bash
# HTTP
curl http://localhost:9080/healthz

# HTTPS
curl https://localhost:9444/healthz
```

### gRPC 健康检查

```bash
# gRPC Health Protocol
grpc-health-probe -addr=localhost:9091

# HTTP Healthz (独立端口)
curl http://localhost:9091/healthz
```

## 防火墙规则建议

### 生产环境

```bash
# 允许 HTTPS (外网访问)
iptables -A INPUT -p tcp --dport 9444 -j ACCEPT

# 允许内网 HTTP (服务间调用)
iptables -A INPUT -s 10.0.0.0/8 -p tcp --dport 9080 -j ACCEPT

# 允许内网 gRPC (服务间调用)
iptables -A INPUT -s 10.0.0.0/8 -p tcp --dport 9090 -j ACCEPT

# 允许内网健康检查
iptables -A INPUT -s 10.0.0.0/8 -p tcp --dport 9091 -j ACCEPT

# 拒绝其他外网访问
iptables -A INPUT -p tcp --dport 9080 -j DROP
iptables -A INPUT -p tcp --dport 9090 -j DROP
iptables -A INPUT -p tcp --dport 9091 -j DROP
```

## 常见问题

### Q: 为什么有两个端口提供 REST API？

A:

- `9080` (HTTP): 内网服务间调用，性能更好，不需要 TLS 开销
- `9444` (HTTPS): 外网访问，提供 TLS 加密保护

### Q: gRPC 为什么需要独立的健康检查端口？

A:

- gRPC 服务本身需要客户端证书（mTLS）
- 健康检查系统（如 Kubernetes Liveness Probe）通常不支持 mTLS
- 独立的 HTTP 健康检查端口更简单、更通用

### Q: 开发环境端口为什么用 18xxx/19xxx？

A:

- 避免与生产环境端口冲突
- 方便本地同时运行多个环境
- 便于识别（18xxx = dev HTTP/HTTPS, 19xxx = dev gRPC）

## 历史遗留配置清理

以下配置已移除（无实际作用）：

```yaml
# ❌ 已移除 - 未被代码使用
server:
  port: 8080
  port-ssl: 8443
```

实际端口配置应使用：

- `insecure.bind-port`
- `secure.bind-port`
- `grpc.bind-port`
- `grpc.healthz-port`
