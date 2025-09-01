package processor

import (
	"fmt"
	"log"
	"time"

	"github.com/meiking/tidb-metrics-crawler/pkg/common"
	"github.com/meiking/tidb-metrics-crawler/pkg/config"
	"github.com/meiking/tidb-metrics-crawler/pkg/prometheus"
	"github.com/meiking/tidb-metrics-crawler/pkg/sink"
	"github.com/prometheus/common/model"
)

// Processor handles data processing and coordination
type Processor struct {
	clients []prometheus.Client
	sink    sink.Sink
}

// NewProcessor creates a new data processor
func NewProcessor(clients []prometheus.Client, outputSink sink.Sink) *Processor {
	return &Processor{
		clients: clients,
		sink:    outputSink,
	}
}

// ProcessMetrics coordinates fetching and processing of all metrics
func (p *Processor) ProcessMetrics(metrics []config.MetricConfig, start, end time.Time, stepStr string) error {
	step, err := time.ParseDuration(stepStr)
	if err != nil {
		return fmt.Errorf("invalid step duration: %v", err)
	}

	// Process each metric for each Prometheus instance
	for _, metric := range metrics {
		log.Printf("Processing metric: %s", metric.Name)

		for _, client := range p.clients {
			log.Printf("Fetching from Prometheus instance: %s", client.Name())

			// Fetch metric data
			result, err := client.FetchRange(metric.Query, start, end, step)
			if err != nil {
				log.Printf("Error fetching metric %s from %s: %v", metric.Name, client.Name(), err)
				continue
			}

			// Process the result
			processedData, err := p.processResult(client.Name(), metric, result)
			if err != nil {
				log.Printf("Error processing results for %s: %v", metric.Name, err)
				continue
			}

			// Write to sink
			if err := p.sink.Write(metric.Name, processedData); err != nil {
				log.Printf("Error writing data to sink for %s: %v", metric.Name, err)
			}
		}
	}

	return nil
}

// processResult converts Prometheus response to common.ProcessedData
func (p *Processor) processResult(instanceName string, metric config.MetricConfig, result model.Value) ([]common.ProcessedData, error) {
	var data []common.ProcessedData

	// Check if we got a matrix result (expected for range queries)
	matrix, ok := result.(model.Matrix)
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", result)
	}

	// Process each time series in the matrix
	for _, series := range matrix {
		// Extract relevant labels
		labels := make(map[string]string)
		for _, key := range metric.LabelKeys {
			if val, exists := series.Metric[model.LabelName(key)]; exists {
				labels[key] = string(val)
			}
		}

		// Process each data point in the time series
		for _, point := range series.Values {
			timestamp := time.Unix(int64(point.Timestamp)/1000, 0)
			value := float64(point.Value)

			data = append(data, common.ProcessedData{
				PrometheusInstance: instanceName,
				MetricName:         metric.Name,
				Timestamp:          timestamp,
				Value:              value,
				Labels:             labels,
			})
		}
	}

	return data, nil
}
