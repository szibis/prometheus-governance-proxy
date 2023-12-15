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

// Hash function to create a consistent hash of the metric name
func hash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
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

// HandleMetrics function to handle incoming metrics
func handleMetrics(w http.ResponseWriter, r *http.Request, workItems chan<- WorkItem, endpoints []string, debug bool) {
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
		}
	}
}

func main() {
	// Define the --remoteWrite.urls, --debug, --workers, and --batchSize flags
	var remoteWriteURLs string
	var debug bool
	var workers int
	var batchSize int
	var releaseAfterSeconds int
	flag.StringVar(&remoteWriteURLs, "remoteWrite.urls", "", "Comma-separated list of remote write endpoints")
	flag.BoolVar(&debug, "debug", false, "Print metrics to stdout")
	flag.IntVar(&workers, "workers", 1, "Number of worker goroutines")
	flag.IntVar(&batchSize, "batchSize", 100, "Number of TimeSeries to batch before sending")
	flag.IntVar(&releaseAfterSeconds, "releaseAfterSeconds", 10, "Number of seconds to wait before releasing a batch")
	flag.Parse()

	// Split the flag value into a slice of endpoints
	endpoints := strings.Split(remoteWriteURLs, ",")

	// Create a buffered channel for work items
	workItems := make(chan WorkItem, 100)

	// Start the worker goroutines
	for i := 0; i < workers; i++ {
		go worker(workItems, batchSize, time.Duration(releaseAfterSeconds)*time.Second)
	}

	http.HandleFunc("/write", func(w http.ResponseWriter, r *http.Request) {
		handleMetrics(w, r, workItems, endpoints, debug)
	})
	http.Handle("/metrics", promhttp.Handler())

	fmt.Println("Server is listening on port 8080")
	http.ListenAndServe(":8080", nil)
}
