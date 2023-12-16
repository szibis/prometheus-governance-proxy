package main

import (
  "fmt"
  "sync"
)

func handleCardinalityLimitation(metricName string, labelName string, cardinalityLimit int, cardinalityLimitMode string) {
    metricLabelsIntf, ok := metricsData.Metrics.Load(metricName)
    if !ok {
        return // No metric found, nothing to clean up
    }

    metricLabels := metricLabelsIntf.(*sync.Map)
    tagDataIntf, ok := metricLabels.Load(labelName)

    if !ok {
        return // No tag data found, nothing to clean up
    }

    tagData := tagDataIntf.(*TagData)
    switch cardinalityLimitMode {
    case "drop_metric":
        if tagData.Cardinality > cardinalityLimit {
            fmt.Println("Dropping metric due to cardinality limit for metric:", metricName, "and tag:", labelName)
            metricsData.Metrics.Delete(metricName)
        }

    case "drop_tag":
        if tagData.Cardinality > cardinalityLimit {
            fmt.Println("Dropping tag due to cardinality limit for metric:", metricName, "and tag:", labelName)
            metricLabels.Delete(labelName)
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
