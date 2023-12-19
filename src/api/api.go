package api

import (
	"encoding/json"
	"net/http"

	"github.com/szibis/prometheus-governance-proxy/config"
  "github.com/szibis/prometheus-governance-proxy/metrics"
)

func HandleMetricsCardinality(w http.ResponseWriter, configCh *config.Config) {
  filteredMetricsData := metrics.GetMetricsData(configCh.JsonMinCardinality)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"metrics": filteredMetricsData})
}
