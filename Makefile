# ============================================================================
# Makefile for IAM Contracts
# ============================================================================
# 项目：iam-contracts - IAM 身份与访问管理系统
# 架构：六边形架构 + DDD + CQRS
# ============================================================================

.DEFAULT_GOAL := help

# ============================================================================
# 变量定义
# ============================================================================

# 项目信息
PROJECT_NAME := iam-contracts
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.0.0-dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")

# Go 相关
GO := go
GO_BUILD := $(GO) build
GO_TEST := $(GO) test
GO_LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

# 目录结构
BIN_DIR := bin
TMP_DIR := tmp
PID_DIR := $(TMP_DIR)/pids
LOG_DIR := logs
COVERAGE_DIR := coverage

# 服务配置
APISERVER_BIN := $(BIN_DIR)/apiserver
APISERVER_CONFIG := configs/apiserver-simple.yaml
APISERVER_PORT := 8080

# 颜色输出
COLOR_RESET := \033[0m
COLOR_BOLD := \033[1m
COLOR_GREEN := \033[32m
COLOR_YELLOW := \033[33m
COLOR_BLUE := \033[34m
COLOR_CYAN := \033[36m
COLOR_RED := \033[31m

# ============================================================================
# .PHONY 声明
# ============================================================================

.PHONY: help version debug
.PHONY: build build-apiserver clean
.PHONY: run run-apiserver stop stop-apiserver restart restart-apiserver
.PHONY: status status-apiserver logs logs-apiserver health
.PHONY: dev dev-apiserver dev-stop dev-status
.PHONY: test test-unit test-coverage test-race test-bench
.PHONY: lint fmt fmt-check
.PHONY: deps deps-download deps-tidy deps-verify
.PHONY: proto proto-gen
.PHONY: install install-tools create-dirs
.PHONY: up down re st log

# ============================================================================
# 帮助信息
# ============================================================================

help: ## 显示帮助信息
	@echo "$(COLOR_BOLD)$(COLOR_CYAN)======================================"
	@echo "  IAM Contracts - 构建和管理工具"
	@echo "======================================$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_BOLD)项目信息:$(COLOR_RESET)"
	@echo "  版本:     $(COLOR_GREEN)$(VERSION)$(COLOR_RESET)"
	@echo "  分支:     $(COLOR_GREEN)$(GIT_BRANCH)$(COLOR_RESET)"
	@echo "  提交:     $(COLOR_GREEN)$(GIT_COMMIT)$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_BOLD)📦 构建命令:$(COLOR_RESET)"
	@grep -E '^build.*:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(COLOR_CYAN)%-20s$(COLOR_RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(COLOR_BOLD)🚀 服务管理:$(COLOR_RESET)"
	@grep -E '^(run|start|stop|restart|status|logs|health).*:.*?## .*$$' $(MAKEFILE_LIST) | grep -v "dev" | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(COLOR_CYAN)%-20s$(COLOR_RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(COLOR_BOLD)🛠️  开发工具:$(COLOR_RESET)"
	@grep -E '^(dev|test|lint|fmt).*:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(COLOR_CYAN)%-20s$(COLOR_RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(COLOR_BOLD)📚 其他命令:$(COLOR_RESET)"
	@grep -E '^(deps|proto|install|clean|version|debug|up|down|st).*:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(COLOR_CYAN)%-20s$(COLOR_RESET) %s\n", $$1, $$2}'
	@echo ""

version: ## 显示版本信息
	@echo "$(COLOR_BOLD)版本信息:$(COLOR_RESET)"
	@echo "  版本:     $(COLOR_GREEN)$(VERSION)$(COLOR_RESET)"
	@echo "  构建时间: $(BUILD_TIME)"
	@echo "  Git 提交: $(GIT_COMMIT)"
	@echo "  Git 分支: $(GIT_BRANCH)"
	@echo "  Go 版本:  $(shell $(GO) version)"

# ============================================================================
# 构建命令
# ============================================================================

build: build-apiserver ## 构建所有服务

build-apiserver: ## 构建 API 服务器
	@echo "$(COLOR_BOLD)$(COLOR_BLUE)🔨 构建 API 服务器...$(COLOR_RESET)"
	@$(MAKE) create-dirs
	@$(GO_BUILD) $(GO_LDFLAGS) -o $(APISERVER_BIN) ./cmd/apiserver/
	@echo "$(COLOR_GREEN)✅ API 服务器构建完成: $(APISERVER_BIN)$(COLOR_RESET)"

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
# ============================================================================
# 测试（扩展）
# ============================================================================

test-unit: ## 运行单元测试
	@echo "$(COLOR_CYAN)🧪 运行单元测试...$(COLOR_RESET)"
	@$(GO_TEST) -v -short ./...

test-coverage: create-dirs ## 生成测试覆盖率报告
	@echo "$(COLOR_CYAN)🧪 生成测试覆盖率报告...$(COLOR_RESET)"
	@mkdir -p $(COVERAGE_DIR)
	@$(GO_TEST) -v -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	@$(GO) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "$(COLOR_GREEN)✅ 覆盖率报告: $(COVERAGE_DIR)/coverage.html$(COLOR_RESET)"
	@$(GO) tool cover -func=$(COVERAGE_DIR)/coverage.out | tail -n 1

test-race: ## 运行竞态检测
	@echo "$(COLOR_CYAN)🧪 运行竞态检测...$(COLOR_RESET)"
	@$(GO_TEST) -v -race ./...

test-bench: ## 运行基准测试
	@echo "$(COLOR_CYAN)🧪 运行基准测试...$(COLOR_RESET)"
	@$(GO_TEST) -v -bench=. -benchmem ./...

# ============================================================================
# 代码质量
# ============================================================================

lint: ## 运行代码检查
	@echo "$(COLOR_CYAN)🔍 运行代码检查...$(COLOR_RESET)"
	@if command -v golangci-lint > /dev/null 2>&1; then \
		golangci-lint run --timeout=5m ./...; \
	else \
		echo "$(COLOR_YELLOW)⚠️  golangci-lint 未安装，使用 go vet$(COLOR_RESET)"; \
		$(GO) vet ./...; \
	fi

fmt: ## 格式化代码
	@echo "$(COLOR_CYAN)✨ 格式化代码...$(COLOR_RESET)"
	@$(GO) fmt ./...
	@gofmt -s -w .
	@echo "$(COLOR_GREEN)✅ 代码格式化完成$(COLOR_RESET)"

fmt-check: ## 检查代码格式
	@echo "$(COLOR_CYAN)🔍 检查代码格式...$(COLOR_RESET)"
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "$(COLOR_RED)❌ 以下文件需要格式化:$(COLOR_RESET)"; \
		gofmt -l .; \
		exit 1; \
	else \
		echo "$(COLOR_GREEN)✅ 代码格式正确$(COLOR_RESET)"; \
	fi

# ============================================================================
# 依赖管理
# ============================================================================

deps: deps-download ## 下载依赖

deps-download: ## 下载所有依赖
	@echo "$(COLOR_CYAN)📦 下载依赖...$(COLOR_RESET)"
	@$(GO) mod download
	@echo "$(COLOR_GREEN)✅ 依赖下载完成$(COLOR_RESET)"

deps-tidy: ## 整理依赖
	@echo "$(COLOR_CYAN)🧹 整理依赖...$(COLOR_RESET)"
	@$(GO) mod tidy
	@echo "$(COLOR_GREEN)✅ 依赖整理完成$(COLOR_RESET)"

deps-verify: ## 验证依赖
	@echo "$(COLOR_CYAN)🔍 验证依赖...$(COLOR_RESET)"
	@$(GO) mod verify
	@echo "$(COLOR_GREEN)✅ 依赖验证通过$(COLOR_RESET)"

# ============================================================================
# Protocol Buffers
# ============================================================================

proto-gen: ## 生成 protobuf 代码
	@echo "$(COLOR_CYAN)🔨 生成 protobuf 代码...$(COLOR_RESET)"
	@if [ -f scripts/proto/generate.sh ]; then \
		bash scripts/proto/generate.sh; \
		echo "$(COLOR_GREEN)✅ Protobuf 代码生成完成$(COLOR_RESET)"; \
	else \
		echo "$(COLOR_YELLOW)⚠️  脚本不存在: scripts/proto/generate.sh$(COLOR_RESET)"; \
	fi

# ============================================================================
# 工具安装
# ============================================================================

install-tools: ## 安装开发工具
	@echo "$(COLOR_CYAN)📦 安装开发工具...$(COLOR_RESET)"
	@echo "安装 Air (热更新)..."
	@go install github.com/air-verse/air@latest
	@echo "安装 mockgen..."
	@go install go.uber.org/mock/mockgen@latest
	@echo "$(COLOR_GREEN)✅ 工具安装完成$(COLOR_RESET)"

# ============================================================================
# 调试和诊断
# ============================================================================

debug: ## 显示调试信息
	@echo "$(COLOR_BOLD)$(COLOR_CYAN)🔍 调试信息:$(COLOR_RESET)"
	@echo "$(COLOR_BOLD)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(COLOR_RESET)"
	@echo "项目名称:     $(PROJECT_NAME)"
	@echo "版本:         $(VERSION)"
	@echo "Git 提交:     $(GIT_COMMIT)"
	@echo "Git 分支:     $(GIT_BRANCH)"
	@echo "构建时间:     $(BUILD_TIME)"
	@echo "Go 版本:      $(shell $(GO) version)"
	@echo "GOPATH:       $(shell go env GOPATH)"
	@echo "GOOS:         $(shell go env GOOS)"
	@echo "GOARCH:       $(shell go env GOARCH)"
	@echo "$(COLOR_BOLD)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(COLOR_RESET)"

ps: ## 显示相关进程
	@echo "$(COLOR_CYAN)🔍 相关进程:$(COLOR_RESET)"
	@ps aux | grep -E "(apiserver|air)" | grep -v grep || echo "$(COLOR_YELLOW)未找到相关进程$(COLOR_RESET)"

ports: ## 检查端口占用
	@echo "$(COLOR_CYAN)🔍 端口占用:$(COLOR_RESET)"
	@lsof -i :$(APISERVER_PORT) 2>/dev/null || echo "$(COLOR_GREEN)端口 $(APISERVER_PORT) 未被占用$(COLOR_RESET)"

# ============================================================================
# CI/CD
# ============================================================================

ci: deps-verify fmt-check lint test ## CI 流程
	@echo "$(COLOR_GREEN)✅ CI 检查通过$(COLOR_RESET)"

release: clean build ## 发布版本
	@echo "$(COLOR_GREEN)✅ 版本 $(VERSION) 发布准备完成$(COLOR_RESET)"

# ============================================================================
# 快捷命令
# ============================================================================

up: run ## 启动服务（别名）
down: stop ## 停止服务（别名）
re: restart ## 重启服务（别名）
st: status ## 查看状态（别名）
log: logs ## 查看日志（别名）
