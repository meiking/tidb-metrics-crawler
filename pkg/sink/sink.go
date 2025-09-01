package sink

import "github.com/meiking/tidb-metrics-crawler/pkg/common"

// Sink defines the interface for output destinations
type Sink interface {
	// Write sends processed data to the output destination
	Write(metricName string, data []common.ProcessedData) error

	// Close cleans up any resources used by the sink
	Close() error
}
