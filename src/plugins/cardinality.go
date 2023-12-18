package plugins

import (
	"fmt"
	"sync"
)

type Cardinality struct {
	config *main.Config
}

func NewCardinality(config *main.Config) *Cardinality {
	return &Cardinality{config: config}
}

func (c *Cardinality) Handle(metricName string, labelName string, tagData *main.TagData, stats *main.Stats) {

	metricLabelsIntf, ok := main.MetricsData.Metrics.Load(metricName)
	if !ok {
		return // No metric found, nothing to clean up
	}

	metricLabels := metricLabelsIntf.(*sync.Map)

	switch c.config.CardinalityLimit.Mode {
	case "drop_metric":
		if tagData.Cardinality > c.config.CardinalityLimit.Limit {
			fmt.Println("Dropping metric due to cardinality limit for metric:", metricName, "and tag:", labelName)
			main.MetricsData.Metrics.Delete(metricName)

			// Increment DroppedMetrics counter
			stats.DroppedMetrics++
		}

	case "drop_tag":
		if tagData.Cardinality > c.config.CardinalityLimit.Limit {
			fmt.Println("Dropping tag due to cardinality limit for metric:", metricName, "and tag:", labelName)
			metricLabels.Delete(labelName)

			// Increment DroppedTags counter
			stats.DroppedTags++
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
