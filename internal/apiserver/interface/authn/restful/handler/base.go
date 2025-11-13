package handler

import (
	pkgHandler "github.com/FangcunMount/iam-contracts/pkg/handler"
)

// BaseHandler 继承公共的 BaseHandler
type BaseHandler struct {
	*pkgHandler.BaseHandler
}

// NewBaseHandler 构造基础处理器
func NewBaseHandler() *BaseHandler {
	return &BaseHandler{
		BaseHandler: pkgHandler.NewBaseHandler(),
	}
}
