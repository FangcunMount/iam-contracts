// Package errors 提供统一的错误处理
package errors

import (
	"errors"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ========== 标准错误类型 ==========

// 预定义的标准错误类型。
//
// 这些错误可以使用 errors.Is() 进行判断：
//
//	if errors.Is(err, sdk.ErrNotFound) {
//	    // 处理资源不存在错误
//	}
var (
	// ErrNotFound 表示请求的资源不存在。
	ErrNotFound = errors.New("not found")

	// ErrUnauthorized 表示用户未认证或 Token 无效。
	ErrUnauthorized = errors.New("unauthorized")

	// ErrPermissionDenied 表示用户没有执行操作的权限。
	ErrPermissionDenied = errors.New("permission denied")

	// ErrInvalidArgument 表示请求参数无效或缺失。
	ErrInvalidArgument = errors.New("invalid argument")

	// ErrAlreadyExists 表示要创建的资源已经存在。
	ErrAlreadyExists = errors.New("already exists")

	// ErrServiceUnavailable 表示服务暂时不可用，可能需要重试。
	ErrServiceUnavailable = errors.New("service unavailable")

	// ErrInternal 表示服务器内部错误。
	ErrInternal = errors.New("internal error")

	// ErrTokenExpired 表示 Token 已过期，需要刷新。
	ErrTokenExpired = errors.New("token expired")

	// ErrTokenInvalid 表示 Token 格式无效或签名验证失败。
	ErrTokenInvalid = errors.New("token invalid")

	// ErrTokenRevoked 表示 Token 已被撤销，无法使用。
	ErrTokenRevoked = errors.New("token revoked")

	// ErrRateLimited 表示请求被限流，需要稍后重试。
	ErrRateLimited = errors.New("rate limited")

	// ErrTimeout 表示请求超时。
	ErrTimeout = errors.New("timeout")
)

// ========== IAM 错误类型 ==========

// IAMError IAM SDK 统一错误类型。
//
// 封装 gRPC 错误为结构化的错误对象，便于错误处理和日志记录。
//
// 实现了 error、errors.Unwrap 和 errors.Is 接口，支持错误链判断。
//
// 示例：
//
//	var iamErr *sdk.IAMError
//	if errors.As(err, &iamErr) {
//	    fmt.Printf("错误码: %s, 消息: %s\n", iamErr.Code, iamErr.Message)
//	    if iamErr.GRPCCode == codes.NotFound {
//	        // 处理资源不存在
//	    }
//	}
type IAMError struct {
	// Code 错误码
	Code string

	// Message 错误信息
	Message string

	// GRPCCode gRPC 状态码
	GRPCCode codes.Code

	// Cause 原始错误
	Cause error
}

// Error 实现 error 接口
func (e *IAMError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("iam: %s: %s (%v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("iam: %s: %s", e.Code, e.Message)
}

// Unwrap 实现 errors.Unwrap
func (e *IAMError) Unwrap() error {
	return e.Cause
}

// Is 实现 errors.Is
func (e *IAMError) Is(target error) bool {
	switch e.GRPCCode {
	case codes.NotFound:
		return target == ErrNotFound
	case codes.Unauthenticated:
		return target == ErrUnauthorized
	case codes.PermissionDenied:
		return target == ErrPermissionDenied
	case codes.InvalidArgument:
		return target == ErrInvalidArgument
	case codes.AlreadyExists:
		return target == ErrAlreadyExists
	case codes.Unavailable:
		return target == ErrServiceUnavailable
	case codes.Internal:
		return target == ErrInternal
	case codes.ResourceExhausted:
		return target == ErrRateLimited
	case codes.DeadlineExceeded:
		return target == ErrTimeout
	}
	return false
}

// ========== 错误包装函数 ==========

// Wrap 将 gRPC 错误封装为 IAMError。
//
// 自动提取 gRPC 状态码和错误消息，转换为结构化错误。
// SDK 内部自动调用，用户通常不需要手动调用。
//
// 参数：
//   - err: 原始错误（通常是 gRPC 错误）
//
// 返回：
//   - error: 封装后的 IAMError，如果 err 为 nil 则返回 nil
func Wrap(err error) error {
	if err == nil {
		return nil
	}

	st, ok := status.FromError(err)
	if !ok {
		return &IAMError{
			Code:     "UNKNOWN",
			Message:  err.Error(),
			GRPCCode: codes.Unknown,
			Cause:    err,
		}
	}

	return &IAMError{
		Code:     st.Code().String(),
		Message:  st.Message(),
		GRPCCode: st.Code(),
		Cause:    err,
	}
}

// WrapWithCode 包装错误并指定错误码
func WrapWithCode(err error, code string, message string) error {
	grpcCode := codes.Unknown
	if st, ok := status.FromError(err); ok {
		grpcCode = st.Code()
	}

	return &IAMError{
		Code:     code,
		Message:  message,
		GRPCCode: grpcCode,
		Cause:    err,
	}
}

// ========== 错误检查函数 ==========

// IsNotFound 判断错误是否为“资源不存在”错误。
//
// 参数：
//   - err: 要检查的错误
//
// 返回：
//   - bool: 如果是 NotFound 错误则返回 true
//
// 示例：
//
//	_, err := client.Identity().GetUser(ctx, "user-123")
//	if sdk.IsNotFound(err) {
//	    fmt.Println("用户不存在")
//	}
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound) || hasGRPCCode(err, codes.NotFound)
}

