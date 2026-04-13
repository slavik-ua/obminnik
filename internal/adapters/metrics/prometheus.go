package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type PrometheusMetrics struct {
	placementLatency *prometheus.HistogramVec
	matchingLatency  prometheus.Histogram
	e2eLatency       prometheus.Histogram
	tradesTotal      prometheus.Counter
	orderBookDepth   *prometheus.GaugeVec
}

func NewPrometheusMetrics() *PrometheusMetrics {
	return &PrometheusMetrics{
		placementLatency: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "exchange_order_placement_latency_seconds",
			Help:    "Time taken to validate order and write to outbox/db",
			Buckets: prometheus.DefBuckets,
		}, []string{"status"}),

		matchingLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "exchange_matching_engine_latency_seconds",
			Help:    "Time taken for the engine to process a match in memory",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1},
		}),

		e2eLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "exchange_order_e2e_latency_seconds",
			Help:    "Time taken from order creation in API to status update in Worker",
			Buckets: prometheus.DefBuckets,
		}),

		tradesTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "exchange_trades_total",
			Help: "Total number of successful trades executed",
		}),
	}
}

func (p *PrometheusMetrics) RecordOrderPlacement(d time.Duration, status string) {
	p.placementLatency.WithLabelValues(status).Observe(d.Seconds())
}

func (p *PrometheusMetrics) RecordMatchingLatency(d time.Duration) {
	p.matchingLatency.Observe(d.Seconds())
}

func (p *PrometheusMetrics) RecordEndToEndLatency(d time.Duration) {
	p.e2eLatency.Observe(d.Seconds())
}

func (p *PrometheusMetrics) RecordTrade(qty int64) {
	p.tradesTotal.Add(float64(qty))
}
