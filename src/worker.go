package main

import (
  "fmt"
  "time"
  "bytes"
  "sync"
  "container/list"

  "github.com/golang/snappy"
  "github.com/golang/protobuf/proto"
  "github.com/prometheus/prometheus/prompb"
)

// Define WorkItem outside for your use case
type WorkItem struct {
    ts prompb.TimeSeries
    endpoints []string
    debug bool
}

func worker(workItems <-chan WorkItem, batchSize int, releaseAfter time.Duration, stats *Stats) {

    batches := make(map[uint32][]prompb.TimeSeries)
    timers := make(map[uint32]*time.Timer)

    for item := range workItems {

        var metricNameValue string

        // Find the value of the __name__ label
        for _, label := range item.ts.Labels {
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
        hash := hash(metricNameValue)
        lock.Lock()
        if item.debug {
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
            // Add the logic to drop the tag from the item.ts
            // Replace with your actual logic to drop tag
        }

        // Add the TimeSeries to the appropriate batch
        batches[hash] = append(batches[hash], item.ts)
        stats.ProcessedMetrics++

        // If the batch is full, send it
        if len(batches[hash]) == batchSize {
            // Increase processed bytes based on the data size
            stats.ProcessedBytes += int64(getBytesSize(batches[hash]))

            // Use the hash to select an endpoint
            endpoint := item.endpoints[hash%uint32(len(item.endpoints))]
            if item.debug {
                fmt.Printf("Debug: For hash %d, Endpoint: %s is chosen for sending\n", hash, endpoint)
            }

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

            lock.Unlock() // unlock here before setting AfterFunc

            endpoint := item.endpoints[hash%uint32(len(item.endpoints))]
            if item.debug {
                fmt.Printf("Debug: For hash %d, Endpoint: %s is chosen for timer\n", hash, endpoint)
            }

            // Start a timer to send the batch after the designated duration
            timers[hash] = time.AfterFunc(releaseAfter, func() {
                lock.Lock() // Lock inside the function passed to AfterFunc
                defer lock.Unlock()

                // Increase processed bytes here
                stats.ProcessedBytes += int64(getBytesSize(batches[hash]))

                // Use the hash to select an endpoint
                endpoint := item.endpoints[hash%uint32(len(item.endpoints))]

                // Send the batch to the selected endpoint
                send(endpoint, batches[hash], item.debug)

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
