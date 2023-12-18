package main

import (
	"encoding/json"
	"log"
	"time"
  "sync"
	"github.com/golang/protobuf/proto"
	"github.com/prometheus/prometheus/prompb"
)

type Stats struct {
	ProcessedMetrics int64 `json:"processed_metrics"`
	DroppedMetrics   int64 `json:"dropped_metrics"`
	DroppedTags      int64 `json:"dropped_tags"`
	ProcessedBytes   int64 `json:"processed_bytes"`
  sync.Mutex
}

// Add a method to reset counts
func (s *Stats) resetCounts() {
    s.DroppedMetrics = 0
    s.DroppedTags = 0
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

func logStats(stats *Stats, period time.Duration) {
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
