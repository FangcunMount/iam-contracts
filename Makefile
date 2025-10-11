.PHONY: help dev build clean test
.PHONY: build-all run-all stop-all status-all logs-all
.PHONY: build-apiserver run-apiserver stop-apiserver

# 服务配置
APISERVER_BIN = iam-contracts
APISERVER_CONFIG = configs/apiserver-simple.yaml
APISERVER_PORT = 8080

# PID 文件目录
PID_DIR = tmp/pids
LOG_DIR = logs

# 默认目标
help: ## 显示帮助信息
	@echo "iam contracts - 服务管理工具"
	@echo "============================="
	@echo ""
	@echo "🏗️  构建命令:"
	@grep -E '^build.*:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "🚀 服务管理:"
	@grep -E '^(run|start|stop|restart|status|logs).*:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "🧪 开发工具:"
	@grep -E '^(dev|test|clean|deps).*:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# =============================================================================
# 构建命令
# =============================================================================

build: build-apiserver ## 构建服务

build-apiserver: ## 构建 API 服务器
	@echo "🔨 构建 iam-contracts..."
	@go build -o $(APISERVER_BIN) ./cmd/apiserver/

# =============================================================================
# 服务运行管理
# =============================================================================

run: run-apiserver ## 启动服务

run-apiserver: ## 启动 API 服务器
	@echo "🚀 启动 iam-contracts..."
	@$(MAKE) create-dirs
	@if [ -f $(PID_DIR)/apiserver.pid ]; then \
			echo "⚠️  iam-contracts 可能已在运行 (PID: $$(cat $(PID_DIR)/apiserver.pid))"; \
		if ! kill -0 $$(cat $(PID_DIR)/apiserver.pid) 2>/dev/null; then \
			echo "🧹 清理无效的 PID 文件"; \
			rm -f $(PID_DIR)/apiserver.pid; \
		else \
			echo "❌ iam-contracts 已在运行，请先停止"; \
			exit 1; \
		fi; \
	fi
	@nohup ./$(APISERVER_BIN) --config=$(APISERVER_CONFIG) > $(LOG_DIR)/apiserver.log 2>&1 & echo $$! > $(PID_DIR)/apiserver.pid
	@echo "✅ iam-contracts 已启动 (PID: $$(cat $(PID_DIR)/apiserver.pid))"

# =============================================================================
# 服务停止管理
# =============================================================================

stop-apiserver: ## 停止 API 服务器
	@echo "⏹️  停止 iam-contracts..."
	@if [ -f $(PID_DIR)/apiserver.pid ]; then \
		PID=$$(cat $(PID_DIR)/apiserver.pid); \
		if kill -0 $$PID 2>/dev/null; then \
			kill $$PID && echo "✅ iam-contracts 已停止 (PID: $$PID)"; \
			rm -f $(PID_DIR)/apiserver.pid; \
		else \
			echo "⚠️  iam-contracts 进程不存在，清理 PID 文件"; \
			rm -f $(PID_DIR)/apiserver.pid; \
		fi; \
	else \
			echo "ℹ️  iam-contracts 未运行"; \
	fi

# =============================================================================
# 服务重启管理
# =============================================================================

restart-apiserver: ## 重启 API 服务器
	@echo "🔄 重启 iam-contracts..."
	@$(MAKE) stop-apiserver
	@sleep 1
	@$(MAKE) run-apiserver

# =============================================================================
# 服务状态和日志
# =============================================================================

status-apiserver: ## 查看 API 服务器状态
	@if [ -f $(PID_DIR)/apiserver.pid ]; then \
		PID=$$(cat $(PID_DIR)/apiserver.pid); \
		if kill -0 $$PID 2>/dev/null; then \
			echo "✅ iam-contracts      - 运行中 (PID: $$PID, Port: $(APISERVER_PORT))"; \
		else \
			echo "❌ iam-contracts      - 已停止 (PID 文件存在但进程不存在)"; \
		fi; \
	else \
			echo "⚪ iam-contracts      - 未运行"; \
	fi

logs-apiserver: ## 查看 API 服务器日志
	@echo "📋 查看 iam-contracts 日志..."
	@tail -f $(LOG_DIR)/apiserver.log

# =============================================================================
# 健康检查
# =============================================================================

health-check: ## 检查所有服务健康状态
	@echo "🔍 健康检查:"
	@echo "============"
	@echo -n "iam-contracts:        "; curl -s http://localhost:$(APISERVER_PORT)/healthz || echo "❌ 无响应"

# =============================================================================
# 测试工具
# =============================================================================

test-message-queue: ## 测试消息队列系统
	@echo "📨 测试消息队列系统..."
	@if [ ! -x "./test-message-queue.sh" ]; then \
		echo "❌ 测试脚本不存在或不可执行"; \
		exit 1; \
	fi
	@./test-message-queue.sh

test-submit: ## 测试答卷提交
	@echo "📝 测试答卷提交..."
	@if [ ! -x "./test-answersheet-submit.sh" ]; then \
		echo "❌ 测试脚本不存在或不可执行"; \
		exit 1; \
	fi
	@./test-answersheet-submit.sh

# =============================================================================
# 开发工具
# =============================================================================

dev: ## 启动开发环境（热更新）
	@echo "🚀 启动开发环境..."
	@mkdir -p tmp
	@echo "启动 iam-contracts..."
	@air -c .air-apiserver.toml & echo $$! > tmp/pids/air-apiserver.pid
	@sleep 2
	@echo "✅ 所有服务已启动（热更新模式）"
	@echo "提示：使用 Ctrl+C 停止所有服务"
	@echo "      或使用 make dev-stop 停止服务"

dev-apiserver: ## 独立启动 API 服务器（热更新）
	@echo "🚀 启动 apiserver 开发环境..."
	@mkdir -p tmp
	@air -c .air-apiserver.toml

dev-stop: ## 停止开发环境
	@echo "⏹️  停止开发环境..."
	@if [ -f tmp/pids/air-apiserver.pid ]; then \
		kill $$(cat tmp/pids/air-apiserver.pid) 2>/dev/null || true; \
		rm -f tmp/pids/air-apiserver.pid; \
	fi
	@echo "✅ 开发环境已停止"

dev-status: ## 查看开发环境状态
	@echo "📊 开发环境状态:"
	@echo "=============="
	@if [ -f tmp/pids/air-apiserver.pid ] && kill -0 $$(cat tmp/pids/air-apiserver.pid) 2>/dev/null; then \
			echo "✅ iam-contracts      - 运行中 (PID: $$(cat tmp/pids/air-apiserver.pid))"; \
	else \
			echo "⚪ iam-contracts      - 未运行"; \
	fi

dev-logs: ## 查看开发环境日志
	@echo "📋 开发环境日志:"
	@echo "=============="
	@tail -f tmp/build-errors-*.log

test: ## 运行测试
	@echo "🧪 运行测试..."
	@go test ./...

clean: ## 清理构建文件和进程
	@echo "🧹 清理构建文件和进程..."
	@$(MAKE) stop-apiserver
	@rm -rf tmp bin $(LOG_DIR)/*.log
	@rm -f $(APISERVER_BIN)
	@go clean
	@echo "✅ 清理完成"

create-dirs: ## 创建必要的目录
	@mkdir -p $(PID_DIR) $(LOG_DIR) 