package metrics

import (
	"time"
)

type IngestMetrics struct {
	collector *MetricsCollector
}

func NewIngestMetrics(collector *MetricsCollector) *IngestMetrics {
	return &IngestMetrics{
		collector: collector,
	}
}

func (m *IngestMetrics) RecordIngestRequest(projectID, organizationID, status string, duration time.Duration) {
	m.collector.IngestRequestsTotal.WithLabelValues(projectID, organizationID, status).Inc()
	m.collector.IngestRequestDuration.WithLabelValues(projectID, organizationID).Observe(duration.Seconds())
}

func (m *IngestMetrics) RecordIngestError(projectID, organizationID, errorType string) {
	m.collector.IngestErrorsTotal.WithLabelValues(projectID, organizationID, errorType).Inc()
}

func (m *IngestMetrics) RecordEventDedupe(projectID, organizationID, result string) {
	m.collector.EventDedupeTotal.WithLabelValues(projectID, organizationID, result).Inc()
}

func (m *IngestMetrics) RecordEvent(projectID, organizationID, eventName, eventType string) {
	m.collector.EventsTotal.WithLabelValues(projectID, organizationID, eventName, eventType).Inc()
}
