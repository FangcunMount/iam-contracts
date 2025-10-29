# 开发环境证书目录

此目录用于存放开发/测试环境的自签名 SSL/TLS 证书。

## 📁 文件说明

- `web-apiserver.crt` - 自签名证书（公钥）
- `web-apiserver.key` - 私钥
- `openssl.cnf` - OpenSSL 配置文件（可选）

## 🔐 生成证书

### 快速生成

```bash
# 从项目根目录执行
./scripts/cert/generate-dev-cert.sh
```

### 手动生成

```bash
# 生成 RSA 4096 位自签名证书，有效期 365 天
openssl req -x509 \
    -newkey rsa:4096 \
    -keyout web-apiserver.key \
    -out web-apiserver.crt \
    -days 365 \
    -nodes \
    -subj "/C=CN/ST=Beijing/L=Beijing/O=IAM/OU=Development/CN=localhost" \
    -addext "subjectAltName=DNS:localhost,DNS:*.localhost,IP:127.0.0.1,IP:::1"

# 设置权限
chmod 600 web-apiserver.key
chmod 644 web-apiserver.crt
```

## 🔒 安全说明

1. **仅用于开发/测试环境**
   - 这些是自签名证书，不被公共 CA 信任
   - 生产环境必须使用正式 CA 签发的证书

2. **不要提交到版本控制**
   - 此目录已加入 `.gitignore`
   - 私钥文件不应被共享或提交

3. **权限设置**
   - 私钥文件: `chmod 600` (仅所有者可读写)
   - 证书文件: `chmod 644` (所有人可读)

## 📝 证书信息

查看证书详情：

```bash
# 查看完整证书信息
openssl x509 -in web-apiserver.crt -text -noout

# 查看有效期
openssl x509 -in web-apiserver.crt -noout -dates

# 查看主体信息
openssl x509 -in web-apiserver.crt -noout -subject

# 查看 SAN (Subject Alternative Names)
openssl x509 -in web-apiserver.crt -noout -ext subjectAltName
```

## 🌐 支持的域名/IP

默认生成的证书支持：

- `localhost`
- `*.localhost` (通配符)
- `127.0.0.1` (IPv4)
- `::1` (IPv6)

## 🔄 证书续期

自签名证书过期后，重新生成即可：

```bash
# 删除旧证书
rm -f web-apiserver.{crt,key}

# 重新生成
../../scripts/cert/generate-dev-cert.sh
```

## 📚 参考

详细说明请参考：[docs/SSL_CERT_GUIDE.md](../../docs/SSL_CERT_GUIDE.md)
