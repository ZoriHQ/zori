package metrics

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
	"zori/internal/config"

	"github.com/m3db/prometheus_remote_client_golang/promremote"
	dto "github.com/prometheus/client_model/go"
)

type MetricsPusher struct {
	client    promremote.Client
	collector *MetricsCollector
	config    *config.Config
	stopCh    chan struct{}
	doneCh    chan struct{}
}

// authRoundTripper adds basic authentication to HTTP requests
type authRoundTripper struct {
	username string
	password string
	rt       http.RoundTripper
}

func (a *authRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.SetBasicAuth(a.username, a.password)
	return a.rt.RoundTrip(req)
}

func NewMetricsPusher(collector *MetricsCollector, cfg *config.Config) (*MetricsPusher, error) {
	if cfg.GrafanaCloudRemoteURL == "" {
		return nil, fmt.Errorf("grafana cloud remote write URL is not configured")
	}

	// Create HTTP client with basic auth for Grafana Cloud
	var httpClient *http.Client
	if cfg.GrafanaCloudUsername != "" && cfg.GrafanaCloudAPIKey != "" {
		httpClient = &http.Client{
			Timeout: 30 * time.Second,
			Transport: &authRoundTripper{
				username: cfg.GrafanaCloudUsername,
				password: cfg.GrafanaCloudAPIKey,
				rt:       http.DefaultTransport,
			},
		}
	} else {
		httpClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	// Create remote write client with Grafana Cloud credentials
	clientCfg := promremote.NewConfig(
		promremote.WriteURLOption(cfg.GrafanaCloudRemoteURL),
		promremote.HTTPClientOption(httpClient),
		promremote.UserAgent("zori-metrics-pusher/1.0"),
	)

	client, err := promremote.NewClient(clientCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create remote write client: %w", err)
	}

	return &MetricsPusher{
		client:    client,
		collector: collector,
		config:    cfg,
		stopCh:    make(chan struct{}),
		doneCh:    make(chan struct{}),
	}, nil
}

func (p *MetricsPusher) Start(ctx context.Context) {
	// Push metrics every 15 seconds
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	defer close(p.doneCh)

	log.Println("Starting metrics pusher to Grafana Cloud")

	// Push immediately on start
	if err := p.pushMetrics(); err != nil {
		log.Printf("Failed to push metrics: %v", err)
	}

	for {
		select {
		case <-ticker.C:
			if err := p.pushMetrics(); err != nil {
				log.Printf("Failed to push metrics: %v", err)
			}
		case <-p.stopCh:
			log.Println("Stopping metrics pusher")
			return
		case <-ctx.Done():
			log.Println("Context cancelled, stopping metrics pusher")
			return
		}
	}
}

func (p *MetricsPusher) Stop() {
	close(p.stopCh)
	<-p.doneCh
}

func (p *MetricsPusher) pushMetrics() error {
	// Gather all metrics from the registry
	metricFamilies, err := p.collector.Registry.Gather()
	if err != nil {
		return fmt.Errorf("failed to gather metrics: %w", err)
	}

	// Convert Prometheus metrics to remote write format
	timeSeries := p.convertToTimeSeries(metricFamilies)
	if len(timeSeries) == 0 {
		log.Println("No metrics to push")
		return nil
	}

	// Push to Grafana Cloud
	_, err = p.client.WriteTimeSeries(context.Background(), timeSeries, promremote.WriteOptions{})
	if err != nil {
		return fmt.Errorf("failed to write time series: %w", err)
	}

	log.Printf("Successfully pushed %d time series to Grafana Cloud", len(timeSeries))
	return nil
}

func (p *MetricsPusher) convertToTimeSeries(metricFamilies []*dto.MetricFamily) []promremote.TimeSeries {
	var timeSeries []promremote.TimeSeries
	now := time.Now()

	for _, mf := range metricFamilies {
		metricName := mf.GetName()
		metricType := mf.GetType()

		for _, m := range mf.GetMetric() {
			// Base labels - add __name__ label
			labels := []promremote.Label{
				{
					Name:  "__name__",
					Value: metricName,
				},
			}

			// Add metric labels
			for _, l := range m.GetLabel() {
				labels = append(labels, promremote.Label{
					Name:  l.GetName(),
					Value: l.GetValue(),
				})
			}

			// Extract value based on metric type
			var value float64
			var hasValue bool

			switch metricType {
			case dto.MetricType_COUNTER:
				if m.Counter != nil {
					value = m.Counter.GetValue()
					hasValue = true
				}
			case dto.MetricType_GAUGE:
				if m.Gauge != nil {
					value = m.Gauge.GetValue()
					hasValue = true
				}
			case dto.MetricType_UNTYPED:
				if m.Untyped != nil {
					value = m.Untyped.GetValue()
					hasValue = true
				}
			case dto.MetricType_HISTOGRAM:
				// For histograms, send count, sum, and buckets
				if m.Histogram != nil {
					// Send count
					countLabels := append([]promremote.Label{}, labels...)
					countLabels[0].Value = metricName + "_count"
					timeSeries = append(timeSeries, promremote.TimeSeries{
						Labels: countLabels,
						Datapoint: promremote.Datapoint{
							Timestamp: now,
							Value:     float64(m.Histogram.GetSampleCount()),
						},
					})

					// Send sum
					sumLabels := append([]promremote.Label{}, labels...)
					sumLabels[0].Value = metricName + "_sum"
					timeSeries = append(timeSeries, promremote.TimeSeries{
						Labels: sumLabels,
						Datapoint: promremote.Datapoint{
							Timestamp: now,
							Value:     m.Histogram.GetSampleSum(),
						},
					})

					// Send buckets
					for _, bucket := range m.Histogram.GetBucket() {
						bucketLabels := append([]promremote.Label{}, labels...)
						bucketLabels[0].Value = metricName + "_bucket"
						bucketLabels = append(bucketLabels, promremote.Label{
							Name:  "le",
							Value: fmt.Sprintf("%v", bucket.GetUpperBound()),
						})
						timeSeries = append(timeSeries, promremote.TimeSeries{
							Labels: bucketLabels,
							Datapoint: promremote.Datapoint{
								Timestamp: now,
								Value:     float64(bucket.GetCumulativeCount()),
							},
						})
					}
				}
				continue
			case dto.MetricType_SUMMARY:
				// For summaries, send count and sum
				if m.Summary != nil {
					// Send count
					countLabels := append([]promremote.Label{}, labels...)
					countLabels[0].Value = metricName + "_count"
					timeSeries = append(timeSeries, promremote.TimeSeries{
						Labels: countLabels,
						Datapoint: promremote.Datapoint{
							Timestamp: now,
							Value:     float64(m.Summary.GetSampleCount()),
						},
					})

					// Send sum
					sumLabels := append([]promremote.Label{}, labels...)
					sumLabels[0].Value = metricName + "_sum"
					timeSeries = append(timeSeries, promremote.TimeSeries{
						Labels: sumLabels,
						Datapoint: promremote.Datapoint{
							Timestamp: now,
							Value:     m.Summary.GetSampleSum(),
						},
					})

					// Send quantiles
					for _, quantile := range m.Summary.GetQuantile() {
						quantileLabels := append([]promremote.Label{}, labels...)
						quantileLabels = append(quantileLabels, promremote.Label{
							Name:  "quantile",
							Value: fmt.Sprintf("%v", quantile.GetQuantile()),
						})
						timeSeries = append(timeSeries, promremote.TimeSeries{
							Labels: quantileLabels,
							Datapoint: promremote.Datapoint{
								Timestamp: now,
								Value:     quantile.GetValue(),
							},
						})
					}
				}
				continue
			}

			// Add simple metric types (counter, gauge, untyped)
			if hasValue {
				timeSeries = append(timeSeries, promremote.TimeSeries{
					Labels: labels,
					Datapoint: promremote.Datapoint{
						Timestamp: now,
						Value:     value,
					},
				})
			}
		}
	}

	return timeSeries
}
