package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type MetricsCollector struct {
	IngestRequestsTotal   *prometheus.CounterVec
	IngestRequestDuration *prometheus.HistogramVec
	IngestErrorsTotal     *prometheus.CounterVec
	EventDedupeTotal      *prometheus.CounterVec

	EventsTotal *prometheus.CounterVec

	NatsMessageProcessed  *prometheus.CounterVec
	NatsMessageDuration   *prometheus.HistogramVec
	NatsMessageAckTotal   *prometheus.CounterVec
	NatsConsumerLagTotal  *prometheus.GaugeVec
	NatsStagesDuration    *prometheus.HistogramVec
	NatsStagesErrorsTotal *prometheus.CounterVec

	ClickhouseInsertDuration *prometheus.HistogramVec
	ClickhouseInsertErrors   *prometheus.CounterVec

	Registry *prometheus.Registry
}

func NewMetricsCollector() *MetricsCollector {
	registry := prometheus.NewRegistry()
	factory := promauto.With(registry)

	return &MetricsCollector{
		IngestRequestsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "zori_ingest_requests_total",
				Help: "Total number of ingestion requests",
			},
			[]string{"project_id", "organization_id", "status"},
		),
		IngestRequestDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "zori_ingest_request_duration_seconds",
				Help:    "Duration of ingestion requests in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"project_id", "organization_id"},
		),
		IngestErrorsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "zori_ingest_errors_total",
				Help: "Total number of ingestion errors",
			},
			[]string{"project_id", "organization_id", "error_type"},
		),
		EventDedupeTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "zori_event_dedupe_total",
				Help: "Total number of events processed by deduplication (new vs duplicate)",
			},
			[]string{"project_id", "organization_id", "result"},
		),

		EventsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "zori_events_total",
				Help: "Total number of events by type",
			},
			[]string{"project_id", "organization_id", "event_name", "event_type"},
		),

		NatsMessageProcessed: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "zori_nats_messages_processed_total",
				Help: "Total number of NATS messages processed",
			},
			[]string{"stream", "consumer", "status"},
		),
		NatsMessageDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "zori_nats_message_processing_duration_seconds",
				Help:    "Duration of NATS message processing in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"stream", "consumer"},
		),
		NatsMessageAckTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "zori_nats_message_ack_total",
				Help: "Total number of NATS message acknowledgments",
			},
			[]string{"stream", "consumer", "ack_type"},
		),
		NatsConsumerLagTotal: factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "zori_nats_consumer_lag_messages",
				Help: "Number of messages pending in NATS consumer",
			},
			[]string{"stream", "consumer"},
		),
		NatsStagesDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "zori_nats_stage_duration_seconds",
				Help:    "Duration of each processing stage in seconds",
				Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
			},
			[]string{"stage"},
		),
		NatsStagesErrorsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "zori_nats_stage_errors_total",
				Help: "Total number of errors in processing stages",
			},
			[]string{"stage"},
		),

		ClickhouseInsertDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "zori_clickhouse_insert_duration_seconds",
				Help:    "Duration of ClickHouse inserts in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"table"},
		),
		ClickhouseInsertErrors: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "zori_clickhouse_insert_errors_total",
				Help: "Total number of ClickHouse insert errors",
			},
			[]string{"table", "error_type"},
		),

		Registry: registry,
	}
}
