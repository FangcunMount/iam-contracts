package logger

import "github.com/FangcunMount/component-base/pkg/log"

// ============================================================================
// 标准日志字段名称
// ============================================================================

// 追踪相关字段
const (
	FieldTraceID   = "trace_id"
	FieldSpanID    = "span_id"
	FieldRequestID = "request_id"
)

// 身份相关字段
const (
	FieldUserID    = "user_id"
	FieldAccountID = "account_id"
	FieldTenantID  = "tenant_id"
	FieldClientIP  = "client_ip"
	FieldUserAgent = "user_agent"
)

// 操作相关字段
const (
	FieldAction     = "action"
	FieldResource   = "resource"
	FieldResourceID = "resource_id"
	FieldResult     = "result"
)

// 性能相关字段
const (
	FieldDurationMS = "duration_ms"
	FieldLatency    = "latency"
)

// 错误相关字段
const (
	FieldError      = "error"
	FieldErrorCode  = "error_code"
	FieldStackTrace = "stack_trace"
)

// HTTP 相关字段
const (
	FieldMethod     = "method"
	FieldPath       = "path"
	FieldQuery      = "query"
	FieldStatusCode = "status_code"
)

// gRPC 相关字段
const (
	FieldGRPCMethod  = "grpc.method"
	FieldGRPCService = "grpc.service"
	FieldGRPCCode    = "grpc.code"
)

// ============================================================================
// 标准操作类型（用于审计日志）
// ============================================================================

const (
	ActionCreate   = "create"
	ActionRead     = "read"
	ActionUpdate   = "update"
	ActionDelete   = "delete"
	ActionList     = "list"
	ActionLogin    = "login"
	ActionLogout   = "logout"
	ActionRegister = "register"
	ActionVerify   = "verify"
	ActionRefresh  = "refresh"
	ActionRevoke   = "revoke"
	ActionBind     = "bind"
	ActionUnbind   = "unbind"
)

// ============================================================================
// 标准资源类型
// ============================================================================

const (
	ResourceUser         = "user"
	ResourceChild        = "child"
	ResourceGuardianship = "guardianship"
	ResourceCredential   = "credential"
	ResourceToken        = "token"
	ResourceSession      = "session"
)

// ============================================================================
// 标准结果类型
// ============================================================================

const (
	ResultSuccess = "success"
	ResultFailed  = "failed"
	ResultDenied  = "denied"
	ResultTimeout = "timeout"
)

// ============================================================================
// 日志事件类型（用于区分日志类别）
// ============================================================================

const (
	EventRequestStart = "request_start"
	EventRequestEnd   = "request_end"
	EventOperation    = "operation"
	EventAudit        = "audit"
	EventSecurity     = "security"
	EventPerformance  = "performance"
)

// ============================================================================
// 辅助函数：生成标准字段
// ============================================================================

// UserFields 生成用户相关字段
func UserFields(userID, accountID, tenantID string) []log.Field {
	fields := make([]log.Field, 0, 3)
	if userID != "" {
		fields = append(fields, log.String(FieldUserID, userID))
	}
	if accountID != "" {
		fields = append(fields, log.String(FieldAccountID, accountID))
	}
	if tenantID != "" {
		fields = append(fields, log.String(FieldTenantID, tenantID))
	}
	return fields
}

// OperationFields 生成操作相关字段
func OperationFields(action, resource, resourceID string) []log.Field {
	fields := []log.Field{
		log.String(FieldAction, action),
		log.String(FieldResource, resource),
	}
	if resourceID != "" {
		fields = append(fields, log.String(FieldResourceID, resourceID))
	}
	return fields
}

// ResultFields 生成操作结果字段
func ResultFields(result string, durationMS int64) []log.Field {
	return []log.Field{
		log.String(FieldResult, result),
		log.Int64(FieldDurationMS, durationMS),
	}
}

// ErrorFields 生成错误相关字段
func ErrorFields(err error, code int) []log.Field {
	fields := []log.Field{
		log.String(FieldError, err.Error()),
	}
	if code != 0 {
		fields = append(fields, log.Int(FieldErrorCode, code))
	}
	return fields
}

// HTTPFields 生成 HTTP 相关字段
func HTTPFields(method, path, query string, statusCode int) []log.Field {
	fields := []log.Field{
		log.String(FieldMethod, method),
		log.String(FieldPath, path),
	}
	if query != "" {
		fields = append(fields, log.String(FieldQuery, query))
	}
	if statusCode != 0 {
		fields = append(fields, log.Int(FieldStatusCode, statusCode))
	}
	return fields
}

// GRPCFields 生成 gRPC 相关字段
func GRPCFields(method, service, code string) []log.Field {
	fields := []log.Field{
		log.String(FieldGRPCMethod, method),
	}
	if service != "" {
		fields = append(fields, log.String(FieldGRPCService, service))
	}
	if code != "" {
		fields = append(fields, log.String(FieldGRPCCode, code))
	}
	return fields
}

// EventField 生成事件类型字段
func EventField(event string) log.Field {
	return log.String("event", event)
}
