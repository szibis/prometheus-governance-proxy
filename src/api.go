package api

import (
	"encoding/json"
	"net/http"

	"path/to/your/project/metrics"
	"path/to/your/project/config"
)

func HandleMetricsCardinality(w http.ResponseWriter, configCh *config.Config) {
	data := metrics.GetMetricsData(configCh.JsonMinCardinality)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"metrics": data})
}
