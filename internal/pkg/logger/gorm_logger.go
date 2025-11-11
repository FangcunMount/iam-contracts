package logger

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	gormlogger "gorm.io/gorm/logger"

	"github.com/FangcunMount/component-base/pkg/log"
)

// 定义 log 级别
const (
	Silent gormlogger.LogLevel = iota + 1
	Error
	Warn
	Info
)

// Config 定义 gorm 日志配置
type Config struct {
	SlowThreshold time.Duration
	Colorful      bool
	LogLevel      gormlogger.LogLevel
}

// NewGormLogger 创建 GORM 日志适配器
// 将 GORM 的日志输出适配到 component-base 的类型化日志系统
func NewGormLogger(level int) gormlogger.Interface {
	config := Config{
		SlowThreshold: 200 * time.Millisecond,
		Colorful:      true,
		LogLevel:      gormlogger.LogLevel(level),
	}

	return &logger{
		Config: config,
	}
}

type logger struct {
	Config
}

// LogMode 设置日志级别
func (l *logger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	newlogger := *l
	newlogger.LogLevel = level

	return &newlogger
}

// Info 打印 info 日志
func (l logger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= Info {
		l.logWithLevel(ctx, Info, msg, data...)
	}
}

// Warn 打印 warn 日志
func (l logger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= Warn {
		l.logWithLevel(ctx, Warn, msg, data...)
	}
}

// Error 打印 error 日志
func (l logger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= Error {
		l.logWithLevel(ctx, Error, msg, data...)
	}
}

// Trace 打印 sql 日志
func (l logger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= 0 {
		return
	}

	elapsed := time.Since(begin)
	switch {
	case err != nil && l.LogLevel >= Error:
		sql, rows := fc()
		fields := l.traceFields(ctx, sql, rows, elapsed)
		fields = append(fields, log.String("error", err.Error()))
		log.SQLError("GORM trace failed", fields...)
	case elapsed > l.SlowThreshold && l.SlowThreshold != 0 && l.LogLevel >= Warn:
		sql, rows := fc()
		fields := l.traceFields(ctx, sql, rows, elapsed)
		fields = append(fields, log.String("event", "slow_query"), log.Duration("slow_threshold", l.SlowThreshold))
		log.SQLWarn("GORM slow query", fields...)
	case l.LogLevel >= Info:
		sql, rows := fc()
		fields := l.traceFields(ctx, sql, rows, elapsed)
		log.SQLDebug("GORM trace", fields...)
	}
}

func (l logger) logWithLevel(ctx context.Context, level gormlogger.LogLevel, msg string, data ...interface{}) {
	formatted := msg
	if len(data) > 0 {
		formatted = fmt.Sprintf(msg, data...)
	}

	fields := []log.Field{
		log.String("caller", fileWithLineNum()),
		log.String("message", formatted),
	}
	fields = append(fields, log.TraceFields(ctx)...)

	switch level {
	case Error:
		log.SQLError("GORM error", fields...)
	case Warn:
		log.SQLWarn("GORM warning", fields...)
	default:
		log.SQL("GORM info", fields...)
	}
}

func (l logger) traceFields(ctx context.Context, sql string, rows int64, elapsed time.Duration) []log.Field {
	fields := []log.Field{
		log.String("caller", fileWithLineNum()),
		log.String("sql", sql),
		log.Float64("elapsed_ms", float64(elapsed.Nanoseconds())/1e6),
	}

	if rows >= 0 {
		fields = append(fields, log.Int64("rows", rows))
	} else {
		fields = append(fields, log.String("rows", "-1"))
	}

	fields = append(fields, log.TraceFields(ctx)...)
	return fields
}

// fileWithLineNum 获取文件名和行号
func fileWithLineNum() string {
	for i := 4; i < 15; i++ {
		_, file, line, ok := runtime.Caller(i)

		// if ok && (!strings.HasPrefix(file, gormSourceDir) || strings.HasSuffix(file, "_test.go")) {
		if ok && !strings.HasSuffix(file, "_test.go") {
			dir, f := filepath.Split(file)

			return filepath.Join(filepath.Base(dir), f) + ":" + strconv.FormatInt(int64(line), 10)
		}
	}

	return ""
}
