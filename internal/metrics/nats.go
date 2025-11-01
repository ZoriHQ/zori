package metrics

import (
	"time"
)

type NatsMetrics struct {
	collector *MetricsCollector
}

func NewNatsMetrics(collector *MetricsCollector) *NatsMetrics {
	return &NatsMetrics{
		collector: collector,
	}
}

func (m *NatsMetrics) RecordMessageProcessed(stream, consumer, status string, duration time.Duration) {
	m.collector.NatsMessageProcessed.WithLabelValues(stream, consumer, status).Inc()
	m.collector.NatsMessageDuration.WithLabelValues(stream, consumer).Observe(duration.Seconds())
}

func (m *NatsMetrics) RecordMessageAck(stream, consumer, ackType string) {
	// ackType can be: "ack", "nak"
	m.collector.NatsMessageAckTotal.WithLabelValues(stream, consumer, ackType).Inc()
}

func (m *NatsMetrics) UpdateConsumerLag(stream, consumer string, lag float64) {
	m.collector.NatsConsumerLagTotal.WithLabelValues(stream, consumer).Set(lag)
}

func (m *NatsMetrics) RecordStageDuration(stage string, duration time.Duration) {
	m.collector.NatsStagesDuration.WithLabelValues(stage).Observe(duration.Seconds())
}

func (m *NatsMetrics) RecordStageError(stage string) {
	m.collector.NatsStagesErrorsTotal.WithLabelValues(stage).Inc()
}

func (m *NatsMetrics) RecordClickHouseInsert(table string, duration time.Duration) {
	m.collector.ClickhouseInsertDuration.WithLabelValues(table).Observe(duration.Seconds())
}

func (m *NatsMetrics) RecordClickHouseError(table, errorType string) {
	m.collector.ClickhouseInsertErrors.WithLabelValues(table, errorType).Inc()
}

type StageTimer struct {
	metrics   *NatsMetrics
	stageName string
	startTime time.Time
}

func (m *NatsMetrics) NewStageTimer(stageName string) *StageTimer {
	return &StageTimer{
		metrics:   m,
		stageName: stageName,
		startTime: time.Now(),
	}
}

func (t *StageTimer) Done() {
	t.metrics.RecordStageDuration(t.stageName, time.Since(t.startTime))
}

func (t *StageTimer) Error() {
	t.metrics.RecordStageError(t.stageName)
	t.Done()
}
