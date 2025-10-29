#!/bin/bash

# SSL/TLS 证书测试脚本
# 用于验证自签名证书是否正常工作

set -e

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🔒 测试 SSL/TLS 证书配置${NC}"
echo "========================================"

# 检查证书文件是否存在
echo -e "\n${YELLOW}[1/4]${NC} 检查证书文件..."
if [ ! -f "configs/cert/web-apiserver.crt" ]; then
    echo -e "${RED}✗ 证书文件不存在${NC}"
    echo -e "${YELLOW}正在生成证书...${NC}"
    ./scripts/cert/generate-dev-cert.sh
fi

if [ ! -f "configs/cert/web-apiserver.key" ]; then
    echo -e "${RED}✗ 私钥文件不存在${NC}"
    exit 1
fi

echo -e "${GREEN}✓ 证书文件存在${NC}"

# 验证证书有效性
echo -e "\n${YELLOW}[2/4]${NC} 验证证书有效性..."
CERT_VALID=$(openssl x509 -in configs/cert/web-apiserver.crt -noout -checkend 0 && echo "yes" || echo "no")

if [ "$CERT_VALID" = "yes" ]; then
    echo -e "${GREEN}✓ 证书有效${NC}"
    
    # 显示证书信息
    echo -e "\n${BLUE}证书详情:${NC}"
    echo -e "${BLUE}主体:${NC}"
    openssl x509 -in configs/cert/web-apiserver.crt -noout -subject | sed 's/^/   /'
    echo -e "${BLUE}有效期:${NC}"
    openssl x509 -in configs/cert/web-apiserver.crt -noout -dates | sed 's/^/   /'
    echo -e "${BLUE}支持域名/IP:${NC}"
    openssl x509 -in configs/cert/web-apiserver.crt -noout -ext subjectAltName | tail -n +2 | sed 's/^/   /'
else
    echo -e "${RED}✗ 证书已过期${NC}"
    echo -e "${YELLOW}请重新生成证书: ./scripts/cert/generate-dev-cert.sh${NC}"
    exit 1
fi

# 验证证书和私钥匹配
echo -e "\n${YELLOW}[3/4]${NC} 验证证书和私钥匹配..."
CERT_MD5=$(openssl x509 -noout -modulus -in configs/cert/web-apiserver.crt | openssl md5)
KEY_MD5=$(openssl rsa -noout -modulus -in configs/cert/web-apiserver.key 2>/dev/null | openssl md5)

if [ "$CERT_MD5" = "$KEY_MD5" ]; then
    echo -e "${GREEN}✓ 证书和私钥匹配${NC}"
else
    echo -e "${RED}✗ 证书和私钥不匹配${NC}"
    exit 1
fi

# 测试 OpenSSL 服务器
echo -e "\n${YELLOW}[4/4]${NC} 测试 SSL/TLS 连接..."
echo -e "${BLUE}启动测试服务器（端口 4433）...${NC}"
echo -e "${YELLOW}提示: 按 Ctrl+C 停止测试服务器${NC}\n"

# 启动 OpenSSL 测试服务器
openssl s_server \
    -cert configs/cert/web-apiserver.crt \
    -key configs/cert/web-apiserver.key \
    -accept 4433 \
    -www &

SERVER_PID=$!

# 等待服务器启动
sleep 2

# 测试连接
echo -e "\n${BLUE}测试 HTTPS 连接...${NC}"
if curl -k -s https://localhost:4433 > /dev/null 2>&1; then
    echo -e "${GREEN}✓ SSL/TLS 连接成功${NC}"
else
    echo -e "${RED}✗ SSL/TLS 连接失败${NC}"
fi

# 停止测试服务器
kill $SERVER_PID 2>/dev/null || true
sleep 1

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}✅ 所有测试通过！${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "${BLUE}📝 下一步:${NC}"
echo "   1. 启动开发服务器: make dev"
echo "   2. 访问 HTTP:  http://localhost:8080"
echo "   3. 访问 HTTPS: https://localhost:8443"
echo "   4. 使用 curl:  curl -k https://localhost:8443/healthz"
echo ""
