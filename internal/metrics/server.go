package metrics

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
	"zori/internal/config"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type MetricsServer struct {
	collector *MetricsCollector
	config    *config.Config
	server    *http.Server
	pusher    *MetricsPusher
	pushCtx   context.Context
	pushStop  context.CancelFunc
}

func NewMetricsServer(collector *MetricsCollector, cfg *config.Config) *MetricsServer {
	return &MetricsServer{
		collector: collector,
		config:    cfg,
	}
}

func (s *MetricsServer) Start() error {
	if !s.config.MetricsEnabled {
		log.Println("Metrics server is disabled")
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
		w.Write([]byte("OK"))
	})

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%s", s.config.MetricsPort),
		Handler: mux,
	}

	log.Printf("Starting metrics server on :%s", s.config.MetricsPort)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Metrics server error: %v", err)
		}
	}()

	if s.config.GrafanaCloudRemoteURL != "" {
		pusher, err := NewMetricsPusher(s.collector, s.config)
		if err != nil {
			log.Printf("Failed to create metrics pusher: %v", err)
			log.Println("Continuing without remote write to Grafana Cloud")
		} else {
			s.pusher = pusher
			s.pushCtx, s.pushStop = context.WithCancel(context.Background())
			go s.pusher.Start(s.pushCtx)
		}
	} else {
		log.Println("Grafana Cloud remote write URL not configured, skipping metrics push")
	}

	return nil
}

func (s *MetricsServer) Stop(ctx context.Context) error {
	if s.pusher != nil && s.pushStop != nil {
		log.Println("Stopping metrics pusher...")
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
