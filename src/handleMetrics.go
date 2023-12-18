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

func handleMetrics(
    w http.ResponseWriter,
    r *http.Request,
    workItems chan<- WorkItem,
    endpoints []string,
    config *Config,
    stats *Stats) {

    if r.Method != http.MethodPost {
        return
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

    for _, ts := range req.Timeseries {
        workItems <- WorkItem{ts: ts, endpoints: endpoints, debug: config.Debug}

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

        metricLabelsInterface, _ := metricsData.Metrics.LoadOrStore(metricName, &sync.Map{})
        metricLabels := metricLabelsInterface.(*sync.Map)

        for _, label := range ts.Labels {
            if label.Name == "__name__" {
                continue
            }

            tagDataIntf, _ := metricLabels.LoadOrStore(label.Name, &TagData{
                Capacity:    config.CardinalityLimit.Capacity,
                Cardinality: 0,
                Values:      list.New(),
            })

            tagData := tagDataIntf.(*TagData)

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

                handleCardinalityLimitation(metricName, label.Name, config.CardinalityLimit.Limit, config.CardinalityLimit.Mode, stats)
                metricLabels.Store(label.Name, tagData)
            }
        }
    }
}
