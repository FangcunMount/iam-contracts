// Package observability 提供可观测性支持
package observability

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// =============================================================================
// Prometheus 具体实现
// =============================================================================

// PrometheusMetrics Prometheus 指标收集器实现
type PrometheusMetrics struct {
	requestsTotal    *prometheus.CounterVec
	requestDuration  *prometheus.HistogramVec
	requestsInFlight *prometheus.GaugeVec
	errorsTotal      *prometheus.CounterVec

	registered bool
}

// PrometheusMetricsOption Prometheus 指标配置选项
type PrometheusMetricsOption func(*prometheusMetricsConfig)

type prometheusMetricsConfig struct {
	namespace   string
	subsystem   string
	constLabels prometheus.Labels
	buckets     []float64
	registerer  prometheus.Registerer
}

// WithPrometheusNamespace 设置命名空间
func WithPrometheusNamespace(ns string) PrometheusMetricsOption {
	return func(c *prometheusMetricsConfig) {
		c.namespace = ns
	}
}

// WithPrometheusSubsystem 设置子系统
func WithPrometheusSubsystem(ss string) PrometheusMetricsOption {
	return func(c *prometheusMetricsConfig) {
		c.subsystem = ss
	}
}

// WithPrometheusConstLabels 设置常量标签
func WithPrometheusConstLabels(labels prometheus.Labels) PrometheusMetricsOption {
	return func(c *prometheusMetricsConfig) {
		c.constLabels = labels
	}
}

// WithPrometheusBuckets 设置直方图桶
func WithPrometheusBuckets(buckets []float64) PrometheusMetricsOption {
	return func(c *prometheusMetricsConfig) {
		c.buckets = buckets
	}
}

// WithPrometheusRegisterer 设置注册器
func WithPrometheusRegisterer(r prometheus.Registerer) PrometheusMetricsOption {
	return func(c *prometheusMetricsConfig) {
		c.registerer = r
	}
}

// NewPrometheusMetrics 创建 Prometheus 指标收集器
func NewPrometheusMetrics(opts ...PrometheusMetricsOption) *PrometheusMetrics {
	cfg := &prometheusMetricsConfig{
		namespace:  "iam",
		subsystem:  "sdk",
		buckets:    prometheus.DefBuckets,
		registerer: prometheus.DefaultRegisterer,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	m := &PrometheusMetrics{
		requestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace:   cfg.namespace,
				Subsystem:   cfg.subsystem,
				Name:        "requests_total",
				Help:        "Total number of gRPC requests",
				ConstLabels: cfg.constLabels,
			},
			[]string{"method", "code"},
		),
		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace:   cfg.namespace,
				Subsystem:   cfg.subsystem,
				Name:        "request_duration_seconds",
				Help:        "gRPC request duration in seconds",
				Buckets:     cfg.buckets,
				ConstLabels: cfg.constLabels,
			},
			[]string{"method"},
		),
		requestsInFlight: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   cfg.namespace,
				Subsystem:   cfg.subsystem,
				Name:        "requests_in_flight",
				Help:        "Number of gRPC requests currently in flight",
				ConstLabels: cfg.constLabels,
			},
			[]string{"method"},
		),
		errorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace:   cfg.namespace,
				Subsystem:   cfg.subsystem,
				Name:        "errors_total",
				Help:        "Total number of gRPC errors",
				ConstLabels: cfg.constLabels,
			},
			[]string{"method", "code"},
		),
	}

	return m
}

// Register 注册到 Prometheus
func (m *PrometheusMetrics) Register(registerer prometheus.Registerer) error {
	if registerer == nil {
		registerer = prometheus.DefaultRegisterer
	}

	collectors := []prometheus.Collector{
		m.requestsTotal,
		m.requestDuration,
		m.requestsInFlight,
		m.errorsTotal,
	}

	for _, c := range collectors {
		if err := registerer.Register(c); err != nil {
			// 如果已注册，忽略错误
			if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
				return err
			}
		}
	}

	m.registered = true
	return nil
}

// MustRegister 注册到 Prometheus（panic on error）
func (m *PrometheusMetrics) MustRegister(registerer prometheus.Registerer) {
	if err := m.Register(registerer); err != nil {
		panic(err)
	}
}

// RecordRequest 实现 MetricsCollector 接口
func (m *PrometheusMetrics) RecordRequest(method string, code string, duration time.Duration) {
	m.requestsTotal.WithLabelValues(method, code).Inc()
	m.requestDuration.WithLabelValues(method).Observe(duration.Seconds())

	if code != "OK" {
		m.errorsTotal.WithLabelValues(method, code).Inc()
	}
}

// IncInFlight 增加正在处理的请求计数
func (m *PrometheusMetrics) IncInFlight(method string) {
	m.requestsInFlight.WithLabelValues(method).Inc()
}

// DecInFlight 减少正在处理的请求计数
func (m *PrometheusMetrics) DecInFlight(method string) {
	m.requestsInFlight.WithLabelValues(method).Dec()
}

// Collectors 返回所有 Collector（用于自定义注册）
func (m *PrometheusMetrics) Collectors() []prometheus.Collector {
	return []prometheus.Collector{
		m.requestsTotal,
		m.requestDuration,
		m.requestsInFlight,
		m.errorsTotal,
	}
}
