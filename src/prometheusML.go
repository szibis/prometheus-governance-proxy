package main

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"net/http"
	"io/ioutil"
	"strings"
	"flag"
	"time"
  "bytes"
  "sync"
  "container/list"
  "os"

	"github.com/golang/snappy"
	"github.com/golang/protobuf/proto"
	"github.com/prometheus/prometheus/prompb"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type WorkItem struct {
	ts        prompb.TimeSeries
	endpoints []string
	debug     bool
}

type MetricTag struct {
	Metric string
	Tag    string
}

type ValueChange struct {
	Previous string
	Current  string
}

type MetricsData struct {
	sync.RWMutex
	Metrics map[string]map[string]*TagData
}

type TagData struct {
	Capacity    int
	Cardinality int
	Values      *list.List
}

var metricsData = MetricsData{
	Metrics: make(map[string]map[string]*TagData),
}

// Hash function to create a consistent hash of the metric name
func hash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

func handleCardinalityLimitation(metricName string, labelName string, cardinalityLimit int, cardinalityLimitMode string) {
	metricsData.Lock()
	defer metricsData.Unlock()

	if metricLabels, ok := metricsData.Metrics[metricName]; ok {
		if tagData, ok := metricLabels[labelName]; ok { 
			switch cardinalityLimitMode {
			case "drop_metric":
				if tagData.Cardinality > cardinalityLimit {
					fmt.Println("Dropping metric due to cardinality limit for metric:", metricName, "and tag:", labelName)
					delete(metricsData.Metrics, metricName)
				}

			case "drop_tag":
				if tagData.Cardinality > cardinalityLimit {
					fmt.Println("Dropping tag due to cardinality limit for metric:", metricName, "and tag:", labelName)
					delete(metricsData.Metrics[metricName], labelName)
				}
					
			case "last_100":
				if len(metricsData.Metrics[metricName]) > 100 {
					cutOffTag := getNthKey(metricsData.Metrics[metricName], 100)
					fmt.Println("Limiting tags to last 100 for metric:", metricName, "and tag:", cutOffTag)
					delete(metricsData.Metrics[metricName], cutOffTag)
				}
					
			default:
				fmt.Println("Error: Invalid cardinalityLimitMode:", cardinalityLimitMode)
				os.Exit(1)
			}

		}
	}
}

// Given a map and an index n, it returns the name of the nth key.
func getNthKey(m map[string]*TagData, n int) string {
	i := 0
	for key := range m {
		if i == n {
			return key
		}
		i++
	}
	return ""
}

