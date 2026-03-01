package middleware_test

import (
	"github.com/Dailiduzhou/library_manage_sys/middleware"
	"testing"
)

func TestMetricsRegistered(t *testing.T) {
	counter := middleware.HttpRequestCountTotal
	histogram := middleware.HttpRequestDurationSeconds

	if counter == nil {
		t.Fatal("Counter metric is nil")
	}

	if histogram == nil {
		t.Fatal("Histogram metric is nil")
	}

	t.Log("Metrics are registered successfully")
}
