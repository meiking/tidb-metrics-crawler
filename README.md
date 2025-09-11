# tidb-metrics-crawler

A tool to crawl metrics data from Prometheus instances and export to various destinations (CSV, MySQL, Feishu).

## Features

- **Multi-Prometheus Support**: Connect to multiple Prometheus instances simultaneously
- **Batch Fetching**: Automatically split time ranges into hourly batches to avoid timeout
- **Retry Mechanism**: 5 retries with exponential backoff for failed requests
- **Multiple Output Destinations**:
  - CSV files
  - MySQL database
  - Feishu (Lark) messages with attachments
- **Flexible Configuration**: YAML config file with command-line overrides
- **Time Range Control**: Specify start/end time and step interval for metrics collection

## Installation

### Prerequisites

- Go 1.20+

### From Source

```bash
# Clone the repository
git clone https://github.com/meiking/tidb-metrics-crawler.git
cd tidb-metrics-crawler

# Build the binary
make build

# The binary will be in ./bin/tidb-metrics-crawler
```

### Using `go install`

```bash
go install github.com/meiking/tidb-metrics-crawler/cmd@latest
```

## Usage

### Basic Workflow

1. Create a configuration file (see example below)
2. Run the crawler with the configuration file
3. Check output in your specified destination

### Example Command

```bash
# Using config file only
./bin/tidb-metrics-crawler -config etc/config.yaml

# Overriding Prometheus instances and time range
./bin/tidb-metrics-crawler -config etc/config.yaml \
  -prometheus "http://prom1:9090,http://prom2:9090" \
  -start "2023-10-01T00:00:00Z" \
  -end "2023-10-02T00:00:00Z"
```

## Configuration

### Example YAML Config (`etc/config.yaml.example`)

```yaml
# Prometheus instances (can be overridden by -prometheus flag)
prometheusInstances:
  - name: "primary-prom"
    address: "http://localhost:9090"
    timeout: 30
    # username: "admin"  # Optional
    # password: "secret" # Optional

# Metrics to collect
metrics:
  - name: "cpu_usage"
    query: "rate(node_cpu_seconds_total{mode!='idle'}[5m])"
    labelKeys: ["instance", "mode", "job"]
  - name: "memory_usage"
    query: "node_memory_used_bytes / node_memory_total_bytes * 100"
    labelKeys: ["instance", "job"]

# Time range (can be overridden by -start, -end, -step flags)
startTime: "2023-10-01T00:00:00Z"
endTime: "2023-10-02T00:00:00Z"
step: "5m"

# Output sink configuration
sink:
  # Can be "csv", "mysql", or "feishu"
  type: "csv"
  
  # CSV sink configuration
  csv:
    outputDir: "./output"
  
  # MySQL sink configuration (when type is "mysql")
  # mysql:
  #   dsn: "user:password@tcp(localhost:3306)/metrics_db"
  #   table: "prometheus_metrics"
  #   batchSize: 1000
  #   createTable: true
  #   truncateTable: false
  
  # Feishu sink configuration (when type is "feishu")
  # feishu:
  #   appID: "your-app-id"
  #   appSecret: "your-app-secret"
  #   receiveID: "user-id-or-chat-id"
  #   receiveIDType: "user_id"
  #   messageTitle: "Metrics Report"
```

## Command-line Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-config` | Path to configuration file | `etc/config.yaml` |
| `-prometheus` | Comma-separated Prometheus addresses (overrides config) | Empty |
| `-start` | Start time in RFC3339 format (overrides config) | Empty |
| `-end` | End time in RFC3339 format (overrides config) | Empty |
| `-step` | Step interval (overrides config) | Empty |

## Project Structure

```
tidb-metrics-crawler/
├── cmd/
│   └── main.go               # Application entry point
├── etc/
│   └── config.yaml.example   # Example configuration
├── pkg/
│   ├── config/               # Configuration parsing
│   ├── common/               # Shared data structures
│   ├── prometheus/           # Prometheus client
│   ├── processor/            # Data processing logic
│   └── sink/                 # Output sinks
│       ├── csv_sink.go       # CSV output
│       ├── mysql_sink.go     # MySQL output
│       └── feishu_sink.go    # Feishu output
├── Makefile                  # Build automation
└── go.mod                    # Go module definition
```

## MySQL Table Schema

When using the MySQL sink with `createTable: true`, the following table will be created:

```sql
CREATE TABLE IF NOT EXISTS prometheus_metrics (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  prometheus_instance VARCHAR(255) NOT NULL,
  metric_name VARCHAR(255) NOT NULL,
  timestamp DATETIME NOT NULL,
  value DOUBLE NOT NULL,
  labels JSON,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_instance (prometheus_instance),
  INDEX idx_metric (metric_name),
  INDEX idx_timestamp (timestamp),
  INDEX idx_instance_metric (prometheus_instance, metric_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.