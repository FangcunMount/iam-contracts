#!/bin/bash
# ============================================================================
# IAM Contracts 部署脚本
# 用于配合 Jenkins 进行自动化部署
# ============================================================================

set -e

# ============================================================================
# 配置变量
# ============================================================================

APP_NAME="apiserver"
DEPLOY_DIR="/opt/iam"
BIN_DIR="${DEPLOY_DIR}/bin"
CONFIG_DIR="${DEPLOY_DIR}/configs"
LOG_DIR="/var/log/iam-contracts"
PID_FILE="/var/run/${APP_NAME}.pid"
SERVICE_NAME="iam-apiserver"

# 是否使用 systemd
USE_SYSTEMD=false

# ============================================================================
# 颜色输出
# ============================================================================

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# ============================================================================
# 检查函数
# ============================================================================

check_systemd() {
    if command -v systemctl &> /dev/null && systemctl is-system-running &> /dev/null; then
        if [ -f "/etc/systemd/system/${SERVICE_NAME}.service" ]; then
            USE_SYSTEMD=true
            log_info "检测到 systemd 服务，将使用 systemd 管理"
        fi
    fi
}

check_running() {
    if [ "$USE_SYSTEMD" = true ]; then
        systemctl is-active --quiet ${SERVICE_NAME}
    else
        if [ -f "${PID_FILE}" ]; then
            PID=$(cat ${PID_FILE})
            ps -p ${PID} > /dev/null 2>&1
        else
            return 1
        fi
    fi
}

# ============================================================================
# 服务管理函数
# ============================================================================

start_service() {
    log_info "启动服务: ${APP_NAME}"
    
    if check_running; then
        log_warn "服务已在运行"
        return 0
    fi
    
    # 确保目录存在
    mkdir -p ${LOG_DIR}
    mkdir -p $(dirname ${PID_FILE})
    
    if [ "$USE_SYSTEMD" = true ]; then
        sudo systemctl start ${SERVICE_NAME}
    else
        cd ${BIN_DIR}
        nohup ./${APP_NAME} --config=${CONFIG_DIR}/apiserver.yaml > ${LOG_DIR}/${APP_NAME}.log 2>&1 &
        echo $! > ${PID_FILE}
    fi
    
    # 等待服务启动
    sleep 3
    
    if check_running; then
        log_success "服务启动成功"
    else
        log_error "服务启动失败"
        return 1
    fi
}

stop_service() {
    log_info "停止服务: ${APP_NAME}"
    
    if ! check_running; then
        log_warn "服务未运行"
        return 0
    fi
    
    if [ "$USE_SYSTEMD" = true ]; then
        sudo systemctl stop ${SERVICE_NAME}
    else
        if [ -f "${PID_FILE}" ]; then
            PID=$(cat ${PID_FILE})
            if ps -p ${PID} > /dev/null 2>&1; then
                kill ${PID}
                
                # 等待进程结束
                for i in {1..30}; do
                    if ! ps -p ${PID} > /dev/null 2>&1; then
                        break
                    fi
                    sleep 1
                done
                
                # 如果还没停止，强制杀死
                if ps -p ${PID} > /dev/null 2>&1; then
                    log_warn "强制停止服务"
                    kill -9 ${PID}
                fi
            fi
            rm -f ${PID_FILE}
        fi
    fi
    
    log_success "服务已停止"
}

restart_service() {
    log_info "重启服务: ${APP_NAME}"
    stop_service
    sleep 2
    start_service
}

status_service() {
    if [ "$USE_SYSTEMD" = true ]; then
        sudo systemctl status ${SERVICE_NAME}
    else
        if check_running; then
            PID=$(cat ${PID_FILE})
            log_success "服务运行中 (PID: ${PID})"
        else
            log_warn "服务未运行"
            return 1
        fi
    fi
}

# ============================================================================
# 健康检查
# ============================================================================

health_check() {
    local max_retry=10
    local retry_count=0
    local health_url="http://localhost:8080/healthz"
    
    log_info "执行健康检查..."
    
    while [ ${retry_count} -lt ${max_retry} ]; do
        if curl -sf ${health_url} > /dev/null 2>&1; then
            log_success "健康检查通过"
            return 0
        fi
        
        log_info "等待服务就绪... ($((retry_count + 1))/${max_retry})"
        sleep 3
        retry_count=$((retry_count + 1))
    done
    
    log_error "健康检查失败"
    return 1
}

# ============================================================================
# 部署函数
# ============================================================================

deploy() {
    log_info "开始部署 ${APP_NAME}"
    
    # 检查 systemd
    check_systemd
    
    # 停止旧服务
    if check_running; then
        stop_service
    fi
    
    # 备份旧版本
    if [ -f "${BIN_DIR}/${APP_NAME}" ]; then
        log_info "备份旧版本"
        cp ${BIN_DIR}/${APP_NAME} ${BIN_DIR}/${APP_NAME}.backup.$(date +%Y%m%d%H%M%S)
        
        # 只保留最近 5 个备份
        ls -t ${BIN_DIR}/${APP_NAME}.backup.* 2>/dev/null | tail -n +6 | xargs rm -f 2>/dev/null || true
    fi
    
    # 启动新服务
    start_service
    
    # 健康检查
    if ! health_check; then
        log_error "部署失败，尝试回滚"
        rollback
        return 1
    fi
    
    log_success "部署成功"
}

# ============================================================================
# 回滚函数
# ============================================================================

rollback() {
    log_warn "执行回滚操作"
    
    # 停止当前服务
    stop_service
    
    # 查找最新备份
    local latest_backup=$(ls -t ${BIN_DIR}/${APP_NAME}.backup.* 2>/dev/null | head -1)
    
    if [ -z "${latest_backup}" ]; then
        log_error "未找到备份文件，无法回滚"
        return 1
    fi
    
    log_info "恢复备份: ${latest_backup}"
    cp ${latest_backup} ${BIN_DIR}/${APP_NAME}
    chmod +x ${BIN_DIR}/${APP_NAME}
    
    # 启动服务
    start_service
    
    # 健康检查
    if health_check; then
        log_success "回滚成功"
    else
        log_error "回滚失败，请手动介入"
        return 1
    fi
}

# ============================================================================
# 主函数
# ============================================================================

main() {
    case "${1:-}" in
        start)
            check_systemd
            start_service
            ;;
        stop)
            check_systemd
            stop_service
            ;;
        restart)
            check_systemd
            restart_service
            ;;
        status)
            check_systemd
            status_service
            ;;
        deploy)
            deploy
            ;;
        rollback)
            rollback
            ;;
        health)
            health_check
            ;;
        *)
            echo "Usage: $0 {start|stop|restart|status|deploy|rollback|health}"
            echo ""
            echo "Commands:"
            echo "  start    - 启动服务"
            echo "  stop     - 停止服务"
            echo "  restart  - 重启服务"
            echo "  status   - 查看服务状态"
            echo "  deploy   - 部署新版本"
            echo "  rollback - 回滚到上一版本"
            echo "  health   - 健康检查"
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@"
