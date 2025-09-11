package prometheus

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/meiking/tidb-metrics-crawler/pkg/config"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// Client defines the interface for Prometheus clients
type Client interface {
	Name() string
	FetchRange(query string, start, end time.Time, step time.Duration) (model.Value, error)
}

// promClient implements the Client interface
type promClient struct {
	name    string
	api     v1.API
	timeout time.Duration
}

// NewClient creates a new Prometheus client
func NewClient(cfg config.PrometheusConfig) (Client, error) {
	client, err := api.NewClient(api.Config{
		Address:      cfg.Address,
		RoundTripper: newAuthRoundTripper(cfg.Username, cfg.Password, http.DefaultTransport),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Prometheus client: %v", err)
	}

	timeout, err := time.ParseDuration(cfg.Timeout)
	if err != nil || timeout == 0 {
		timeout = 30 * time.Second // Default timeout
	}

	return &promClient{
		name:    cfg.Name,
		api:     v1.NewAPI(client),
		timeout: timeout,
	}, nil
}

// Name returns the client name
func (c *promClient) Name() string {
	return c.name
}

// FetchRange fetches metrics for a time range with retries
func (c *promClient) FetchRange(query string, start, end time.Time, step time.Duration) (model.Value, error) {
	// Maximum retry attempts
	maxRetries := 5
	retryDelay := 2 * time.Second // Initial delay between retries

	for attempt := 1; attempt <= maxRetries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
		defer cancel()

		result, warnings, err := c.api.QueryRange(ctx, query, v1.Range{
			Start: start,
			End:   end,
			Step:  step,
		})

		// Log warnings but don't treat them as errors
		for _, w := range warnings {
			fmt.Printf("Prometheus warning (instance: %s, attempt %d): %v\n", c.name, attempt, w)
		}

		// If successful, return the result
		if err == nil {
			return result, nil
		}

		// Check if we should retry
		if attempt == maxRetries {
			return nil, fmt.Errorf("failed after %d retries: %v", maxRetries, err)
		}

		// Log retry attempt
		fmt.Printf("Retry %d/%d for Prometheus instance %s (error: %v). Waiting %v...\n",
			attempt, maxRetries, c.name, err, retryDelay)

		// Wait before next retry (exponential backoff)
		time.Sleep(retryDelay)
		retryDelay *= 2 // Double the delay for next attempt
	}

	return nil, errors.New("maximum retry attempts exceeded")
}

// authRoundTripper handles basic authentication
type authRoundTripper struct {
	username string
	password string
	rt       http.RoundTripper
}

func newAuthRoundTripper(username, password string, rt http.RoundTripper) http.RoundTripper {
	return &authRoundTripper{
		username: username,
		password: password,
		rt:       rt,
	}
}

func (a *authRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if a.username != "" && a.password != "" {
		req.SetBasicAuth(a.username, a.password)
	}
	return a.rt.RoundTrip(req)
}
