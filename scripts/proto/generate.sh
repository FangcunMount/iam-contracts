#!/bin/bash

# Proto文件生成脚本
# 用于生成gRPC相关的Go代码

set -e

echo "🔧 生成Proto文件..."

# 设置路径
PROTO_PATH="internal/apiserver/interface/grpc/proto"
OUTPUT_PATH="internal/apiserver/interface/grpc"

# 检查protoc是否安装
if ! command -v protoc &> /dev/null; then
    echo "❌ protoc 未安装，请先安装 Protocol Buffers"
    exit 1
fi

# 检查Go插件是否安装
if ! command -v protoc-gen-go &> /dev/null; then
    echo "❌ protoc-gen-go 未安装，正在安装..."
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
fi

if ! command -v protoc-gen-go-grpc &> /dev/null; then
    echo "❌ protoc-gen-go-grpc 未安装，正在安装..."
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
fi

# 创建输出目录
mkdir -p ${OUTPUT_PATH}

echo "✅ Proto文件生成完成！"
echo "📁 输出目录: ${OUTPUT_PATH}" 