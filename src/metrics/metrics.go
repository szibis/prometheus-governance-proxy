package metrics

import (
	"container/list"
	"sync"
)

type TagData struct {
    Cardinality int
    Values      *list.List
    Capacity    int
}

type MetricsDataStruct struct {
    Metrics *sync.Map
}

var MetricsData = MetricsDataStruct {
    Metrics: new(sync.Map),
}

type MetricData struct {
    Name     string
    TagData  *TagData
}

func GetMetricsData(minCardinality int) []*MetricData {
    filteredMetricsData := make([]*MetricData, 0)

    MetricsData.Metrics.Range(func(k, v interface{}) bool {
        if data, ok := v.(*TagData); ok {
            if data.Cardinality >= minCardinality {
                filteredMetricsData = append(filteredMetricsData, &MetricData{Name: k.(string), TagData: data})
            }
        }
        return true
    })

    return filteredMetricsData
}
