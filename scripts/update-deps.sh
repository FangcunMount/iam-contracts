#!/bin/bash

# 依赖更新脚本
# 用于更新项目依赖到最新版本

set -e

COLOR_RESET="\033[0m"
COLOR_GREEN="\033[32m"
COLOR_YELLOW="\033[33m"
COLOR_BLUE="\033[34m"
COLOR_CYAN="\033[36m"
COLOR_RED="\033[31m"

echo -e "${COLOR_CYAN}═══════════════════════════════════════${COLOR_RESET}"
echo -e "${COLOR_CYAN}    依赖更新工具${COLOR_RESET}"
echo -e "${COLOR_CYAN}═══════════════════════════════════════${COLOR_RESET}"
echo ""

# 显示当前版本
echo -e "${COLOR_BLUE}📦 当前依赖版本:${COLOR_RESET}"
echo -e "${COLOR_YELLOW}component-base:${COLOR_RESET}"
go list -m github.com/FangcunMount/component-base
echo ""

# 更新 component-base
echo -e "${COLOR_CYAN}🔄 更新 component-base 到最新版本...${COLOR_RESET}"
if go get -u github.com/FangcunMount/component-base@latest; then
    echo -e "${COLOR_GREEN}✅ component-base 更新成功${COLOR_RESET}"
else
    echo -e "${COLOR_RED}❌ component-base 更新失败${COLOR_RESET}"
    exit 1
fi

# 更新所有依赖（可选）
if [ "$1" == "--all" ] || [ "$1" == "-a" ]; then
    echo ""
    echo -e "${COLOR_CYAN}🔄 更新所有依赖...${COLOR_RESET}"
    go get -u ./...
    echo -e "${COLOR_GREEN}✅ 所有依赖已更新${COLOR_RESET}"
fi

# 整理依赖
echo ""
echo -e "${COLOR_CYAN}🧹 整理依赖...${COLOR_RESET}"
go mod tidy
echo -e "${COLOR_GREEN}✅ 依赖整理完成${COLOR_RESET}"

# 验证依赖
echo ""
echo -e "${COLOR_CYAN}🔍 验证依赖...${COLOR_RESET}"
go mod verify
echo -e "${COLOR_GREEN}✅ 依赖验证通过${COLOR_RESET}"

# 显示更新后的版本
echo ""
echo -e "${COLOR_BLUE}📦 更新后版本:${COLOR_RESET}"
echo -e "${COLOR_YELLOW}component-base:${COLOR_RESET}"
go list -m github.com/FangcunMount/component-base
echo ""

# 检查是否有不兼容的更改
echo -e "${COLOR_CYAN}🧪 编译检查...${COLOR_RESET}"
if go build ./...; then
    echo -e "${COLOR_GREEN}✅ 编译成功，依赖兼容${COLOR_RESET}"
else
    echo -e "${COLOR_RED}❌ 编译失败，可能存在不兼容的更改${COLOR_RESET}"
    echo -e "${COLOR_YELLOW}⚠️  请检查代码并修复编译错误${COLOR_RESET}"
    exit 1
fi

echo ""
echo -e "${COLOR_GREEN}═══════════════════════════════════════${COLOR_RESET}"
echo -e "${COLOR_GREEN}    ✅ 依赖更新完成！${COLOR_RESET}"
echo -e "${COLOR_GREEN}═══════════════════════════════════════${COLOR_RESET}"
echo ""
echo -e "${COLOR_YELLOW}提示: 运行测试确保一切正常:${COLOR_RESET}"
echo -e "  make test-unit"
echo ""
