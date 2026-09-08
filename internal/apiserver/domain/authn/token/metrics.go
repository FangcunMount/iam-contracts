package token

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var legacyRefreshContextFallbackTotal = promauto.NewCounter(prometheus.CounterOpts{
	Name: "iam_legacy_refresh_context_fallback_total",
	Help: "Count of refresh flows that fell back to RefreshToken auth context because Session lacked it.",
})

var audienceFailures = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "iam_token_audience_failure_total", Help: "Audience constraint failures by bounded category.",
}, []string{"reason"})