// IsUnauthorized 判断错误是否为“未认证”错误。
//
// 通常表示 Token 缺失、过期或无效。
//
// 参数：
//   - err: 要检查的错误
//
// 返回：
//   - bool: 如果是未认证错误则返回 true
func IsUnauthorized(err error) bool {
	return errors.Is(err, ErrUnauthorized) || hasGRPCCode(err, codes.Unauthenticated)
}

// IsPermissionDenied 判断错误是否为“权限不足”错误。
//
// 表示用户已认证但没有执行该操作的权限。
//
// 参数：
//   - err: 要检查的错误
//
// 返回：
//   - bool: 如果是权限不足错误则返回 true
func IsPermissionDenied(err error) bool {
	return errors.Is(err, ErrPermissionDenied) || hasGRPCCode(err, codes.PermissionDenied)
}

// IsInvalidArgument 检查是否为参数无效错误
func IsInvalidArgument(err error) bool {
	return errors.Is(err, ErrInvalidArgument) || hasGRPCCode(err, codes.InvalidArgument)
}

// IsServiceUnavailable 检查是否为服务不可用错误
func IsServiceUnavailable(err error) bool {
	return errors.Is(err, ErrServiceUnavailable) || hasGRPCCode(err, codes.Unavailable)
}

// IsTimeout 检查是否为超时错误
func IsTimeout(err error) bool {
	return errors.Is(err, ErrTimeout) || hasGRPCCode(err, codes.DeadlineExceeded)
}

// IsRetryable 判断错误是否可以重试。
//
// 可重试的错误类型包括：
//   - Unavailable: 服务不可用
//   - ResourceExhausted: 资源耗尽（限流）
//   - Aborted: 操作被中止
//   - DeadlineExceeded: 超时
//
// 参数：
//   - err: 要检查的错误
//
// 返回：
//   - bool: 如果错误可重试则返回 true
//
// 示例：
//
//	if sdk.IsRetryable(err) {
//	    // 稍后重试
//	    time.Sleep(time.Second)
//	    return retry()
//	}
func IsRetryable(err error) bool {
	if hasGRPCCode(err, codes.Unavailable) ||
		hasGRPCCode(err, codes.ResourceExhausted) ||
		hasGRPCCode(err, codes.Aborted) ||
		hasGRPCCode(err, codes.DeadlineExceeded) {
		return true
	}
	return false
}

