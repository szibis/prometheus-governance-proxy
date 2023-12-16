package main

import (
    "net/http"
    "io/ioutil"
    "sync"
    "fmt"
    "github.com/golang/snappy"
    "github.com/golang/protobuf/proto"
    "github.com/prometheus/prometheus/prompb"
    "container/list"
)

// Define WorkItem outside for your use case
type WorkItem struct {
    ts prompb.TimeSeries
    endpoints []string
    debug bool
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

        var req prompb.WriteRequest
        if err := proto.Unmarshal(data, &req); err != nil {
            fmt.Println("Error unmarshalling the WriteRequest:", err)
            return
        }

        for _, ts := range req.Timeseries {
            workItems <- WorkItem{ts: ts, endpoints: endpoints, debug: debug}

            metricName := ""
            for _, label := range ts.Labels {
                if label.Name == "__name__" {
                    metricName = label.Value
                    break
                }
            }

            // if metric name not found, continue to next time series
            if metricName == "" {
                continue
            }

            metricLabelsInterface, _ := metricsData.Metrics.LoadOrStore(metricName, &sync.Map{})
            metricLabels := metricLabelsInterface.(*sync.Map)
            
            for _, label := range ts.Labels {
                if label.Name == "__name__" {
                    continue
                }

                tagDataIntf, _ := metricLabels.LoadOrStore(label.Name, &TagData{
                    Capacity:    cardinalityCapacity,
                    Cardinality: 0,
                    Values:      list.New(),
                })

                tagData := tagDataIntf.(*TagData)

                // Check if this tag value already stored
                found := false
                for e := tagData.Values.Front(); e != nil; e = e.Next() {
                    if e.Value.(string) == label.Value {
                        found = true
                        break
                    }
                }

                if !found {
                    tagData.Cardinality++
                    tagData.Values.PushBack(label.Value)

                    if tagData.Values.Len() > tagData.Capacity {
                        tagData.Values.Remove(tagData.Values.Front())
                    }

                    handleCardinalityLimitation(metricName, label.Name, cardinalityLimit, cardinalityLimitMode)

                    metricLabels.Store(label.Name, tagData)
                }
            }
        }
    }
}
