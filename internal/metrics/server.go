package metrics

import (
	"context"
	"fmt"
	"net/http"
	"time"
	"zori/internal/config"
	"zori/internal/telemetry"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type MetricsServer struct {
	collector *MetricsCollector
	config    *config.Config
	logger    *telemetry.Logger
	server    *http.Server
	pusher    *MetricsPusher
	pushCtx   context.Context
	pushStop  context.CancelFunc
}

func NewMetricsServer(collector *MetricsCollector, cfg *config.Config, logger *telemetry.Logger) *MetricsServer {
	return &MetricsServer{
		collector: collector,
		config:    cfg,
		logger:    logger,
	}
}

func (s *MetricsServer) Start() error {
	if !s.config.MetricsEnabled {
		s.logger.Info("Metrics server is disabled")
		return nil
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(
		s.collector.Registry,
		promhttp.HandlerOpts{
			EnableOpenMetrics: true,
		},
	))

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			s.logger.Error("Failed to write health check response", telemetry.Error(err))
		}
	})

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%s", s.config.MetricsPort),
		Handler: mux,
	}

	s.logger.Info("Starting metrics server", telemetry.String("port", s.config.MetricsPort))
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("Metrics server error", telemetry.Error(err))
		}
	}()

	if s.config.GrafanaCloudRemoteURL != "" {
		pusher, err := NewMetricsPusher(s.collector, s.config, s.logger)
		if err != nil {
			s.logger.Warn("Failed to create metrics pusher, continuing without remote write", telemetry.Error(err))
		} else {
			s.pusher = pusher
			s.pushCtx, s.pushStop = context.WithCancel(context.Background())
			go s.pusher.Start(s.pushCtx)
		}
	} else {
		s.logger.Info("Grafana Cloud remote write URL not configured, skipping metrics push")
	}

	return nil
}

func (s *MetricsServer) Stop(ctx context.Context) error {
	if s.pusher != nil && s.pushStop != nil {
		s.logger.Info("Stopping metrics pusher")
		s.pushStop()
		s.pusher.Stop()
	}

	if s.server == nil {
		return nil
	}

	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return s.server.Shutdown(shutdownCtx)
}
