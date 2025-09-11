package processor

import (
	"errors"
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

	// Validate time range
	if start.After(end) {
		return errors.New("start time must be before end time")
	}

	// Process each metric for each Prometheus instance
	for _, metric := range metrics {
		log.Printf("Processing metric: %s", metric.Name)

		for _, client := range p.clients {
			log.Printf("Processing Prometheus instance: %s", client.Name())

			// Fetch data in hourly batches
			if err := p.processInHourlyBatches(client, metric, start, end, step); err != nil {
				log.Printf("Error processing metric %s for instance %s: %v",
					metric.Name, client.Name(), err)
				// Continue with next client instead of failing entirely
				continue
			}
		}
	}

	return nil
}

// processInHourlyBatches splits the time range into 1-hour chunks and processes each
func (p *Processor) processInHourlyBatches(
	client prometheus.Client,
	metric config.MetricConfig,
	globalStart, globalEnd time.Time,
	step time.Duration,
) error {
	// Calculate total duration
	totalDuration := globalEnd.Sub(globalStart)
	log.Printf("Total time range: %v. Will split into hourly batches.", totalDuration)

	// Process each hourly batch
	currentStart := globalStart
	batchNumber := 1

	for currentStart.Before(globalEnd) {
		// Calculate end of current batch (1 hour later or global end, whichever comes first)
		currentEnd := currentStart.Add(1 * time.Hour)
		if currentEnd.After(globalEnd) {
			currentEnd = globalEnd
		}

		log.Printf("Processing batch %d: %s to %s",
			batchNumber,
			currentStart.Format(time.RFC3339),
			currentEnd.Format(time.RFC3339))

		// Fetch data for this batch
		result, err := client.FetchRange(metric.Query, currentStart, currentEnd, step)
		if err != nil {
			return fmt.Errorf("failed to fetch batch %d: %v", batchNumber, err)
		}

		// Process and write the batch data
		processedData, err := p.processBatchResult(client.Name(), metric.Name, metric.LabelKeys, result)
		if err != nil {
			return fmt.Errorf("failed to process batch %d results: %v", batchNumber, err)
		}

		if len(processedData) > 0 {
			log.Printf("Writing %d records from batch %d to sink", len(processedData), batchNumber)
			if err := p.sink.Write(metric.Name, processedData); err != nil {
				return fmt.Errorf("failed to write batch %d to sink: %v", batchNumber, err)
			}
		} else {
			log.Printf("No data found for batch %d", batchNumber)
		}

		// Move to next batch
		currentStart = currentEnd
		batchNumber++
	}

	log.Printf("Completed processing all %d batches for metric %s", batchNumber-1, metric.Name)
	return nil
}

// processBatchResult converts Prometheus response to ProcessedData
func (p *Processor) processBatchResult(
	instanceName, metricName string,
	labelKeys []string,
	result model.Value,
) ([]common.ProcessedData, error) {
	var processed []common.ProcessedData

	// Check result type
	vector, ok := result.(model.Vector)
	if !ok {
		matrix, ok := result.(model.Matrix)
		if !ok {
			return nil, fmt.Errorf("unsupported result type: %T", result)
		}

		// Process matrix result (time series with multiple samples)
		for _, series := range matrix {
			labels := extractLabels(series.Metric, labelKeys)

			for _, sample := range series.Values {
				processed = append(processed, common.ProcessedData{
					PrometheusInstance: instanceName,
					MetricName:         metricName,
					Timestamp:          time.Unix(int64(sample.Timestamp.Unix()), 0),
					Value:              float64(sample.Value),
					Labels:             labels,
				})
			}
		}
		return processed, nil
	}

	// Process vector result (single sample per time series)
	for _, sample := range vector {
		labels := extractLabels(sample.Metric, labelKeys)
		processed = append(processed, common.ProcessedData{
			PrometheusInstance: instanceName,
			MetricName:         metricName,
			Timestamp:          time.Unix(int64(sample.Timestamp.Unix()), 0),
			Value:              float64(sample.Value),
			Labels:             labels,
		})
	}

	return processed, nil
}

// extractLabels extracts specified labels from the metric
func extractLabels(metric model.Metric, keys []string) map[string]string {
	labels := make(map[string]string)
	for _, key := range keys {
		if val, exists := metric[model.LabelName(key)]; exists {
			labels[key] = string(val)
		}
	}
	return labels
}
