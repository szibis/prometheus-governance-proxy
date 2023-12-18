package api

import (
	"encoding/json"
	"net/http"

	"github.com/szibis/prometheus-governance-proxy/metrics"
	"github.com/szibis/prometheus-governance-proxy/config"
)

func HandleMetricsCardinality(w http.ResponseWriter, configCh *config.Config) {
	data := metrics.GetMetricsData(configCh.JsonMinCardinality)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"metrics": data})
}
