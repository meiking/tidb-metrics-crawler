package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the main application configuration
type Config struct {
	PrometheusInstances []PrometheusConfig `yaml:"prometheus_instances"`
	Metrics             []MetricConfig     `yaml:"metrics"`
	TimeRange           TimeRangeConfig    `yaml:"time_range"`
	Sink                SinkConfig         `yaml:"sink"`
}

// PrometheusConfig contains configuration for a Prometheus instance
type PrometheusConfig struct {
	Name     string `yaml:"name"`
	Address  string `yaml:"address"`
	Timeout  string `yaml:"timeout"`
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
}

// MetricConfig contains configuration for a specific metric to fetch
type MetricConfig struct {
	Name      string   `yaml:"name"`
	Query     string   `yaml:"query"`
	LabelKeys []string `yaml:"label_keys"`
}

// TimeRangeConfig contains time range configuration
type TimeRangeConfig struct {
	Start string `yaml:"start"`
	End   string `yaml:"end"`
	Step  string `yaml:"step"`
}

// SinkConfig contains configuration for output sinks
type SinkConfig struct {
	Type   string       `yaml:"type"`
	CSV    CSVConfig    `yaml:"csv,omitempty"`
	Feishu FeishuConfig `yaml:"feishu,omitempty"`
}

// CSVConfig contains configuration for CSV sink
type CSVConfig struct {
	OutputDir string `yaml:"output_dir"`
}

// FeishuConfig contains configuration for Feishu sink
type FeishuConfig struct {
	AppID         string `yaml:"app_id"`
	AppSecret     string `yaml:"app_secret"`
	ReceiveID     string `yaml:"receive_id"`
	ReceiveIDType string `yaml:"receive_id_type"`
	MessageTitle  string `yaml:"message_title"`
}

// Load reads and parses a YAML configuration file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
