package sink

import (
	"fmt"

	"github.com/meiking/tidb-metrics-crawler/pkg/config"
)

// NewSink creates the appropriate sink based on configuration
func NewSink(cfg config.SinkConfig) (Sink, error) {
	switch cfg.Type {
	case "csv":
		return NewCSVSink(cfg.CSV)
	case "feishu":
		return NewFeishuSink(cfg.Feishu)
	default:
		return nil, fmt.Errorf("unsupported sink type: %s", cfg.Type)
	}
}
