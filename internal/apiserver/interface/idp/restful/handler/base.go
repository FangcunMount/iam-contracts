// Package handler IDP 模块 REST API 处理器基础
package handler

import (
	pkgHandler "github.com/FangcunMount/iam-contracts/pkg/handler"
)

// BaseHandler 继承公共的 BaseHandler
type BaseHandler struct {
	*pkgHandler.BaseHandler
}

// NewBaseHandler 创建基础 Handler
func NewBaseHandler() *BaseHandler {
	return &BaseHandler{
		BaseHandler: pkgHandler.NewBaseHandler(),
	}
}
