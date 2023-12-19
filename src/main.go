package main

import (
	"flag"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/szibis/prometheus-governance-proxy/config"
	"github.com/szibis/prometheus-governance-proxy/worker"
	"github.com/szibis/prometheus-governance-proxy/handle"
	"github.com/szibis/prometheus-governance-proxy/stats"
  "github.com/szibis/prometheus-governance-proxy/api"
)

var lock = sync.RWMutex{}

func main() {
	var configFile string
	flag.StringVar(&configFile, "config-file", "config.yml", "The name of the YAML configuration file")
	flag.Parse()

	// Initialize the config
	conf := &config.Config{}
	conf.ReadConfig(configFile)

	endpoints := strings.Split(conf.RemoteWriteURLs, ",")

	workItems := make(chan worker.WorkItem, 100)

	// Initialize the stats
	stat := &stats.Stats{}

	// Start the worker goroutines
	for i := 0; i < conf.Workers; i++ {
		go worker.Worker(workItems, conf.BatchSize, time.Duration(conf.ReleaseAfterSeconds)*time.Second, stat)
	}

	// Log the stats periodically.
  go worker.LogStats(stat, time.Duration(conf.StatsIntervalSeconds)*time.Second)

  http.HandleFunc("/write", func(w http.ResponseWriter, r *http.Request) {
      handle.HandleMetrics(w, r, workItems, endpoints, conf, stat)
  })

  http.HandleFunc("/metrics_cardinality", func(w http.ResponseWriter, r *http.Request) {
      api.HandleMetricsCardinality(w, conf)
  })

	// Expose Prometheus metrics
	http.Handle("/metrics", promhttp.Handler())

	fmt.Println("Server is listening on port 8080")
	http.ListenAndServe(":8080", nil)
} 