func hasGRPCCode(err error, code codes.Code) bool {
	if st, ok := status.FromError(err); ok {
		return st.Code() == code
	}
	var iamErr *IAMError
	if errors.As(err, &iamErr) {
		return iamErr.GRPCCode == code
	}
	return false
}

// ========== 错误创建函数 ==========

// NewNotFound 创建一个资源不存在错误。
//
// 参数：
//   - resource: 资源名称（如 "user"、"child"）
//
// 返回：
//   - error: IAMError 实例
func NewNotFound(resource string) error {
	return &IAMError{
		Code:     "NOT_FOUND",
		Message:  fmt.Sprintf("%s not found", resource),
		GRPCCode: codes.NotFound,
		Cause:    ErrNotFound,
	}
}

// NewInvalidArgument 创建一个参数无效错误。
//
// 参数：
//   - field: 无效的字段名
//   - reason: 原因描述
//
// 返回：
//   - error: IAMError 实例
func NewInvalidArgument(field, reason string) error {
	return &IAMError{
		Code:     "INVALID_ARGUMENT",
		Message:  fmt.Sprintf("invalid %s: %s", field, reason),
		GRPCCode: codes.InvalidArgument,
		Cause:    ErrInvalidArgument,
	}
}

// NewUnauthorized 创建未认证错误
func NewUnauthorized(reason string) error {
	return &IAMError{
		Code:     "UNAUTHENTICATED",
		Message:  reason,
		GRPCCode: codes.Unauthenticated,
		Cause:    ErrUnauthorized,
	}
}

// NewPermissionDenied 创建权限不足错误
func NewPermissionDenied(reason string) error {
	return &IAMError{
		Code:     "PERMISSION_DENIED",
		Message:  reason,
		GRPCCode: codes.PermissionDenied,
		Cause:    ErrPermissionDenied,
	}
}

// ========== 业务错误分类 ==========

// ErrorCategory 错误分类
type ErrorCategory int

const (
	CategoryUnknown    ErrorCategory = iota
	CategoryClient                   // 客户端错误（4xx）
	CategoryServer                   // 服务端错误（5xx）
	CategoryNetwork                  // 网络错误
	CategoryAuth                     // 认证/授权错误
	CategoryValidation               // 参数验证错误
	CategoryRateLimit                // 限流错误
)

func (c ErrorCategory) String() string {
	switch c {
	case CategoryClient:
		return "client"
	case CategoryServer:
		return "server"
	case CategoryNetwork:
		return "network"
	case CategoryAuth:
		return "auth"
	case CategoryValidation:
		return "validation"
	case CategoryRateLimit:
		return "rate_limit"
	default:
		return "unknown"
	}
}

// Categorize 对错误进行分类
func Categorize(err error) ErrorCategory {
	if err == nil {
		return CategoryUnknown
	}

	st, ok := status.FromError(err)
	if !ok {
		var iamErr *IAMError
		if errors.As(err, &iamErr) {
			st = status.New(iamErr.GRPCCode, iamErr.Message)
		} else {
			return CategoryUnknown
		}
	}

	switch st.Code() {
	case codes.InvalidArgument, codes.OutOfRange, codes.FailedPrecondition:
		return CategoryValidation
	case codes.Unauthenticated, codes.PermissionDenied:
		return CategoryAuth
	case codes.NotFound, codes.AlreadyExists:
		return CategoryClient
	case codes.ResourceExhausted:
		return CategoryRateLimit
	case codes.Unavailable, codes.DeadlineExceeded, codes.Canceled:
		return CategoryNetwork
	case codes.Internal, codes.Unknown, codes.DataLoss, codes.Unimplemented:
		return CategoryServer
	default:
		return CategoryUnknown
	}
}

// ========== 更多检查函数 ==========

// IsAlreadyExists 检查是否为资源已存在错误
func IsAlreadyExists(err error) bool {
	return errors.Is(err, ErrAlreadyExists) || hasGRPCCode(err, codes.AlreadyExists)
}

