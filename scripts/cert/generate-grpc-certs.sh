#!/bin/bash
# gRPC mTLS 证书生成脚本
# 用于开发环境快速生成测试证书

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# 默认输出到仓库内的开发证书目录，可通过环境变量 CERT_DIR 覆盖（例如生产机使用 /data/iam-contracts/grpc）
CERT_DIR="${CERT_DIR:-$PROJECT_ROOT/configs/cert/grpc}"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 创建目录结构
create_directories() {
    log_info "Creating directory structure under: $CERT_DIR"
    mkdir -p "$CERT_DIR/ca"
    mkdir -p "$CERT_DIR/server"
    mkdir -p "$CERT_DIR/clients"
}

# 生成根 CA
generate_root_ca() {
    log_info "Generating Root CA..."
    
    # 生成根 CA 私钥
    openssl genrsa -out "$CERT_DIR/ca/root-ca.key" 4096
    
    # 生成根 CA 证书
    openssl req -x509 -new -nodes \
        -key "$CERT_DIR/ca/root-ca.key" \
        -sha256 -days 3650 \
        -out "$CERT_DIR/ca/root-ca.crt" \
        -subj "/C=CN/ST=Shanghai/L=Shanghai/O=FangcunMount/OU=IAM/CN=IAM Root CA (Dev)"
    
    log_info "Root CA generated: $CERT_DIR/ca/root-ca.crt"
}

# 生成中间 CA
generate_intermediate_ca() {
    log_info "Generating Intermediate CA..."
    
    # 生成中间 CA 私钥
    openssl genrsa -out "$CERT_DIR/ca/intermediate-ca.key" 4096
    
    # 生成中间 CA CSR
    openssl req -new \
        -key "$CERT_DIR/ca/intermediate-ca.key" \
        -out "$CERT_DIR/ca/intermediate-ca.csr" \
        -subj "/C=CN/ST=Shanghai/L=Shanghai/O=FangcunMount/OU=IAM/CN=IAM Intermediate CA (Dev)"
    
    # 创建扩展配置
    cat > "$CERT_DIR/ca/intermediate-ca.ext" << EOF
basicConstraints = critical, CA:TRUE, pathlen:0
keyUsage = critical, keyCertSign, cRLSign
subjectKeyIdentifier = hash
authorityKeyIdentifier = keyid:always, issuer
EOF

    # 签发中间 CA 证书
    openssl x509 -req \
        -in "$CERT_DIR/ca/intermediate-ca.csr" \
        -CA "$CERT_DIR/ca/root-ca.crt" \
        -CAkey "$CERT_DIR/ca/root-ca.key" \
        -CAcreateserial \
        -out "$CERT_DIR/ca/intermediate-ca.crt" \
        -days 1825 -sha256 \
        -extfile "$CERT_DIR/ca/intermediate-ca.ext"
    
    # 创建 CA 证书链
    cat "$CERT_DIR/ca/intermediate-ca.crt" "$CERT_DIR/ca/root-ca.crt" > "$CERT_DIR/ca/ca-chain.crt"
    
    # 清理临时文件
    rm -f "$CERT_DIR/ca/intermediate-ca.csr" "$CERT_DIR/ca/intermediate-ca.ext"
    
    log_info "Intermediate CA generated: $CERT_DIR/ca/intermediate-ca.crt"
    log_info "CA chain created: $CERT_DIR/ca/ca-chain.crt"
}

# 生成服务端证书
generate_server_cert() {
    log_info "Generating Server certificate..."
    
    # 生成服务端私钥
    openssl genrsa -out "$CERT_DIR/server/iam-grpc.key" 2048
    
    # 生成服务端 CSR
    openssl req -new \
        -key "$CERT_DIR/server/iam-grpc.key" \
        -out "$CERT_DIR/server/iam-grpc.csr" \
        -subj "/C=CN/ST=Shanghai/L=Shanghai/O=FangcunMount/OU=IAM/CN=iam-grpc.svc"
    
    # 创建 SAN 扩展配置
    cat > "$CERT_DIR/server/iam-grpc.ext" << EOF
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
DNS.4 = iam-grpc
IP.1 = 127.0.0.1
IP.2 = ::1
EOF

    # 签发服务端证书
    openssl x509 -req \
        -in "$CERT_DIR/server/iam-grpc.csr" \
        -CA "$CERT_DIR/ca/intermediate-ca.crt" \
        -CAkey "$CERT_DIR/ca/intermediate-ca.key" \
        -CAcreateserial \
        -out "$CERT_DIR/server/iam-grpc.crt" \
        -days 365 -sha256 \
        -extfile "$CERT_DIR/server/iam-grpc.ext"
    
    # 创建服务端完整证书链
    cat "$CERT_DIR/server/iam-grpc.crt" "$CERT_DIR/ca/intermediate-ca.crt" > "$CERT_DIR/server/iam-grpc-fullchain.crt"
    
    # 清理临时文件
    rm -f "$CERT_DIR/server/iam-grpc.csr" "$CERT_DIR/server/iam-grpc.ext"
    
    log_info "Server certificate generated: $CERT_DIR/server/iam-grpc.crt"
}

