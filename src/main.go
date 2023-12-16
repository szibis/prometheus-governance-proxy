package main

import (
  "encoding/json"
  "fmt"
  "sync"
  "net/http"
  "strings"
  "flag"
  "time"

  "github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
  // Define the --config-file flag
  var configFile string
  flag.StringVar(&configFile, "config-file", "config.yml", "The name of the YAML configuration file")
  flag.Parse()

  // Initialize the config
  config := &Config{}
  config.readConfig(configFile)

  // Split the config value into a slice of endpoints
  endpoints := strings.Split(config.RemoteWriteURLs, ",")

  // Create a buffered channel for work items
  workItems := make(chan WorkItem, 100)

  // Start the worker goroutines
  for i := 0; i < config.Workers; i++ {
    go worker(workItems, config.BatchSize, time.Duration(config.ReleaseAfterSeconds)*time.Second)
  }

  // Handle incoming metrics and post work items for workers
  http.HandleFunc("/write", func(w http.ResponseWriter, r *http.Request) {
    handleMetrics(w, r, workItems, endpoints, config.Debug, config.CardinalityLimit.Capacity, config.CardinalityLimit.Limit, config.CardinalityLimit.Mode)
  })

  // Expose metrics cardinality as JSON
  http.HandleFunc("/metrics_cardinality", func(w http.ResponseWriter, r *http.Request) {
      data := make(map[string]map[string]map[string]int)

      metricsData.Metrics.Range(func(key, value interface{}) bool {
          metric := key.(string)

          labels := value.(*sync.Map)
          labels.Range(func(labelKey, labelValue interface{}) bool {
              label := labelKey.(string)
              tagData := labelValue.(*TagData)
              if tagData.Cardinality > config.JsonMinCardinality {
                  if _, exists := data[metric]; !exists {
                      data[metric] = make(map[string]map[string]int)
                      data[metric]["tags"] = make(map[string]int)
                  }
                  data[metric]["tags"][label] = tagData.Cardinality
              }
              return true
          })
          
          return true
      })

      w.Header().Set("Content-Type", "application/json")
      json.NewEncoder(w).Encode(map[string]interface{}{"cardinality": data})
  })

  // Expose Prometheus metrics
  http.Handle("/metrics", promhttp.Handler())

  fmt.Println("Server is listening on port 8080")
  http.ListenAndServe(":8080", nil)
}
