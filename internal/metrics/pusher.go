package metrics

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
	"zori/internal/config"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/prometheus/prompb"
)

type MetricsPusher struct {
	httpClient *http.Client
	endpoint   string
	collector  *MetricsCollector
	config     *config.Config
	stopCh     chan struct{}
	doneCh     chan struct{}
}

func NewMetricsPusher(collector *MetricsCollector, cfg *config.Config) (*MetricsPusher, error) {
	if cfg.GrafanaCloudRemoteURL == "" {
		return nil, fmt.Errorf("grafana cloud remote write URL is not configured")
	}

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	return &MetricsPusher{
		httpClient: httpClient,
		endpoint:   cfg.GrafanaCloudRemoteURL,
		collector:  collector,
		config:     cfg,
		stopCh:     make(chan struct{}),
		doneCh:     make(chan struct{}),
	}, nil
}

func (p *MetricsPusher) Start(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	defer close(p.doneCh)

	log.Println("Starting metrics pusher to Grafana Cloud")

	if err := p.pushMetrics(ctx); err != nil {
		log.Printf("Failed to push metrics: %v", err)
	}

	for {
		select {
		case <-ticker.C:
			if err := p.pushMetrics(ctx); err != nil {
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

func (p *MetricsPusher) pushMetrics(ctx context.Context) error {
	metricFamilies, err := p.collector.Registry.Gather()
	if err != nil {
		return fmt.Errorf("failed to gather metrics: %w", err)
	}

	timeSeries := p.convertToTimeSeries(metricFamilies)
	if len(timeSeries) == 0 {
		return nil
	}

	writeRequest := &prompb.WriteRequest{
		Timeseries: timeSeries,
	}

	data, err := proto.Marshal(writeRequest)
	if err != nil {
		return fmt.Errorf("failed to marshal write request: %w", err)
	}

	compressed := snappy.Encode(nil, data)

	req, err := http.NewRequestWithContext(ctx, "POST", p.endpoint, bytes.NewReader(compressed))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Encoding", "snappy")
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0")
	req.Header.Set("User-Agent", "zori-metrics-pusher/1.0")

	if p.config.GrafanaCloudUsername != "" && p.config.GrafanaCloudAPIKey != "" {
		req.SetBasicAuth(p.config.GrafanaCloudUsername, p.config.GrafanaCloudAPIKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("remote write failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (p *MetricsPusher) convertToTimeSeries(metricFamilies []*dto.MetricFamily) []prompb.TimeSeries {
	var timeSeries []prompb.TimeSeries
	now := time.Now().UnixMilli()

	for _, mf := range metricFamilies {
		metricName := mf.GetName()
		metricType := mf.GetType()

		for _, m := range mf.GetMetric() {
			labels := []prompb.Label{
				{
					Name:  "__name__",
					Value: metricName,
				},
			}

			for _, l := range m.GetLabel() {
				labels = append(labels, prompb.Label{
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
					countLabels := make([]prompb.Label, len(labels))
					copy(countLabels, labels)
					countLabels[0].Value = metricName + "_count"
					timeSeries = append(timeSeries, prompb.TimeSeries{
						Labels: countLabels,
						Samples: []prompb.Sample{
							{
								Timestamp: now,
								Value:     float64(m.Histogram.GetSampleCount()),
							},
						},
					})

					sumLabels := make([]prompb.Label, len(labels))
					copy(sumLabels, labels)
					sumLabels[0].Value = metricName + "_sum"
					timeSeries = append(timeSeries, prompb.TimeSeries{
						Labels: sumLabels,
						Samples: []prompb.Sample{
							{
								Timestamp: now,
								Value:     m.Histogram.GetSampleSum(),
							},
						},
					})

					for _, bucket := range m.Histogram.GetBucket() {
						bucketLabels := make([]prompb.Label, len(labels)+1)
						copy(bucketLabels, labels)
						bucketLabels[0].Value = metricName + "_bucket"
						bucketLabels[len(bucketLabels)-1] = prompb.Label{
							Name:  "le",
							Value: fmt.Sprintf("%v", bucket.GetUpperBound()),
						}
						timeSeries = append(timeSeries, prompb.TimeSeries{
							Labels: bucketLabels,
							Samples: []prompb.Sample{
								{
									Timestamp: now,
									Value:     float64(bucket.GetCumulativeCount()),
								},
							},
						})
					}
				}
				continue
			case dto.MetricType_SUMMARY:
				if m.Summary != nil {
					countLabels := make([]prompb.Label, len(labels))
					copy(countLabels, labels)
					countLabels[0].Value = metricName + "_count"
					timeSeries = append(timeSeries, prompb.TimeSeries{
						Labels: countLabels,
						Samples: []prompb.Sample{
							{
								Timestamp: now,
								Value:     float64(m.Summary.GetSampleCount()),
							},
						},
					})

					sumLabels := make([]prompb.Label, len(labels))
					copy(sumLabels, labels)
					sumLabels[0].Value = metricName + "_sum"
					timeSeries = append(timeSeries, prompb.TimeSeries{
						Labels: sumLabels,
						Samples: []prompb.Sample{
							{
								Timestamp: now,
								Value:     m.Summary.GetSampleSum(),
							},
						},
					})

					for _, quantile := range m.Summary.GetQuantile() {
						quantileLabels := make([]prompb.Label, len(labels)+1)
						copy(quantileLabels, labels)
						quantileLabels[len(quantileLabels)-1] = prompb.Label{
							Name:  "quantile",
							Value: fmt.Sprintf("%v", quantile.GetQuantile()),
						}
						timeSeries = append(timeSeries, prompb.TimeSeries{
							Labels: quantileLabels,
							Samples: []prompb.Sample{
								{
									Timestamp: now,
									Value:     quantile.GetValue(),
								},
							},
						})
					}
				}
				continue
			}

			if hasValue {
				timeSeries = append(timeSeries, prompb.TimeSeries{
					Labels: labels,
					Samples: []prompb.Sample{
						{
							Timestamp: now,
							Value:     value,
						},
					},
				})
			}
		}
	}

	return timeSeries
}