# 生成客户端证书
generate_client_cert() {
    local service_name=$1
    local ou=$2
    local description=$3
    
    log_info "Generating client certificate for $service_name ($description)..."
    
    # 生成客户端私钥
    openssl genrsa -out "$CERT_DIR/clients/$service_name.key" 2048
    
    # 生成客户端 CSR
    openssl req -new \
        -key "$CERT_DIR/clients/$service_name.key" \
        -out "$CERT_DIR/clients/$service_name.csr" \
        -subj "/C=CN/ST=Shanghai/L=Shanghai/O=FangcunMount/OU=$ou/CN=$service_name.svc"
    
    # 创建扩展配置
    cat > "$CERT_DIR/clients/$service_name.ext" << EOF
basicConstraints = CA:FALSE
keyUsage = critical, digitalSignature, keyEncipherment
extendedKeyUsage = clientAuth
subjectKeyIdentifier = hash
authorityKeyIdentifier = keyid, issuer
subjectAltName = DNS:$service_name.svc
EOF

    # 签发客户端证书
    openssl x509 -req \
        -in "$CERT_DIR/clients/$service_name.csr" \
        -CA "$CERT_DIR/ca/intermediate-ca.crt" \
        -CAkey "$CERT_DIR/ca/intermediate-ca.key" \
        -CAcreateserial \
        -out "$CERT_DIR/clients/$service_name.crt" \
        -days 365 -sha256 \
        -extfile "$CERT_DIR/clients/$service_name.ext"
    
    # 清理临时文件
    rm -f "$CERT_DIR/clients/$service_name.csr" "$CERT_DIR/clients/$service_name.ext"
    
    log_info "Client certificate generated: $CERT_DIR/clients/$service_name.crt"
}

# 验证证书
verify_certificates() {
    log_info "Verifying certificates..."
    
    # 验证服务端证书
    if openssl verify -CAfile "$CERT_DIR/ca/ca-chain.crt" "$CERT_DIR/server/iam-grpc.crt" > /dev/null 2>&1; then
        log_info "✅ Server certificate is valid"
    else
        log_error "❌ Server certificate verification failed"
    fi
    
    # 验证客户端证书
    for cert in "$CERT_DIR/clients"/*.crt; do
        if [ -f "$cert" ]; then
            service_name=$(basename "$cert" .crt)
            if openssl verify -CAfile "$CERT_DIR/ca/ca-chain.crt" "$cert" > /dev/null 2>&1; then
                log_info "✅ Client certificate '$service_name' is valid"
            else
                log_error "❌ Client certificate '$service_name' verification failed"
            fi
        fi
    done
}

# 显示证书信息
show_cert_info() {
    local cert_file=$1
    log_info "Certificate info for: $cert_file"
    openssl x509 -in "$cert_file" -noout -subject -issuer -dates -ext subjectAltName 2>/dev/null || true
    echo ""
}

# 生成所有证书
generate_all() {
    log_info "=== Generating gRPC mTLS Certificates for Development ==="
    echo ""
    
    create_directories
    
    # 检查是否已存在证书
    if [ -f "$CERT_DIR/ca/root-ca.crt" ]; then
        log_warn "Certificates already exist. Use 'clean' command to remove them first."
        log_warn "Or use 'force' to regenerate."
        if [ "$1" != "force" ]; then
            exit 1
        fi
        log_warn "Forcing regeneration..."
    fi
    
    generate_root_ca
    generate_intermediate_ca
    generate_server_cert
    
    # 生成客户端证书
    generate_client_cert "qs" "QS" "心理健康测评系统"
    generate_client_cert "admin" "Admin" "内部管理工具"
    generate_client_cert "ops" "Ops" "运维工具"
    
    echo ""
    verify_certificates
    
    echo ""
    log_info "=== Certificate Generation Complete ==="
    log_info "Certificates location: $CERT_DIR"
    echo ""
    
    # 显示证书信息
    log_info "Server certificate:"
    show_cert_info "$CERT_DIR/server/iam-grpc.crt"
    
    # 设置权限
    chmod 600 "$CERT_DIR"/**/*.key
    log_info "Private key permissions set to 600"
}

# 清理证书
clean() {
    log_warn "Removing all certificates in $CERT_DIR..."
    rm -rf "$CERT_DIR/ca" "$CERT_DIR/server" "$CERT_DIR/clients"
    log_info "Certificates removed"
}

# 显示帮助
show_help() {
    cat << EOF
gRPC mTLS Certificate Generator

Usage: $0 [command]

Commands:
    generate    Generate all certificates (default)
    force       Force regenerate all certificates
    clean       Remove all certificates
    verify      Verify existing certificates
    info        Show certificate information
    help        Show this help message

Examples:
    $0                  # Generate certificates
    $0 generate         # Generate certificates
    $0 force            # Force regenerate
    $0 clean            # Clean up
    $0 verify           # Verify certificates
EOF
}

# 主函数
main() {
    case "${1:-generate}" in
        generate)
            generate_all
            ;;
        force)
            clean
            generate_all force
            ;;
        clean)
            clean
            ;;
        verify)
            verify_certificates
            ;;
        info)
            for cert in "$CERT_DIR"/**/*.crt; do
                if [ -f "$cert" ]; then
                    show_cert_info "$cert"
                fi
            done
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            log_error "Unknown command: $1"
            show_help
            exit 1
            ;;
    esac
}

main "$@"
