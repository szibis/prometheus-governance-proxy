package stats

import (
	"encoding/json"
	"log"
	"sync"
	"time"
  
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

func (s *Stats) resetCounts() {
	s.DroppedMetrics = 0
	s.DroppedTags = 0
}

func getBytesSize(batch []prompb.TimeSeries) int {
	req := &prompb.WriteRequest{
		Timeseries: batch,
	}

	data, err := proto.Marshal(req)
	if err != nil {
		return 0
	}

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

func LogStats(stats *Stats, period time.Duration) {
	ticker := time.NewTicker(period)
	defer ticker.Stop()

	for range ticker.C {
		logJSON(stats)

		stats.ProcessedMetrics = 0
		stats.ProcessedBytes = 0
		stats.DroppedMetrics = 0
		stats.DroppedTags = 0
	}
}
