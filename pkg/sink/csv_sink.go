package sink

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/meiking/tidb-metrics-crawler/pkg/common"
	"github.com/meiking/tidb-metrics-crawler/pkg/config"
)

// CSVSink writes processed data to CSV files
type CSVSink struct {
	outputDir string
	files     map[string]*os.File
	writers   map[string]*csv.Writer
}

// NewCSVSink creates a new CSV sink
func NewCSVSink(cfg config.CSVConfig) (*CSVSink, error) {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %v", err)
	}

	return &CSVSink{
		outputDir: cfg.OutputDir,
		files:     make(map[string]*os.File),
		writers:   make(map[string]*csv.Writer),
	}, nil
}

// Write writes processed data to a CSV file
func (s *CSVSink) Write(metricName string, data []common.ProcessedData) error {
	if len(data) == 0 {
		return nil // Nothing to write
	}

	// Create writer if it doesn't exist
	if _, exists := s.writers[metricName]; !exists {
		if err := s.createWriter(metricName, data[0]); err != nil {
			return err
		}
	}

	// Write data rows
	writer := s.writers[metricName]
	for _, item := range data {
		row, err := s.createDataRow(item)
		if err != nil {
			return err
		}

		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %v", err)
		}
	}

	// Flush after writing batch
	writer.Flush()

	return writer.Error()
}

// Close cleans up resources
func (s *CSVSink) Close() error {
	var lastErr error

	// Close all files
	for name, file := range s.files {
		if err := file.Close(); err != nil {
			lastErr = fmt.Errorf("error closing file %s: %v", name, err)
		}
		delete(s.files, name)
		delete(s.writers, name)
	}

	return lastErr
}

// createWriter initializes a new CSV writer for a metric
func (s *CSVSink) createWriter(metricName string, sampleData common.ProcessedData) error {
	// Create filename with timestamp
	timestamp := time.Now().Format("20060102150405")
	filename := fmt.Sprintf("%s_%s.csv", metricName, timestamp)
	path := filepath.Join(s.outputDir, filename)

	// Create file
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %v", err)
	}

	// Create writer and write header
	writer := csv.NewWriter(file)
	header, err := s.createHeaderRow(sampleData)
	if err != nil {
		file.Close()
		return err
	}

	if err := writer.Write(header); err != nil {
		file.Close()
		return fmt.Errorf("failed to write CSV header: %v", err)
	}

	// Store writer and file
	s.files[metricName] = file
	s.writers[metricName] = writer

	return nil
}

// createHeaderRow creates the CSV header row
func (s *CSVSink) createHeaderRow(sample common.ProcessedData) ([]string, error) {
	// Base columns
	header := []string{
		"prometheus_instance",
		"metric_name",
		"timestamp",
		"value",
	}

	// Add label columns
	for key := range sample.Labels {
		header = append(header, fmt.Sprintf("label_%s", key))
	}

	return header, nil
}

// createDataRow creates a CSV row from processed data
func (s *CSVSink) createDataRow(data common.ProcessedData) ([]string, error) {
	// Base data
	row := []string{
		data.PrometheusInstance,
		data.MetricName,
		data.Timestamp.Format(time.RFC3339),
		fmt.Sprintf("%f", data.Value),
	}

	// Add label values in consistent order
	for _, key := range getSortedLabelKeys(data.Labels) {
		row = append(row, data.Labels[key])
	}

	return row, nil
}

// getSortedLabelKeys returns sorted label keys for consistent CSV output
func getSortedLabelKeys(labels map[string]string) []string {
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
