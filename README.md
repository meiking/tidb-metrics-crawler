## Usage
```
Usage of ./tidb-metrics-crawler-linux-amd64:
  -config string
        Path to configuration file (default "etc/config.yaml")
  -end string
        End time in RFC3339 format (overrides config)
  -prometheus string
        Comma-separated list of Prometheus addresses (overrides config)
  -start string
        Start time in RFC3339 format (overrides config)
  -step string
        Step interval (overrides config)
```

### example
```
./tidb-metrics-crawler-linux-amd64 \
    -config config.prod-ap-southeast-1-a01.yaml \
    -start "2025-09-02T00:00:00Z" \
    -end "2025-09-02T02:00:00Z" \
    -step 1m
```

## config.yaml
```
prometheus_instances:
  - name: primary-prometheus
    address: http://localhost:9090
    timeout: 30s
    # username: admin
    # password: secret

  - name: secondary-prometheus
    address: http://prometheus.example.com:9090
    timeout: 60s

metrics:
  - name: cpu_usage
    query: rate(node_cpu_seconds_total{mode!='idle'}[5m])
    label_keys: ["instance", "mode", "job"]

  - name: memory_usage
    query: node_memory_used_bytes / node_memory_total_bytes * 100
    label_keys: ["instance", "job"]

time_range:
  start: "2023-01-01T00:00:00Z"
  end: "2023-01-02T00:00:00Z"
  step: "5m"

sink:
  type: "csv" # Can be "csv" or "feishu"
  csv:
    output_dir: "./output"
  feishu:
    app_id: "your_app_id"
    app_secret: "your_app_secret"
    receive_id: "user_id_or_chat_id"
    receive_id_type: "user_id" # Can be "user_id", "chat_id", "open_id"
    message_title: "TiDB Metrics Report"
```