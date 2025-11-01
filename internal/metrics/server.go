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

	return nil
}

func (s *MetricsServer) Stop(ctx context.Context) error {
	if s.server == nil {
		return nil
	}

	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return s.server.Shutdown(shutdownCtx)
}