// IsRateLimited 检查是否为限流错误
func IsRateLimited(err error) bool {
	return errors.Is(err, ErrRateLimited) || hasGRPCCode(err, codes.ResourceExhausted)
}

// IsInternal 检查是否为内部错误
func IsInternal(err error) bool {
	return errors.Is(err, ErrInternal) || hasGRPCCode(err, codes.Internal)
}

// IsTokenExpired 检查是否为 Token 过期错误
func IsTokenExpired(err error) bool {
	return errors.Is(err, ErrTokenExpired)
}

// IsTokenInvalid 检查是否为 Token 无效错误
func IsTokenInvalid(err error) bool {
	return errors.Is(err, ErrTokenInvalid)
}

// IsCanceled 检查是否为取消错误
func IsCanceled(err error) bool {
	return hasGRPCCode(err, codes.Canceled)
}

// ========== 错误提取 ==========

// GRPCCode 提取 gRPC 状态码
func GRPCCode(err error) codes.Code {
	if st, ok := status.FromError(err); ok {
		return st.Code()
	}
	var iamErr *IAMError
	if errors.As(err, &iamErr) {
		return iamErr.GRPCCode
	}
	return codes.Unknown
}

// Message 提取错误消息
func Message(err error) string {
	if st, ok := status.FromError(err); ok {
		return st.Message()
	}
	var iamErr *IAMError
	if errors.As(err, &iamErr) {
		return iamErr.Message
	}
	if err != nil {
		return err.Error()
	}
	return ""
}

// =============================================================================
// 错误详情提取
// =============================================================================

// ErrorDetails 错误详情
type ErrorDetails struct {
	// Code 错误码
	Code string

	// GRPCCode gRPC 状态码
	GRPCCode codes.Code

	// Message 错误消息
	Message string

	// Category 错误分类
	Category ErrorCategory

	// IsRetryable 是否可重试
	IsRetryable bool

	// IsClientError 是否客户端错误
	IsClientError bool

	// IsServerError 是否服务端错误
	IsServerError bool

	// SuggestedAction 建议的处理动作
	SuggestedAction ErrorAction

	// RetryAfter 建议的重试时间（如果适用）
	RetryAfter time.Duration
}

// ErrorAction 错误处理动作
type ErrorAction int

const (
	ActionNone       ErrorAction = iota // 无需特殊处理
	ActionRetry                         // 应该重试
	ActionReauth                        // 需要重新认证
	ActionForbidden                     // 禁止访问，无需重试
	ActionNotFound                      // 资源不存在
	ActionBadRequest                    // 修正请求参数
	ActionRateLimit                     // 等待后重试
	ActionFailover                      // 尝试备用服务
	ActionEscalate                      // 升级处理（人工介入）
)

func (a ErrorAction) String() string {
	switch a {
	case ActionRetry:
		return "retry"
	case ActionReauth:
		return "reauth"
	case ActionForbidden:
		return "forbidden"
	case ActionNotFound:
		return "not_found"
	case ActionBadRequest:
		return "bad_request"
	case ActionRateLimit:
		return "rate_limit"
	case ActionFailover:
		return "failover"
	case ActionEscalate:
		return "escalate"
	default:
		return "none"
	}
}

