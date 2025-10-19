#!/bin/bash

# ============================================================================
# IAM Contracts 数据库重置脚本
# ============================================================================
# 功能: 删除并重新创建数据库，用于开发环境快速重置
# 警告: 此操作将删除所有数据，请谨慎使用！
# 使用: ./reset-db.sh [options]
# ============================================================================

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 默认配置
DB_HOST="${DB_HOST:-127.0.0.1}"
DB_PORT="${DB_PORT:-3306}"
DB_USER="${DB_USER:-root}"
DB_PASSWORD="${DB_PASSWORD:-}"
DB_NAME="${DB_NAME:-iam_contracts}"

# 脚本目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ============================================================================
# 工具函数
# ============================================================================

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_banner() {
    echo ""
    echo -e "${RED}============================================${NC}"
    echo -e "${RED}  IAM Contracts - 数据库重置工具${NC}"
    echo -e "${RED}  ⚠️  警告: 此操作将删除所有数据！${NC}"
    echo -e "${RED}============================================${NC}"
    echo ""
}

show_help() {
    cat << EOF
使用方法: $0 [选项]

选项:
    -h, --help              显示帮助信息
    -H, --host HOST         数据库主机 (默认: 127.0.0.1)
    -P, --port PORT         数据库端口 (默认: 3306)
    -u, --user USER         数据库用户 (默认: root)
    -p, --password PASS     数据库密码
    -d, --database DB       数据库名称 (默认: iam_contracts)
    -f, --force             强制执行，跳过所有确认提示

警告:
    此脚本将完全删除数据库 '${DB_NAME}' 及其所有数据！
    仅在开发环境中使用，切勿在生产环境使用！

示例:
    # 交互式重置
    $0

    # 强制重置（跳过确认）
    $0 --force

    # 指定数据库连接
    $0 -H localhost -u root -p mypassword
EOF
}

# 确认操作
confirm_reset() {
    if [ "${FORCE}" = true ]; then
        return 0
    fi
    
    echo ""
    print_warning "您即将执行以下操作:"
    echo "  1. 删除数据库: ${DB_NAME}"
    echo "  2. 删除所有表和数据"
    echo "  3. 重新创建数据库和表结构"
    echo "  4. 重新加载种子数据"
    echo ""
    print_error "此操作不可恢复！"
    echo ""
    
    read -p "$(echo -e ${RED}[危险操作]${NC} 请输入数据库名称 '${DB_NAME}' 以确认: )" -r
    echo
    
    if [ "${REPLY}" = "${DB_NAME}" ]; then
        read -p "$(echo -e ${RED}[二次确认]${NC} 确定要继续吗? [yes/NO]: )" -r
        echo
        
        if [ "${REPLY}" = "yes" ]; then
            return 0
        fi
    fi
    
    print_info "操作已取消"
    return 1
}

# 删除数据库
drop_database() {
    print_info "删除数据库 '${DB_NAME}'..."
    
    local mysql_cmd="mysql -h${DB_HOST} -P${DB_PORT} -u${DB_USER}"
    
    if [ -n "${DB_PASSWORD}" ]; then
        mysql_cmd="${mysql_cmd} -p${DB_PASSWORD}"
    fi
    
    if ${mysql_cmd} -e "DROP DATABASE IF EXISTS ${DB_NAME};" 2>/dev/null; then
        print_success "数据库已删除"
        return 0
    else
        print_error "删除数据库失败"
        return 1
    fi
}

# 重置数据库
reset_database() {
    print_banner
    
    # 显示配置
    print_info "数据库配置:"
    echo "  主机: ${DB_HOST}"
    echo "  端口: ${DB_PORT}"
    echo "  用户: ${DB_USER}"
    echo "  数据库: ${DB_NAME}"
    echo ""
    
    # 确认操作
    if ! confirm_reset; then
        exit 0
    fi
    
    echo ""
    print_info "开始重置数据库..."
    echo ""
    
    # 删除数据库
    if ! drop_database; then
        exit 1
    fi
    
    # 调用初始化脚本
    print_info "重新初始化数据库..."
    
    local init_cmd="${SCRIPT_DIR}/init-db.sh"
    init_cmd="${init_cmd} -H ${DB_HOST} -P ${DB_PORT} -u ${DB_USER}"
    
    if [ -n "${DB_PASSWORD}" ]; then
        init_cmd="${init_cmd} -p ${DB_PASSWORD}"
    fi
    
    init_cmd="${init_cmd} -d ${DB_NAME} --skip-confirm"
    
    if ${init_cmd}; then
        echo ""
        print_success "数据库重置完成！"
        echo ""
    else
        print_error "数据库重置失败"
        exit 1
    fi
}

# ============================================================================
# 解析命令行参数
# ============================================================================

FORCE=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        -H|--host)
            DB_HOST="$2"
            shift 2
            ;;
        -P|--port)
            DB_PORT="$2"
            shift 2
            ;;
        -u|--user)
            DB_USER="$2"
            shift 2
            ;;
        -p|--password)
            DB_PASSWORD="$2"
            shift 2
            ;;
        -d|--database)
            DB_NAME="$2"
            shift 2
            ;;
        -f|--force)
            FORCE=true
            shift
            ;;
        *)
            print_error "未知选项: $1"
            show_help
            exit 1
            ;;
    esac
done

# ============================================================================
# 执行重置
# ============================================================================

reset_database
