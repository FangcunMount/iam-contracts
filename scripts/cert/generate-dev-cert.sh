#!/bin/bash

# 生成开发环境证书脚本
# 用于生成自签名证书，用于HTTPS开发环境

set -e

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🔐 生成开发环境自签名证书${NC}"
echo "========================================"

# 创建证书目录
CERT_DIR="configs/cert"
mkdir -p "$CERT_DIR"

# 检查是否已存在证书
if [ -f "$CERT_DIR/web-apiserver.crt" ] || [ -f "$CERT_DIR/web-apiserver.key" ]; then
    echo -e "${YELLOW}⚠️  证书文件已存在${NC}"
    echo -e "${YELLOW}是否要覆盖现有证书? [y/N]${NC}"
    read -r response
    if [[ ! "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
        echo -e "${BLUE}取消操作${NC}"
        exit 0
    fi
    rm -f "$CERT_DIR/web-apiserver.crt" "$CERT_DIR/web-apiserver.key"
fi

# 生成私钥和证书（支持多域名）
echo -e "\n${YELLOW}[1/3]${NC} 生成自签名证书（RSA 4096 位）..."
openssl req -x509 \
    -newkey rsa:4096 \
    -keyout "$CERT_DIR/web-apiserver.key" \
    -out "$CERT_DIR/web-apiserver.crt" \
    -days 365 \
    -nodes \
    -subj "/C=CN/ST=Beijing/L=Beijing/O=IAM Contracts/OU=Development/CN=localhost" \
    -addext "subjectAltName=DNS:localhost,DNS:*.localhost,IP:127.0.0.1,IP:::1"

# 设置权限
echo -e "${YELLOW}[2/3]${NC} 设置文件权限..."
chmod 600 "$CERT_DIR/web-apiserver.key"
chmod 644 "$CERT_DIR/web-apiserver.crt"

# 验证证书
echo -e "${YELLOW}[3/3]${NC} 验证证书..."
CERT_INFO=$(openssl x509 -in "$CERT_DIR/web-apiserver.crt" -noout -dates 2>&1)

echo ""
echo -e "${GREEN}✅ 证书生成完成！${NC}"
echo "========================================"
echo -e "${BLUE}📁 证书位置:${NC}"
echo "   私钥: $CERT_DIR/web-apiserver.key"
echo "   证书: $CERT_DIR/web-apiserver.crt"
echo ""
echo -e "${BLUE}� 有效期:${NC}"
echo "$CERT_INFO" | sed 's/^/   /'
echo ""
echo -e "${BLUE}🌐 支持的域名/IP:${NC}"
echo "   • localhost"
echo "   • *.localhost"
echo "   • 127.0.0.1"
echo "   • ::1 (IPv6)"
echo ""
echo -e "${YELLOW}�💡 使用提示:${NC}"
echo "   1. 这是自签名证书，仅用于开发/测试环境"
echo "   2. 生产环境请使用正式的 CA 签发证书"
echo "   3. 浏览器访问时会显示安全警告，这是正常的"
echo "   4. 使用 curl 时需要添加 -k 参数跳过证书验证"
echo ""
echo -e "${YELLOW}📖 查看证书详情:${NC}"
echo "   openssl x509 -in $CERT_DIR/web-apiserver.crt -text -noout"
echo ""
echo -e "${GREEN}配置完成！可以使用 'make dev' 启动服务${NC}"
echo "========================================" 