// Analyze 分析错误，返回详细信息
func Analyze(err error) *ErrorDetails {
	if err == nil {
		return nil
	}

	details := &ErrorDetails{}

	// 提取 gRPC 状态
	st, ok := status.FromError(err)
	if ok {
		details.GRPCCode = st.Code()
		details.Message = st.Message()
		details.Code = st.Code().String()
	} else {
		var iamErr *IAMError
		if errors.As(err, &iamErr) {
			details.GRPCCode = iamErr.GRPCCode
			details.Message = iamErr.Message
			details.Code = iamErr.Code
		} else {
			details.GRPCCode = codes.Unknown
			details.Message = err.Error()
			details.Code = "UNKNOWN"
		}
	}

	// 分类
	details.Category = Categorize(err)

	// 客户端/服务端错误判断
	details.IsClientError = isClientError(details.GRPCCode)
	details.IsServerError = isServerError(details.GRPCCode)

	// 可重试性判断
	details.IsRetryable = isRetryableCode(details.GRPCCode)

	// 建议动作
	details.SuggestedAction = suggestAction(details.GRPCCode)

	// 限流时的重试时间
	if details.GRPCCode == codes.ResourceExhausted {
		details.RetryAfter = 5 * time.Second // 默认 5 秒
	}

	return details
}

func isClientError(code codes.Code) bool {
	switch code {
	case codes.InvalidArgument, codes.NotFound, codes.AlreadyExists,
		codes.PermissionDenied, codes.Unauthenticated, codes.FailedPrecondition,
		codes.OutOfRange:
		return true
	}
	return false
}

func isServerError(code codes.Code) bool {
	switch code {
	case codes.Internal, codes.Unknown, codes.DataLoss, codes.Unimplemented:
		return true
	}
	return false
}

func isRetryableCode(code codes.Code) bool {
	switch code {
	case codes.Unavailable, codes.ResourceExhausted, codes.Aborted, codes.DeadlineExceeded:
		return true
	}
	return false
}

func suggestAction(code codes.Code) ErrorAction {
	switch code {
	case codes.Unavailable, codes.Aborted:
		return ActionRetry
	case codes.DeadlineExceeded:
		return ActionFailover
	case codes.ResourceExhausted:
		return ActionRateLimit
	case codes.Unauthenticated:
		return ActionReauth
	case codes.PermissionDenied:
		return ActionForbidden
	case codes.NotFound:
		return ActionNotFound
	case codes.InvalidArgument, codes.FailedPrecondition, codes.OutOfRange:
		return ActionBadRequest
	case codes.Internal, codes.Unknown, codes.DataLoss:
		return ActionEscalate
	default:
		return ActionNone
	}
}

// =============================================================================
// 错误匹配器
// =============================================================================

// ErrorMatcher 错误匹配器接口
type ErrorMatcher interface {
	Match(err error) bool
}

// CodeMatcher 按 gRPC 状态码匹配
type CodeMatcher struct {
	codes []codes.Code
}

// NewCodeMatcher 创建状态码匹配器
func NewCodeMatcher(c ...codes.Code) *CodeMatcher {
	return &CodeMatcher{codes: c}
}

// Match 匹配错误
func (m *CodeMatcher) Match(err error) bool {
	code := GRPCCode(err)
	for _, c := range m.codes {
		if c == code {
			return true
		}
	}
	return false
}

// CategoryMatcher 按错误分类匹配
type CategoryMatcher struct {
	categories []ErrorCategory
}

// NewCategoryMatcher 创建分类匹配器
func NewCategoryMatcher(c ...ErrorCategory) *CategoryMatcher {
	return &CategoryMatcher{categories: c}
}

// Match 匹配错误
func (m *CategoryMatcher) Match(err error) bool {
	cat := Categorize(err)
	for _, c := range m.categories {
		if c == cat {
			return true
		}
	}
	return false
}

// =============================================================================
// 预定义匹配器
// =============================================================================

// AuthErrors 认证/授权相关错误匹配器
var AuthErrors = NewCodeMatcher(codes.Unauthenticated, codes.PermissionDenied)

// NetworkErrors 网络相关错误匹配器
var NetworkErrors = NewCodeMatcher(codes.Unavailable, codes.DeadlineExceeded, codes.Canceled)

// ServerErrors 服务端错误匹配器
var ServerErrors = NewCodeMatcher(codes.Internal, codes.Unknown, codes.DataLoss, codes.Unimplemented)

// ValidationErrors 参数验证错误匹配器
var ValidationErrors = NewCodeMatcher(codes.InvalidArgument, codes.FailedPrecondition, codes.OutOfRange)

