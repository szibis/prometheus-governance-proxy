package worker

import (
	"fmt"
	"time"
	"sync"
	"encoding/json"
	"log"

	"github.com/prometheus/prometheus/prompb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/proto"

	"github.com/szibis/prometheus-governance-proxy/config"
	"github.com/szibis/prometheus-governance-proxy/stats"
	"github.com/szibis/prometheus-governance-proxy/metrics"
  "github.com/szibis/prometheus-governance-proxy/send"
)

type WorkItem struct {
    Ts         prompb.TimeSeries
    Endpoints  []string
    Debug      bool
}

var lock = sync.RWMutex{}

// Hash function to create a consistent hash of the metric name
func hash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

func getBytesSize(batch []prompb.TimeSeries) int {
	// The WriteRequest is what gets sent over the wire
	req := &prompb.WriteRequest{
		Timeseries: batch,
	}

	// We marshal the WriteRequest to a byte slice
	data, err := proto.Marshal(req)
	if err != nil {
		return 0
	}

	// Return length of byte slice
	return len(data)
}

func logJSON(i interface{}) {
	bytes, err := json.Marshal(i)
	if err != nil {
		log.Printf("Could not encode info: %v", err)
		return
	}
	log.Println(string(bytes))
}

func LogStats(stats *stats.Stats, period time.Duration) {
	ticker := time.NewTicker(period)
	defer ticker.Stop()

	for range ticker.C {
		logJSON(stats)

		// Reset the metrics after logging.
		stats.ProcessedMetrics = 0
		stats.ProcessedBytes = 0
		stats.DroppedMetrics = 0
		stats.DroppedTags = 0
	}
}

// Worker function
func Worker(workItems <-chan WorkItem, batchSize int, releaseAfter time.Duration, stats *stats.Stats) {

    batches := make(map[uint32][]prompb.TimeSeries)
    timers := make(map[uint32]*time.Timer)

    for item := range workItems {

        var metricNameValue string

        // Find the value of the __name__ label
        for _, label := range item.Ts.Labels {
            if label.Name == "__name__" {
                metricNameValue = label.Value
                break
            }
        }

        if metricNameValue == "" {
            fmt.Printf("The TimeSeries doesn't have a __name__ label. Skipping...\n")
            continue
        }

        // Create a consistent hash of the metric name value
        hash := Hash(metricNameValue)
        lock.Lock()
        if item.Debug {
            fmt.Printf("Debug: Primary Hash(%s) = %d\n", metricNameValue, hash)
        }

        // Hypothetical conditions to check if you need to drop metric or a tag in it
        isMetricToBeDropped := false // Replace with your actual logic
        isTagToBeDropped := false    // Replace with your actual logic

        if isMetricToBeDropped {
            stats.DroppedMetrics++
            lock.Unlock()
            continue        // If we are dropping the metric, we don't process it further
        }

        // If only specific tag in the metric requires dropping
        if isTagToBeDropped {
            stats.DroppedTags++
            // Add the logic to drop the tag from the item.Ts
            // Replace with your actual logic to drop tag
        }

        // Add the TimeSeries to the appropriate batch
        batches[hash] = append(batches[hash], item.Ts)
        stats.ProcessedMetrics++

        // If the batch is full, send it
        if len(batches[hash]) == batchSize {
            // Increase processed bytes based on the data size
            stats.ProcessedBytes += int64(GetBytesSize(batches[hash]))

            // Use the hash to select an endpoint
            endpoint := item.Endpoints[hash%uint32(len(item.Endpoints))]
            if item.Debug {
                fmt.Printf("Debug: For hash %d, Endpoint: %s is chosen for sending\n", hash, endpoint)
            }

            // Send the batch to the selected endpoint
            Send(endpoint, batches[hash], item.Debug)

            // Clear the batch
            batches[hash] = nil

            // Stop the timer
            if timers[hash] != nil {
                timers[hash].Stop()
                timers[hash] = nil
            }
        } else if timers[hash] == nil {

            lock.Unlock() // unlock here before setting AfterFunc

            endpoint := item.Endpoints[hash%uint32(len(item.Endpoints))]
            if item.Debug {
                fmt.Printf("Debug: For hash %d, Endpoint: %s is chosen for timer\n", hash, endpoint)
            }

            // Start a timer to send the batch after the designated duration
            timers[hash] = time.AfterFunc(releaseAfter, func() {
                lock.Lock() // Lock inside the function passed to AfterFunc
                defer lock.Unlock()

                // Increase processed bytes here
                stats.ProcessedBytes += int64(GetBytesSize(batches[hash]))

                // Use the hash to select an endpoint
                endpoint := item.Endpoints[hash%uint32(len(item.Endpoints))]

                // Send the batch to the selected endpoint
                Send(endpoint, batches[hash], item.Debug)

                // Clear the batch
                batches[hash] = nil

                // Clear the timer
                timers[hash] = nil
            })

            continue
        }
        lock.Unlock()
    }
}
