package logger

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/FangcunMount/component-base/pkg/log"
	"github.com/redis/go-redis/v9"
)

// RedisHook Redis 命令执行钩子
// 用于记录 Redis 命令执行情况，类似于 GORM logger
type RedisHook struct {
	enabled       bool
	slowThreshold time.Duration
}

// 确保实现了接口
var _ redis.Hook = (*RedisHook)(nil)

// NewRedisHook 创建 Redis 钩子
// enabled: 是否启用日志记录
// slowThreshold: 慢命令阈值（超过此时间会记录警告）
func NewRedisHook(enabled bool, slowThreshold time.Duration) *RedisHook {
	if slowThreshold <= 0 {
		slowThreshold = 200 * time.Millisecond // 默认 200ms
	}

	return &RedisHook{
		enabled:       enabled,
		slowThreshold: slowThreshold,
	}
}

// DialHook 拨号钩子
func (h *RedisHook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		if !h.enabled {
			return next(ctx, network, addr)
		}

		start := time.Now()
		conn, err := next(ctx, network, addr)
		elapsed := time.Since(start)

		if err != nil {
			log.RedisError("Redis dial failed",
				log.String("network", network),
				log.String("addr", addr),
				log.Float64("elapsed_ms", float64(elapsed.Nanoseconds())/1e6),
				log.String("error", err.Error()),
			)
		} else {
			log.RedisDebug("Redis dial success",
				log.String("network", network),
				log.String("addr", addr),
				log.Float64("elapsed_ms", float64(elapsed.Nanoseconds())/1e6),
			)
		}

		return conn, err
	}
}

// ProcessHook 命令执行钩子
func (h *RedisHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		if !h.enabled {
			return next(ctx, cmd)
		}

		start := time.Now()
		err := next(ctx, cmd)
		elapsed := time.Since(start)

		h.logCommand(ctx, cmd, err, elapsed)
		return err
	}
}

// ProcessPipelineHook 管道命令钩子
func (h *RedisHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		if !h.enabled {
			return next(ctx, cmds)
		}

		start := time.Now()
		err := next(ctx, cmds)
		elapsed := time.Since(start)

		h.logPipeline(ctx, cmds, err, elapsed)
		return err
	}
}

// logCommand 记录单个命令执行
func (h *RedisHook) logCommand(ctx context.Context, cmd redis.Cmder, err error, elapsed time.Duration) {
	cmdStr := formatCommand(cmd)
	fields := []log.Field{
		log.String("command", cmdStr),
		log.Float64("elapsed_ms", float64(elapsed.Nanoseconds())/1e6),
	}

	// 添加分布式追踪字段
	fields = append(fields, log.TraceFields(ctx)...)

	switch {
	case err != nil && err != redis.Nil:
		// 命令执行错误（redis.Nil 表示 key 不存在，是正常情况）
		fields = append(fields, log.String("error", err.Error()))
		log.RedisError("Redis command failed", fields...)

	case elapsed > h.slowThreshold:
		// 慢命令警告
		fields = append(fields,
			log.String("event", "slow_command"),
			log.Duration("slow_threshold", h.slowThreshold),
		)
		log.RedisWarn("Redis slow command", fields...)

	default:
		// 正常执行
		log.RedisDebug("Redis command executed", fields...)
	}
}

// logPipeline 记录管道命令执行
func (h *RedisHook) logPipeline(ctx context.Context, cmds []redis.Cmder, err error, elapsed time.Duration) {
	cmdCount := len(cmds)
	fields := []log.Field{
		log.Int("command_count", cmdCount),
		log.Float64("elapsed_ms", float64(elapsed.Nanoseconds())/1e6),
	}

	// 添加分布式追踪字段
	fields = append(fields, log.TraceFields(ctx)...)

	// 记录每个命令的名称
	if cmdCount > 0 && cmdCount <= 10 {
		// 最多记录前 10 个命令
		cmdNames := make([]string, 0, cmdCount)
		for _, cmd := range cmds {
			cmdNames = append(cmdNames, cmd.Name())
		}
		fields = append(fields, log.String("commands", strings.Join(cmdNames, ", ")))
	}

	switch {
	case err != nil:
		// 管道执行错误
		fields = append(fields, log.String("error", err.Error()))
		log.RedisError("Redis pipeline failed", fields...)

	case elapsed > h.slowThreshold:
		// 慢管道警告
		fields = append(fields,
			log.String("event", "slow_pipeline"),
			log.Duration("slow_threshold", h.slowThreshold),
		)
		log.RedisWarn("Redis slow pipeline", fields...)

	default:
		// 正常执行
		log.RedisDebug("Redis pipeline executed", fields...)
	}
}

// formatCommand 格式化 Redis 命令，用于日志输出
func formatCommand(cmd redis.Cmder) string {
	args := cmd.Args()
	if len(args) == 0 {
		return cmd.Name()
	}

	// 构建命令字符串，对敏感参数进行脱敏
	parts := make([]string, 0, len(args))
	cmdName := strings.ToUpper(cmd.Name())
	parts = append(parts, cmdName)

	// 根据命令类型决定是否脱敏
	needMask := false
	switch cmdName {
	case "AUTH", "HELLO":
		needMask = true
	}

	for i := 1; i < len(args); i++ {
		arg := fmt.Sprintf("%v", args[i])

		// 对敏感参数脱敏
		if needMask && i > 0 {
			arg = "***"
		} else if len(arg) > 100 {
			// 截断过长的参数
			arg = arg[:100] + "..."
		}

		parts = append(parts, arg)
	}

	result := strings.Join(parts, " ")
	if len(result) > 500 {
		result = result[:500] + "..."
	}

	return result
}
