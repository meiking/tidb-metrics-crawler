package prometheus

import (
	"context"
	"fmt"
	"time"

	"github.com/meiking/tidb-metrics-crawler/pkg/config"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// Client represents a Prometheus client
type Client interface {
	Name() string
	FetchRange(query string, start, end time.Time, step time.Duration) (model.Value, error)
}

// prometheusClient implements the Client interface
type prometheusClient struct {
	name    string
	api     v1.API
	timeout time.Duration
}

// NewClient creates a new Prometheus client
func NewClient(cfg config.PrometheusConfig) (Client, error) {
	// Parse timeout duration
	timeout, err := time.ParseDuration(cfg.Timeout)
	if err != nil {
		return nil, fmt.Errorf("invalid timeout: %v", err)
	}

	// Create Prometheus API client
	client, err := api.NewClient(api.Config{
		Address:      cfg.Address,
		RoundTripper: createAuthTransport(cfg.Username, cfg.Password),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}

	return &prometheusClient{
		name:    cfg.Name,
		api:     v1.NewAPI(client),
		timeout: timeout,
	}, nil
}

// Name returns the name of the Prometheus instance
func (c *prometheusClient) Name() string {
	return c.name
}

// FetchRange executes a range query against Prometheus
func (c *prometheusClient) FetchRange(query string, start, end time.Time, step time.Duration) (model.Value, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	result, warnings, err := c.api.QueryRange(ctx, query, v1.Range{
		Start: start,
		End:   end,
		Step:  step,
	})

	if len(warnings) > 0 {
		fmt.Printf("Warnings: %v\n", warnings)
	}

	if err != nil {
		return nil, fmt.Errorf("query failed: %v", err)
	}

	return result, nil
}
