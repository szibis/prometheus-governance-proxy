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

  main "github.com/szibis/prometheus-governance-proxy"
  "github.com/szibis/prometheus-governance-proxy/plugins"
)

func handleMetrics(
    w http.ResponseWriter,
    r *http.Request,
    workItems chan<- WorkItem,
    endpoints []string,
    config *main.Config,
    stats *main.Stats) {

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

    // Instantiate the plugins here
    var cardinalityPlugin *plugins.Cardinality
    if config.CardinalityLimit.Enable {
        cardinalityPlugin = plugins.NewCardinality(config)
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

            tagDataIntf, _ := metricLabels.LoadOrStore(label.Name, &main.TagData{
                Capacity:    config.CardinalityLimit.Capacity,
                Cardinality: 0,
                Values:      list.New(),
            })

            tagData := tagDataIntf.(*main.TagData)

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
        }
    }
}
