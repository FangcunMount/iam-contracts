# gRPC mTLS 证书配置指南

本目录存放 gRPC 服务的 mTLS 证书和密钥文件。

## 目录结构

```
grpc/
├── ca/                          # CA 证书
│   ├── root-ca.crt              # 根 CA 证书
│   ├── intermediate-ca.crt      # 中间 CA 证书
│   └── ca-chain.crt             # CA 证书链（root + intermediate）
├── server/                      # 服务端证书
│   ├── iam-grpc.crt             # IAM gRPC 服务端证书
│   └── iam-grpc.key             # IAM gRPC 服务端私钥
└── clients/                     # 客户端证书
    ├── qs.crt                   # QS 服务客户端证书
    ├── qs.key                   # QS 服务客户端私钥
    ├── admin.crt                # 管理工具客户端证书
    ├── admin.key                # 管理工具客户端私钥
    └── ops.crt                  # 运维工具客户端证书
        ops.key
```

## 证书生成脚本

### 1. 生成根 CA

```bash
# 生成根 CA 私钥
openssl genrsa -out ca/root-ca.key 4096

# 生成根 CA 证书
openssl req -x509 -new -nodes -key ca/root-ca.key -sha256 -days 3650 \
    -out ca/root-ca.crt \
    -subj "/C=CN/ST=Shanghai/L=Shanghai/O=FangcunMount/OU=IAM/CN=IAM Root CA"
```

### 2. 生成中间 CA

```bash
# 生成中间 CA 私钥
openssl genrsa -out ca/intermediate-ca.key 4096

# 生成中间 CA CSR
openssl req -new -key ca/intermediate-ca.key \
    -out ca/intermediate-ca.csr \
    -subj "/C=CN/ST=Shanghai/L=Shanghai/O=FangcunMount/OU=IAM/CN=IAM Intermediate CA"

# 签发中间 CA 证书
openssl x509 -req -in ca/intermediate-ca.csr \
    -CA ca/root-ca.crt -CAkey ca/root-ca.key \
    -CAcreateserial -out ca/intermediate-ca.crt \
    -days 1825 -sha256 \
    -extfile <(cat <<EOF
basicConstraints = critical, CA:TRUE, pathlen:0
keyUsage = critical, keyCertSign, cRLSign
subjectKeyIdentifier = hash
authorityKeyIdentifier = keyid:always, issuer
EOF
)

# 创建 CA 证书链
cat ca/intermediate-ca.crt ca/root-ca.crt > ca/ca-chain.crt
```

### 3. 生成服务端证书（IAM gRPC Server）

```bash
# 生成服务端私钥
openssl genrsa -out server/iam-grpc.key 2048

# 生成服务端 CSR
openssl req -new -key server/iam-grpc.key \
    -out server/iam-grpc.csr \
    -subj "/C=CN/ST=Shanghai/L=Shanghai/O=FangcunMount/OU=IAM/CN=iam-grpc.svc"

# 签发服务端证书（包含 SAN）
openssl x509 -req -in server/iam-grpc.csr \
    -CA ca/intermediate-ca.crt -CAkey ca/intermediate-ca.key \
    -CAcreateserial -out server/iam-grpc.crt \
    -days 365 -sha256 \
    -extfile <(cat <<EOF
basicConstraints = CA:FALSE
keyUsage = critical, digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectKeyIdentifier = hash
authorityKeyIdentifier = keyid, issuer
subjectAltName = @alt_names

[alt_names]
DNS.1 = iam-grpc.svc
DNS.2 = iam-grpc.internal.example.com
DNS.3 = localhost
IP.1 = 127.0.0.1
EOF
)
```

### 4. 生成客户端证书（QS 服务）

```bash
# 生成 QS 客户端私钥
openssl genrsa -out clients/qs.key 2048

# 生成 QS 客户端 CSR
openssl req -new -key clients/qs.key \
    -out clients/qs.csr \
    -subj "/C=CN/ST=Shanghai/L=Shanghai/O=FangcunMount/OU=QS/CN=qs.svc"

# 签发 QS 客户端证书
openssl x509 -req -in clients/qs.csr \
    -CA ca/intermediate-ca.crt -CAkey ca/intermediate-ca.key \
    -CAcreateserial -out clients/qs.crt \
    -days 365 -sha256 \
    -extfile <(cat <<EOF
basicConstraints = CA:FALSE
keyUsage = critical, digitalSignature, keyEncipherment
extendedKeyUsage = clientAuth
subjectKeyIdentifier = hash
authorityKeyIdentifier = keyid, issuer
subjectAltName = @alt_names

[alt_names]
DNS.1 = qs.svc
DNS.2 = qs.internal.example.com
EOF
)
```

### 5. 生成管理工具客户端证书

```bash
# 生成 Admin 客户端私钥
openssl genrsa -out clients/admin.key 2048

# 生成 Admin 客户端 CSR
openssl req -new -key clients/admin.key \
    -out clients/admin.csr \
    -subj "/C=CN/ST=Shanghai/L=Shanghai/O=FangcunMount/OU=Admin/CN=admin.svc"

# 签发 Admin 客户端证书
openssl x509 -req -in clients/admin.csr \
    -CA ca/intermediate-ca.crt -CAkey ca/intermediate-ca.key \
    -CAcreateserial -out clients/admin.crt \
    -days 365 -sha256 \
    -extfile <(cat <<EOF
basicConstraints = CA:FALSE
keyUsage = critical, digitalSignature, keyEncipherment
extendedKeyUsage = clientAuth
subjectKeyIdentifier = hash
authorityKeyIdentifier = keyid, issuer
subjectAltName = DNS:admin.svc
EOF
)
```

## 证书验证

```bash
# 验证服务端证书
openssl verify -CAfile ca/ca-chain.crt server/iam-grpc.crt

# 验证客户端证书
openssl verify -CAfile ca/ca-chain.crt clients/qs.crt

# 查看证书信息
openssl x509 -in server/iam-grpc.crt -text -noout

# 检查证书过期时间
openssl x509 -in server/iam-grpc.crt -noout -enddate
```

## 证书轮换流程

1. **生成新证书**：在旧证书过期前 30 天生成新证书
2. **部署新证书**：先部署到客户端，确保客户端可以验证新旧两个证书
3. **更新服务端**：更新服务端证书
4. **撤销旧证书**：将旧证书添加到 CRL 或从信任列表移除
5. **监控告警**：设置证书过期告警（提前 7 天）

## 配置示例

### 服务端配置 (apiserver.yaml)

```yaml
grpc:
  mtls:
    enabled: true
    cert_file: "./configs/cert/grpc/server/iam-grpc.crt"
    key_file: "./configs/cert/grpc/server/iam-grpc.key"
    ca_file: "./configs/cert/grpc/ca/ca-chain.crt"
    require_client_cert: true
    allowed_services:
      - qs
      - admin
      - ops
```

### 客户端配置 (QS 服务)

```yaml
iam:
  grpc:
    address: "iam-grpc.internal.example.com:9090"
    mtls:
      enabled: true
      cert_file: "/etc/qs/certs/qs.crt"
      key_file: "/etc/qs/certs/qs.key"
      ca_file: "/etc/qs/certs/ca-chain.crt"
      server_name: "iam-grpc.svc"
```

## 安全注意事项

1. **私钥保护**：私钥文件权限设为 600，仅允许服务用户读取
2. **证书存储**：生产环境使用 Vault 或 K8s Secrets 管理证书
3. **定期轮换**：证书有效期不超过 1 年，定期轮换
4. **CRL/OCSP**：配置证书吊销检查
5. **最小权限**：每个服务使用独立的客户端证书
