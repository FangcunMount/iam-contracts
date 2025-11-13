package core

import (
	"net/http"

	"github.com/FangcunMount/component-base/pkg/errors"
	"github.com/gin-gonic/gin"
)

// ErrResponse 定义了当发生错误时返回的消息
// 如果 Reference 不存在，则省略
// swagger:model
type ErrResponse struct {
	// Code 定义了业务错误代码
	Code int `json:"code"`

	// Message 包含此消息的详细信息
	// 此消息适合暴露给外部
	Message string `json:"message"`

	// Reference 返回参考文档，可能有助于解决此错误
	Reference string `json:"reference,omitempty"`
}

// SuccessResponse 定义了当请求成功时返回的消息
// swagger:model
type SuccessResponse struct {
	// Code 定义了业务成功代码
	Code int `json:"code"`

	// Message 包含此消息的详细信息
	Message string `json:"message"`

	// Data 包含成功响应的数据
	Data interface{} `json:"data,omitempty"`
}

// WriteResponse 将错误或响应数据写入HTTP响应。
// 所有响应(包括业务错误)统一返回 HTTP 200,通过响应体中的 code 字段区分成功/失败。
// 如果err不为空，则将解析后的错误信息写入响应体的 ErrResponse 结构；
// 否则，将data作为成功响应写入。
func WriteResponse(c *gin.Context, err error, data interface{}) {
	if err != nil {
		// 使用 errors.ParseCoder 解析自定义错误
		coder := errors.ParseCoder(err)
		// 统一返回 HTTP 200，业务错误通过响应体中的 code 字段表示
		c.JSON(http.StatusOK, ErrResponse{
			Code:      coder.Code(),
			Message:   coder.String(),
			Reference: coder.Reference(),
		})

		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Code:    0, // 0 表示成功
		Message: "success",
		Data:    data,
	})
}
