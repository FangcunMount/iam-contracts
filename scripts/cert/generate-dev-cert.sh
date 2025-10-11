#!/bin/bash

# 生成开发环境证书脚本
# 用于生成自签名证书，用于HTTPS开发环境

set -e

echo "🔐 生成开发环境证书..."

# 创建证书目录
mkdir -p configs/cert

# 生成私钥和证书
openssl req -x509 \
    -newkey rsa:4096 \
    -keyout configs/cert/web-apiserver.key \
    -out configs/cert/web-apiserver.crt \
    -days 365 \
    -nodes \
    -subj "/C=CN/ST=Beijing/L=Beijing/O=iam contracts/OU=Development/CN=localhost"

echo "✅ 证书生成完成！"
echo "   📁 私钥: configs/cert/web-apiserver.key"
echo "   📁 证书: configs/cert/web-apiserver.crt"
echo ""
echo "💡 提示："
echo "   - 这些是自签名证书，仅用于开发环境"
echo "   - 生产环境请使用正式的SSL证书"
echo "   - 浏览器可能会显示安全警告，这是正常的" 