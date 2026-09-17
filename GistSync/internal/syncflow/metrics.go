package syncflow

import (
	"log"
	"sync/atomic"
	"time"
)

type MetricsObserver interface {
	Record(action string, duration time.Duration, success bool, manifestBytes int, snapshotCount int)
}

type metricsObserver struct {
	totalCalls int64
	totalFails int64
}

func newMetricsObserver() MetricsObserver {
	return &metricsObserver{}
}

func (m *metricsObserver) Record(action string, duration time.Duration, success bool, manifestBytes int, snapshotCount int) {
	calls := atomic.AddInt64(&m.totalCalls, 1)
	fails := atomic.LoadInt64(&m.totalFails)
	if !success {
		fails = atomic.AddInt64(&m.totalFails, 1)
	}
	failureRate := float64(fails) / float64(calls)
	log.Printf(
		"[syncflow] action=%s duration_ms=%d success=%t calls=%d fail_rate=%.4f manifest_bytes=%d snapshot_count=%d",
		action,
		duration.Milliseconds(),
		success,
		calls,
		failureRate,
		manifestBytes,
		snapshotCount,
	)
}
