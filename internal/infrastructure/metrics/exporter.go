package metrics

import (
	"context"

	"github.com/epicsagas/korean-geocode/internal/infrastructure/ratelimit"
	"github.com/prometheus/client_golang/prometheus"
)

// Exporter는 Rate Limiter 메트릭을 Prometheus에 노출합니다
type Exporter struct {
	rateLimiter ratelimit.RateLimiter
	providers   []string

	// Prometheus metrics
	quotaGauge *prometheus.GaugeVec
	usageGauge *prometheus.GaugeVec
	remainingGauge *prometheus.GaugeVec
	utilizationGauge *prometheus.GaugeVec
}

// NewExporter는 새로운 Prometheus Exporter를 생성합니다
func NewExporter(rateLimiter ratelimit.RateLimiter, providers []string) *Exporter {
	return &Exporter{
		rateLimiter: rateLimiter,
		providers:   providers,
		quotaGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "geocode_quota_total",
				Help: "Total daily quota for each provider",
			},
			[]string{"provider"},
		),
		usageGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "geocode_usage_total",
				Help: "Current usage count for each provider",
			},
			[]string{"provider"},
		),
		remainingGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "geocode_quota_remaining",
				Help: "Remaining quota for each provider",
			},
			[]string{"provider"},
		),
		utilizationGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "geocode_quota_utilization_percent",
				Help: "Quota utilization percentage for each provider (0-100)",
			},
			[]string{"provider"},
		),
	}
}

// Describe implements prometheus.Collector
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	e.quotaGauge.Describe(ch)
	e.usageGauge.Describe(ch)
	e.remainingGauge.Describe(ch)
	e.utilizationGauge.Describe(ch)
}

// Collect implements prometheus.Collector
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	ctx := context.Background()

	for _, provider := range e.providers {
		// 쿼터와 사용량 조회
		quota, err := e.rateLimiter.GetQuota(ctx, provider)
		if err != nil || quota == 0 {
			continue
		}

		usage, err := e.rateLimiter.GetUsage(ctx, provider)
		if err != nil {
			continue
		}

		// 메트릭 설정
		remaining := quota - usage
		if remaining < 0 {
			remaining = 0
		}

		utilization := float64(usage) / float64(quota) * 100
		if utilization > 100 {
			utilization = 100
		}

		e.quotaGauge.WithLabelValues(provider).Set(float64(quota))
		e.usageGauge.WithLabelValues(provider).Set(float64(usage))
		e.remainingGauge.WithLabelValues(provider).Set(float64(remaining))
		e.utilizationGauge.WithLabelValues(provider).Set(utilization)
	}

	// 메트릭 수집
	e.quotaGauge.Collect(ch)
	e.usageGauge.Collect(ch)
	e.remainingGauge.Collect(ch)
	e.utilizationGauge.Collect(ch)
}

// Register registers the exporter with Prometheus
func (e *Exporter) Register() error {
	return prometheus.Register(e)
}
