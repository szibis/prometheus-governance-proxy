package handle

import (
	"container/list"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

  "github.com/golang/protobuf/proto"
  "github.com/golang/snappy"
  "github.com/prometheus/prometheus/prompb"
	"github.com/szibis/prometheus-governance-proxy/stats"
	"github.com/szibis/prometheus-governance-proxy/worker"
  "github.com/szibis/prometheus-governance-proxy/config"
  "github.com/szibis/prometheus-governance-proxy/metrics"
  "github.com/szibis/prometheus-governance-proxy/plugins"
)

type TagData struct {
       Cardinality int
       Values      *list.List
       Capacity    int
}

func HandleMetrics(w http.ResponseWriter, r *http.Request, workItems chan<- worker.WorkItem, endpoints []string, config *config.Config, stats *stats.Stats) {
	if r.Method != http.MethodPost {
		return
	}

  var cardinalityPlugin *plugins.Cardinality
  if config.CardinalityLimit.Enable {
      cardinalityPlugin = plugins.NewCardinality(config)
  }

	compressed, _ := ioutil.ReadAll(r.Body)
	data, err := snappy.Decode(nil, compressed)
	if err != nil {
		fmt.Println("Error decompressing data:", err)
		return
	}

	var req prompb.WriteRequest
	if err := proto.Unmarshal(data, &req); err != nil {
		fmt.Println("Error unmarshalling the WriteRequest:", err)
		return
	}

	metricsCardinality := make(map[string]map[string]int)

	for _, ts := range req.Timeseries {
		workItems <- worker.WorkItem{Ts: ts, Endpoints: endpoints, Debug: config.Debug}

		metricName := ""
		for _, label := range ts.Labels {
			if label.Name == "__name__" {
				metricName = label.Value
				break
			}
		}

		if metricName == "" {
			continue
		}

		metricLabelsInterface, _ := metrics.MetricsData.Metrics.LoadOrStore(metricName, &sync.Map{})
		metricLabels := metricLabelsInterface.(*sync.Map)

		metricsCardinality[metricName] = make(map[string]int)

		for _, label := range ts.Labels {
			if label.Name == "__name__" {
				continue
			}

			tagDataIntf, _ := metricLabels.LoadOrStore(label.Name, &metrics.TagData{
				Capacity:    config.CardinalityLimit.Capacity,
				Cardinality: 0,
				Values:      list.New(),
			})

			tagData := tagDataIntf.(*metrics.TagData)

			if tagData.Values == nil {
				tagData.Values = list.New()
			}

			found := false
			for e := tagData.Values.Front(); e != nil; e = e.Next() {
				value, ok := e.Value.(string)
				if !ok {
					fmt.Printf("value is not a string, it's a %T\n", e.Value)
					continue
				}
				if value == label.Value {
					found = true
					break
				}
			}

			if !found {
				tagData.Cardinality++
				tagData.Values.PushBack(label.Value)

				if tagData.Values != nil && tagData.Values.Len() > tagData.Capacity {
					firstElement := tagData.Values.Front()
					if firstElement != nil {
						tagData.Values.Remove(firstElement)
					}
				}

				if cardinalityPlugin != nil {
					cardinalityPlugin.Handle(metricName, label.Name, tagData, stats)
				}

				metricLabels.Store(label.Name, tagData)
			}

			// Increment the tag label name count in metricsCardinality
			_, ok := metricsCardinality[metricName][label.Name]
			if ok {
				metricsCardinality[metricName][label.Name]++
			} else {
				metricsCardinality[metricName][label.Name] = 1
			}
		}
	}

	// Filter the counters based on the jsonMinCardinality
  filteredMetricsCardinality := make(map[string]map[string]int)
  for metricName, labels := range metricsCardinality {
    for labelName, count := range labels {
      if count >= config.JsonMinCardinality {  // fixed field name
        if filteredMetricsCardinality[metricName] == nil {
          filteredMetricsCardinality[metricName] = make(map[string]int)
        }
        filteredMetricsCardinality[metricName][labelName] = count
      }
    }
  }

	// Convert the result to JSON
	result, err := json.Marshal(filteredMetricsCardinality)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write the result to response
	w.Header().Set("Content-Type", "application/json")
	w.Write(result)
}
