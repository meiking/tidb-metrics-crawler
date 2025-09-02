package sink

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/meiking/tidb-metrics-crawler/pkg/common"
	"github.com/meiking/tidb-metrics-crawler/pkg/config"
)

// MySQLSink stores processed data directly in MySQL database
type MySQLSink struct {
	db        *sql.DB
	cfg       config.MySQLConfig
	tableName string
	batchSize int
	batchData [][]interface{} // Buffer for batch inserts
}

// NewMySQLSink creates a new MySQL sink
func NewMySQLSink(cfg config.MySQLConfig) (*MySQLSink, error) {
	log.Printf("Initializing MySQL sink with DSN: %s, createTable: %v, truncateTable: %v", cfg.DSN, cfg.CreateTable, cfg.TruncateTable)

	// Set defaults
	tableName := cfg.Table
	if tableName == "" {
		tableName = "prometheus_metrics"
	}

	batchSize := cfg.BatchSize
	if batchSize <= 0 {
		batchSize = 1000
	}

	// Validate configuration
	if cfg.DSN == "" {
		return nil, fmt.Errorf("MySQL DSN is required")
	}

	// Connect to MySQL
	db, err := sql.Open("mysql", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL: %v", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping MySQL: %v", err)
	}

	// Create table if needed
	if cfg.CreateTable {
		log.Printf("Creating table %s if it does not exist", tableName)
		if err := createMetricsTable(db, tableName); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to create table: %v", err)
		}
	}

	// Truncate table if requested
	if cfg.TruncateTable {
		_, err := db.Exec(fmt.Sprintf("TRUNCATE TABLE %s", tableName))
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to truncate table: %v", err)
		}
	}

	return &MySQLSink{
		db:        db,
		cfg:       cfg,
		tableName: tableName,
		batchSize: batchSize,
		batchData: make([][]interface{}, 0, batchSize),
	}, nil
}

// Write processes data and saves to MySQL (using batch inserts)
func (s *MySQLSink) Write(metricName string, data []common.ProcessedData) error {
	if len(data) == 0 {
		return nil
	}

	// Convert processed data to database records
	for _, item := range data {
		// Flush batch when it reaches the configured size
		if len(s.batchData) >= s.batchSize {
			if err := s.flushBatch(); err != nil {
				return err
			}
		}

		// Convert labels map to JSON string
		labelsJSON, err := common.MapToJSONString(item.Labels)
		if err != nil {
			return fmt.Errorf("failed to convert labels to JSON: %v", err)
		}

		// Add to batch
		s.batchData = append(s.batchData, []interface{}{
			item.PrometheusInstance,
			metricName,
			item.Timestamp,
			item.Value,
			labelsJSON,
		})
	}

	return nil
}

// Close cleans up resources and flushes remaining batch data
func (s *MySQLSink) Close() error {
	// Flush any remaining data in batch
	if len(s.batchData) > 0 {
		if err := s.flushBatch(); err != nil {
			return fmt.Errorf("failed to flush final batch: %v", err)
		}
	}

	// Close database connection
	return s.db.Close()
}

// flushBatch inserts the current batch of data into MySQL
func (s *MySQLSink) flushBatch() error {
	log.Printf("Inserting batch of %d records into MySQL", len(s.batchData))
	if len(s.batchData) == 0 {
		return nil
	}

	// Create placeholders for batch insert
	placeholders := make([]string, len(s.batchData))
	for i := range placeholders {
		placeholders[i] = "(?, ?, ?, ?, ?)"
	}

	// Build query
	query := fmt.Sprintf(
		"INSERT INTO %s (prometheus_instance, metric_name, timestamp, value, labels) VALUES %s",
		s.tableName,
		strings.Join(placeholders, ","),
	)

	// Flatten the batch data for the query
	args := make([]interface{}, 0, len(s.batchData)*5)
	for _, row := range s.batchData {
		args = append(args, row...)
	}

	// Execute the insert
	_, err := s.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("insert failed: %v", err)
	}

	// Clear the batch
	s.batchData = s.batchData[:0]

	return nil
}

// createMetricsTable creates the metrics table if it doesn't exist
func createMetricsTable(db *sql.DB, tableName string) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			prometheus_instance VARCHAR(255) NOT NULL,
			metric_name VARCHAR(255) NOT NULL,
			timestamp DATETIME NOT NULL,
			value DOUBLE NOT NULL,
			labels JSON,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_instance_metric (prometheus_instance, metric_name),
			INDEX idx_timestamp (timestamp)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`, tableName)

	_, err := db.Exec(query)
	return err
}