// ResourceErrors 资源相关错误匹配器
var ResourceErrors = NewCodeMatcher(codes.NotFound, codes.AlreadyExists)

// RetryableErrors 可重试错误匹配器
var RetryableErrors = NewCodeMatcher(codes.Unavailable, codes.ResourceExhausted, codes.Aborted, codes.DeadlineExceeded)

// =============================================================================
// 错误处理助手
// =============================================================================

// HandleError 错误处理助手
type ErrorHandler struct {
	handlers map[codes.Code]func(error) error
	fallback func(error) error
}

// NewErrorHandler 创建错误处理器
func NewErrorHandler() *ErrorHandler {
	return &ErrorHandler{
		handlers: make(map[codes.Code]func(error) error),
		fallback: func(err error) error { return err },
	}
}

// On 注册特定状态码的处理器
func (h *ErrorHandler) On(code codes.Code, handler func(error) error) *ErrorHandler {
	h.handlers[code] = handler
	return h
}

// OnAuth 注册认证错误处理器
func (h *ErrorHandler) OnAuth(handler func(error) error) *ErrorHandler {
	h.handlers[codes.Unauthenticated] = handler
	h.handlers[codes.PermissionDenied] = handler
	return h
}

// OnNetwork 注册网络错误处理器
func (h *ErrorHandler) OnNetwork(handler func(error) error) *ErrorHandler {
	h.handlers[codes.Unavailable] = handler
	h.handlers[codes.DeadlineExceeded] = handler
	return h
}

// OnNotFound 注册资源不存在处理器
func (h *ErrorHandler) OnNotFound(handler func(error) error) *ErrorHandler {
	h.handlers[codes.NotFound] = handler
	return h
}

// Fallback 设置默认处理器
func (h *ErrorHandler) Fallback(handler func(error) error) *ErrorHandler {
	h.fallback = handler
	return h
}

// Handle 处理错误
func (h *ErrorHandler) Handle(err error) error {
	if err == nil {
		return nil
	}

	code := GRPCCode(err)
	if handler, ok := h.handlers[code]; ok {
		return handler(err)
	}
	return h.fallback(err)
}

// =============================================================================
// 便捷函数
// =============================================================================

// MustHandle 必须成功处理，否则 panic
func MustHandle(err error, handlers ...func(error) error) {
	if err == nil {
		return
	}
	for _, h := range handlers {
		if h(err) == nil {
			return
		}
	}
	panic(fmt.Sprintf("unhandled error: %v", err))
}

// IgnoreNotFound 忽略 NotFound 错误
func IgnoreNotFound(err error) error {
	if IsNotFound(err) {
		return nil
	}
	return err
}

// IgnoreAlreadyExists 忽略 AlreadyExists 错误
func IgnoreAlreadyExists(err error) error {
	if IsAlreadyExists(err) {
		return nil
	}
	return err
}

// AsIAMError 转换为 IAMError
func AsIAMError(err error) (*IAMError, bool) {
	var iamErr *IAMError
	if errors.As(err, &iamErr) {
		return iamErr, true
	}
	return nil, false
}

// ToHTTPStatus 转换为 HTTP 状态码
func ToHTTPStatus(err error) int {
	code := GRPCCode(err)
	switch code {
	case codes.OK:
		return 200
	case codes.InvalidArgument, codes.FailedPrecondition, codes.OutOfRange:
		return 400
	case codes.Unauthenticated:
		return 401
	case codes.PermissionDenied:
		return 403
	case codes.NotFound:
		return 404
	case codes.AlreadyExists, codes.Aborted:
		return 409
	case codes.ResourceExhausted:
		return 429
	case codes.Canceled:
		return 499
	case codes.Unimplemented:
		return 501
	case codes.Unavailable:
		return 503
	case codes.DeadlineExceeded:
		return 504
	default:
		return 500
	}
}