// Send function to send the given metrics to the given endpoint
func send(endpoint string, ts []prompb.TimeSeries, debug bool) {
	// Create a WriteRequest
	req := &prompb.WriteRequest{
		Timeseries: ts,
	}

	// Marshal the WriteRequest to a byte slice
	data, err := proto.Marshal(req)
	if err != nil {
		fmt.Println("Error marshalling the WriteRequest:", err)
		return
	}

	// Compress the data using snappy
	compressed := snappy.Encode(nil, data)

	// Send the data to the endpoint
	resp, err := http.Post(endpoint, "application/x-protobuf", bytes.NewReader(compressed))
	if err != nil {
		fmt.Println("Error sending the request:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Println("Received non-204 response:", resp.StatusCode, "Response body:", string(body))
	}

	// If debug is enabled, print the metric to stdout
	if debug {
		for _, t := range ts {
			labels := make(map[string]string)
			for _, label := range t.Labels {
				labels[label.Name] = label.Value
			}
			for _, sample := range t.Samples {
				debugInfo := map[string]interface{}{
					"name":      labels["__name__"],
					"labels":    labels,
					"value":     sample.Value,
					"timestamp": sample.Timestamp,
				}
				jsonDebugInfo, _ := json.Marshal(debugInfo)
				fmt.Println(string(jsonDebugInfo))
			}
		}
	}
}

// Worker function to process work items
func worker(workItems <-chan WorkItem, batchSize int, releaseAfter time.Duration) {
	batches := make(map[uint32][]prompb.TimeSeries)
	timers := make(map[uint32]*time.Timer)

	for item := range workItems {
		// Get the metric name
		metricName := item.ts.Labels[0].Name

		// Create a consistent hash of the metric name
		hash := hash(metricName)

		// Add the TimeSeries to the appropriate batch
		batches[hash] = append(batches[hash], item.ts)

		// If the batch is full, send it
		if len(batches[hash]) == batchSize {
			// Use the hash to select an endpoint
			endpoint := item.endpoints[hash%uint32(len(item.endpoints))]

			// Send the batch to the selected endpoint
			send(endpoint, batches[hash], item.debug)

			// Clear the batch
			batches[hash] = nil

			// Stop the timer
			if timers[hash] != nil {
				timers[hash].Stop()
				timers[hash] = nil
			}
		} else if timers[hash] == nil {
			// Start a timer to send the batch after the specified duration
			timers[hash] = time.AfterFunc(releaseAfter, func() {
				// Use the hash to select an endpoint
				endpoint := item.endpoints[hash%uint32(len(item.endpoints))]

				// Send the batch to the selected endpoint
				send(endpoint, batches[hash], item.debug)

				// Clear the batch
				batches[hash] = nil

				// Clear the timer
				timers[hash] = nil
			})
		}
	}
}

func handleMetrics(w http.ResponseWriter, r *http.Request, workItems chan<- WorkItem, endpoints []string, debug bool, cardinalityCapacity int, cardinalityLimit int, cardinalityLimitMode string) {
	if r.Method == http.MethodPost {
		compressed, _ := ioutil.ReadAll(r.Body)

		// Decompress the data
		data, err := snappy.Decode(nil, compressed)
		if err != nil {
			fmt.Println("Error decompressing the data:", err)
			return
		}

		// Unmarshal the data into a WriteRequest
		var req prompb.WriteRequest
		if err := proto.Unmarshal(data, &req); err != nil {
			fmt.Println("Error unmarshalling the WriteRequest:", err)
			return
		}

		// Process each TimeSeries in the WriteRequest
		for _, ts := range req.Timeseries {
			workItems <- WorkItem{ts: ts, endpoints: endpoints, debug: debug}
			
			metricName := ""
			for _, label := range ts.Labels {
				if label.Name == "__name__" {
					metricName = label.Value
					break
				}
			}

			// If metric name is not found, continue to next TimeSeries
			if metricName == "" {
				continue
			}

			metricsData.Lock()
		
			for _, label := range ts.Labels {
				if label.Name == "__name__" {
					continue // Omit __name__ tag
				}
				if _, ok := metricsData.Metrics[metricName]; !ok {
					metricsData.Metrics[metricName] = make(map[string]*TagData)
				}
				if _, ok := metricsData.Metrics[metricName][label.Name]; !ok {
					metricsData.Metrics[metricName][label.Name] = &TagData{
						Capacity:    cardinalityCapacity,
						Cardinality: 0,
						Values:      list.New(),
					}
				}

				tagData := metricsData.Metrics[metricName][label.Name]

				// Check if this tag value is already stored
				found := false
				for e := tagData.Values.Front(); e != nil; e = e.Next() {
					if e.Value.(string) == label.Value {
						found = true
						break
					}
				}
				if !found {
					// Increment cardinality
					tagData.Cardinality++

					// Add new tag value to the stored values
					tagData.Values.PushBack(label.Value)

					// Spawn a goroutine to handle cardinality limitation
					go handleCardinalityLimitation(metricName, label.Name, cardinalityLimit, cardinalityLimitMode)

					// Check if we need to remove the oldest value
					if tagData.Values.Len() > tagData.Capacity {
						tagData.Values.Remove(tagData.Values.Front())
					}
				}
			}
			
			metricsData.Unlock()
		}
	}
}

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
		metricsData.RLock()
    		defer metricsData.RUnlock()  // make sure the lock is always released

    		data := make(map[string]map[string]map[string]int)

    		for metric, labels := range metricsData.Metrics {
      			for label, tagData := range labels {
        			if tagData.Cardinality > config.JsonMinCardinality {
          				if _, exists := data[metric]; !exists {
            				data[metric] = make(map[string]map[string]int)
            				data[metric]["tags"] = make(map[string]int)
          				}
          				data[metric]["tags"][label] = tagData.Cardinality
        			}
      			}
    		}

    		w.Header().Set("Content-Type", "application/json")
    		json.NewEncoder(w).Encode(map[string]interface{}{"cardinality": data})
	})

	// Expose Prometheus metrics
	http.Handle("/metrics", promhttp.Handler())

	fmt.Println("Server is listening on port 8080")
	http.ListenAndServe(":8080", nil)
}
