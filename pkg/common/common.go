package common

import "time"

// ProcessedData represents a single piece of processed metric data
// This is moved to a common package to avoid circular dependencies
type ProcessedData struct {
	PrometheusInstance string            `json:"prometheusInstance"`
	MetricName         string            `json:"metricName"`
	Timestamp          time.Time         `json:"timestamp"`
	Value              float64           `json:"value"`
	Labels             map[string]string `json:"labels"`
}
