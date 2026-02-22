package middleware

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	HttpRequestCountTotal      *prometheus.CounterVec
	HttpRequestDurationSeconds *prometheus.HistogramVec
)

func init() {
	HttpRequestCountTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "library_http_request_count_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	HttpRequestDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "library_http_request_duration_seconds",
			Help:    "HTTP request latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	prometheus.MustRegister(HttpRequestCountTotal)
	prometheus.MustRegister(HttpRequestDurationSeconds)
}
