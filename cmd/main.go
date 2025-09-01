package main

import (
	"flag"
	"log"
	"strings"
	"time"

	"github.com/meiking/tidb-metrics-crawler/pkg/config"
	"github.com/meiking/tidb-metrics-crawler/pkg/processor"
	"github.com/meiking/tidb-metrics-crawler/pkg/prometheus"
	"github.com/meiking/tidb-metrics-crawler/pkg/sink"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "etc/config.yaml", "Path to configuration file")
	promAddrs := flag.String("prometheus", "", "Comma-separated list of Prometheus addresses (overrides config)")
	st := flag.String("start", "", "Start time in RFC3339 format (overrides config)")
	et := flag.String("end", "", "End time in RFC3339 format (overrides config)")
	step := flag.String("step", "", "Step interval (overrides config)")
	flag.Parse()

	// Load and parse configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	var startTime, endTime time.Time

	// Parse time range
	if *st != "" {
		startTime, err = time.Parse(time.RFC3339, cfg.TimeRange.Start)
		if err != nil {
			log.Fatalf("Invalid start time format: %v", err)
		}
	} else {
		startTime, err = time.Parse(time.RFC3339, cfg.TimeRange.Start)
		if err != nil {
			log.Fatalf("Invalid start time format: %v", err)
		}
	}

	if *et != "" {
		endTime, err = time.Parse(time.RFC3339, *et)
		if err != nil {
			log.Fatalf("Invalid end time format: %v", err)
		}
	} else {
		endTime, err = time.Parse(time.RFC3339, cfg.TimeRange.End)
		if err != nil {
			log.Fatalf("Invalid end time format: %v", err)
		}
	}

	if *step != "" {
		cfg.TimeRange.Step = *step
	}

	if *promAddrs != "" {
		// Override Prometheus addresses from command line
		var instances []config.PrometheusConfig
		for _, addr := range strings.Split(*promAddrs, ",") {
			instances = append(instances, config.PrometheusConfig{
				Name:    addr,
				Address: addr,
				Timeout: "30s",
			})
		}
		cfg.PrometheusInstances = instances
	}

	// Create Prometheus clients
	var clients []prometheus.Client
	for _, instanceCfg := range cfg.PrometheusInstances {
		client, err := prometheus.NewClient(instanceCfg)
		if err != nil {
			log.Printf("Skipping invalid Prometheus instance %s: %v", instanceCfg.Name, err)
			continue
		}
		clients = append(clients, client)
	}

	if len(clients) == 0 {
		log.Fatal("No valid Prometheus instances configured")
	}

	// Create output sink
	outputSink, err := sink.NewSink(cfg.Sink)
	if err != nil {
		log.Fatalf("Failed to create output sink: %v", err)
	}
	defer outputSink.Close()

	// Create and run processor
	dataProcessor := processor.NewProcessor(clients, outputSink)
	if err := dataProcessor.ProcessMetrics(
		cfg.Metrics,
		startTime,
		endTime,
		cfg.TimeRange.Step,
	); err != nil {
		log.Fatalf("Error processing metrics: %v", err)
	}

	log.Println("Metrics processing completed successfully")
}
