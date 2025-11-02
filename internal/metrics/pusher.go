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
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	defer close(p.doneCh)

	log.Println("Starting metrics pusher to Grafana Cloud")

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
	metricFamilies, err := p.collector.Registry.Gather()
	if err != nil {
		return fmt.Errorf("failed to gather metrics: %w", err)
	}

	timeSeries := p.convertToTimeSeries(metricFamilies)
	if len(timeSeries) == 0 {
		return nil
	}

	_, err = p.client.WriteTimeSeries(context.Background(), timeSeries, promremote.WriteOptions{})
	if err != nil {
		return fmt.Errorf("failed to write time series: %w", err)
	}

	return nil
}

func (p *MetricsPusher) convertToTimeSeries(metricFamilies []*dto.MetricFamily) []promremote.TimeSeries {
	var timeSeries []promremote.TimeSeries
	now := time.Now()

	for _, mf := range metricFamilies {
		metricName := mf.GetName()
		metricType := mf.GetType()

		for _, m := range mf.GetMetric() {
			labels := []promremote.Label{
				{
					Name:  "__name__",
					Value: metricName,
				},
			}

			for _, l := range m.GetLabel() {
				labels = append(labels, promremote.Label{
					Name:  l.GetName(),
					Value: l.GetValue(),
				})
			}

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
				if m.Histogram != nil {
					countLabels := append([]promremote.Label{}, labels...)
					countLabels[0].Value = metricName + "_count"
					timeSeries = append(timeSeries, promremote.TimeSeries{
						Labels: countLabels,
						Datapoint: promremote.Datapoint{
							Timestamp: now,
							Value:     float64(m.Histogram.GetSampleCount()),
						},
					})

					sumLabels := append([]promremote.Label{}, labels...)
					sumLabels[0].Value = metricName + "_sum"
					timeSeries = append(timeSeries, promremote.TimeSeries{
						Labels: sumLabels,
						Datapoint: promremote.Datapoint{
							Timestamp: now,
							Value:     m.Histogram.GetSampleSum(),
						},
					})

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
				if m.Summary != nil {
					countLabels := append([]promremote.Label{}, labels...)
					countLabels[0].Value = metricName + "_count"
					timeSeries = append(timeSeries, promremote.TimeSeries{
						Labels: countLabels,
						Datapoint: promremote.Datapoint{
							Timestamp: now,
							Value:     float64(m.Summary.GetSampleCount()),
						},
					})

					sumLabels := append([]promremote.Label{}, labels...)
					sumLabels[0].Value = metricName + "_sum"
					timeSeries = append(timeSeries, promremote.TimeSeries{
						Labels: sumLabels,
						Datapoint: promremote.Datapoint{
							Timestamp: now,
							Value:     m.Summary.GetSampleSum(),
						},
					})

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
