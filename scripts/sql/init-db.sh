#!/bin/bash

# ============================================================================
# IAM Contracts 数据库初始化脚本
# ============================================================================
# 功能: 创建数据库、执行表结构初始化、加载种子数据
# 使用: ./init-db.sh [options]
# ============================================================================

set -e  # 遇到错误立即退出

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 默认配置
DB_HOST="${DB_HOST:-127.0.0.1}"
DB_PORT="${DB_PORT:-3306}"
DB_USER="${DB_USER:-root}"
DB_PASSWORD="${DB_PASSWORD:-}"
DB_NAME="${DB_NAME:-iam_contracts}"

# 脚本目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SQL_DIR="${SCRIPT_DIR}"

# ============================================================================
# 工具函数
# ============================================================================

# 打印消息
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

# 打印横幅
print_banner() {
    echo ""
    echo -e "${BLUE}============================================${NC}"
    echo -e "${BLUE}  IAM Contracts - 数据库初始化工具${NC}"
    echo -e "${BLUE}============================================${NC}"
    echo ""
}

# 显示帮助信息
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
    --schema-only           仅创建表结构，不加载种子数据
    --seed-only             仅加载种子数据（需要表已存在）
    --skip-confirm          跳过确认提示

环境变量:
    DB_HOST                 数据库主机
    DB_PORT                 数据库端口
    DB_USER                 数据库用户
    DB_PASSWORD             数据库密码
    DB_NAME                 数据库名称

示例:
    # 使用默认配置初始化
    $0

    # 指定数据库连接信息
    $0 -H localhost -P 3306 -u root -p mypassword

    # 使用环境变量
    export DB_HOST=localhost
    export DB_USER=root
    export DB_PASSWORD=mypassword
    $0

    # 仅创建表结构
    $0 --schema-only

    # 仅加载种子数据
    $0 --seed-only
EOF
}

# 检查 MySQL 客户端
check_mysql_client() {
    if ! command -v mysql &> /dev/null; then
        print_error "未找到 mysql 客户端，请先安装 MySQL 客户端工具"
        print_info "macOS: brew install mysql-client"
        print_info "Ubuntu/Debian: sudo apt-get install mysql-client"
        print_info "CentOS/RHEL: sudo yum install mysql"
        exit 1
    fi
}

# 测试数据库连接
test_connection() {
    print_info "测试数据库连接..."
    
    local mysql_cmd="mysql -h${DB_HOST} -P${DB_PORT} -u${DB_USER}"
    
    if [ -n "${DB_PASSWORD}" ]; then
        mysql_cmd="${mysql_cmd} -p${DB_PASSWORD}"
    fi
    
    if ${mysql_cmd} -e "SELECT 1;" &> /dev/null; then
        print_success "数据库连接成功"
        return 0
    else
        print_error "数据库连接失败"
        print_info "请检查数据库连接信息:"
        print_info "  主机: ${DB_HOST}"
        print_info "  端口: ${DB_PORT}"
        print_info "  用户: ${DB_USER}"
        return 1
    fi
}

# 检查数据库是否存在
database_exists() {
    local mysql_cmd="mysql -h${DB_HOST} -P${DB_PORT} -u${DB_USER}"
    
    if [ -n "${DB_PASSWORD}" ]; then
        mysql_cmd="${mysql_cmd} -p${DB_PASSWORD}"
    fi
    
    local result=$(${mysql_cmd} -e "SELECT SCHEMA_NAME FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME='${DB_NAME}';" 2>/dev/null | grep -c "${DB_NAME}" || true)
    
    if [ "${result}" -gt 0 ]; then
        return 0
    else
        return 1
    fi
}

# 执行 SQL 文件
execute_sql_file() {
    local sql_file=$1
    local description=$2
    
    if [ ! -f "${sql_file}" ]; then
        print_error "SQL 文件不存在: ${sql_file}"
        return 1
    fi
    
    print_info "${description}..."
    
    local mysql_cmd="mysql -h${DB_HOST} -P${DB_PORT} -u${DB_USER}"
    
    if [ -n "${DB_PASSWORD}" ]; then
        mysql_cmd="${mysql_cmd} -p${DB_PASSWORD}"
    fi
    
    if ${mysql_cmd} < "${sql_file}"; then
        print_success "${description}完成"
        return 0
    else
        print_error "${description}失败"
        return 1
    fi
}

# 确认操作
confirm_action() {
    local message=$1
    
    if [ "${SKIP_CONFIRM}" = true ]; then
        return 0
    fi
    
    read -p "$(echo -e ${YELLOW}[确认]${NC} ${message} [y/N]: )" -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        return 0
    else
        print_info "操作已取消"
        return 1
    fi
}

# ============================================================================
# 主要功能
# ============================================================================

# 初始化数据库
init_database() {
    print_banner
    
    # 显示配置信息
    print_info "数据库配置:"
    echo "  主机: ${DB_HOST}"
    echo "  端口: ${DB_PORT}"
    echo "  用户: ${DB_USER}"
    echo "  数据库: ${DB_NAME}"
    echo ""
    
    # 检查 MySQL 客户端
    check_mysql_client
    
    # 测试连接
    if ! test_connection; then
        exit 1
    fi
    
    # 检查数据库是否存在
    if database_exists; then
        print_warning "数据库 '${DB_NAME}' 已存在"
        
        if ! confirm_action "是否要重新初始化? 这将删除所有现有数据!"; then
            exit 0
        fi
    fi
    
    # 执行初始化
    if [ "${SEED_ONLY}" = true ]; then
        # 仅加载种子数据
        if ! database_exists; then
            print_error "数据库不存在，无法仅加载种子数据"
            print_info "请先运行完整初始化或使用 --schema-only 创建表结构"
            exit 1
        fi
        
        execute_sql_file "${SQL_DIR}/seed.sql" "加载种子数据"
    elif [ "${SCHEMA_ONLY}" = true ]; then
        # 仅创建表结构
        execute_sql_file "${SQL_DIR}/init.sql" "创建数据库和表结构"
    else
        # 完整初始化
        execute_sql_file "${SQL_DIR}/init.sql" "创建数据库和表结构"
        execute_sql_file "${SQL_DIR}/seed.sql" "加载种子数据"
    fi
    
    # 显示完成信息
    echo ""
    print_success "数据库初始化完成！"
    echo ""
    print_info "默认账户信息:"
    echo "  系统管理员:"
    echo "    用户名: admin"
    echo "    密码: admin123"
    echo ""
    echo "  演示租户管理员:"
    echo "    用户名: zhangsan"
    echo "    密码: admin123"
    echo ""
    echo "  演示租户监护人:"
    echo "    用户名: lisi"
    echo "    密码: admin123"
    echo ""
    print_warning "注意: 请在生产环境中修改默认密码！"
    echo ""
}

# ============================================================================
# 解析命令行参数
# ============================================================================

SCHEMA_ONLY=false
SEED_ONLY=false
SKIP_CONFIRM=false

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
        --schema-only)
            SCHEMA_ONLY=true
            shift
            ;;
        --seed-only)
            SEED_ONLY=true
            shift
            ;;
        --skip-confirm)
            SKIP_CONFIRM=true
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
# 执行初始化
# ============================================================================

init_database
