// Package code defines error codes for web-framework platform.
//
// 错误代码定义包，用于定义Web框架平台的错误代码。
//
// 错误代码格式：
//   - 成功：0
//   - 客户端错误：1000-1999
//   - 服务器错误：2000-2999
//   - 数据库错误：3000-3999
//   - 认证错误：4000-4999
//   - 业务错误：5000-5999
//
// 使用示例：
//
//	return errors.WithCode(code.ErrInvalidParameter, "invalid parameter")
package code
