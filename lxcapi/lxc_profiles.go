package lxcapi

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// ProjectHandler handles the synchronization request. It processes the HTTP request
// and sends the appropriate response back to the client.
func ProfilesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Println("Request Method:", r.Method, "|", "Request API:", r.URL.Path)
	//recursion := r.URL.Query().Get("recursion")

	if IsTrusted(r) {
		response := map[string]any{
			"type":        "sync",
			"status":      "Success",
			"status_code": 200,
			"operation":   "",
			"error_code":  0,
			"error":       "",
			"metadata": []map[string]any{
				{
					"name":        "default",
					"description": "Default LXC profile",
					"config": map[string]any{
						"features.images":          "true",
						"features.networks":        "true",
						"features.networks.zones":  "true",
						"features.profiles":        "true",
						"features.storage.buckets": "true",
						"features.storage.volumes": "true",
					},
					"used_by": []string{
						"/1.0/profiles/default",
						"/1.0/networks/lxdbr0",
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
		return
	}
	response := map[string]any{}
	json.NewEncoder(w).Encode(response)
}
