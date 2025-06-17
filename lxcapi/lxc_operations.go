package lxcapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// OperationsHandler handles the synchronization request. It processes the HTTP request
// and sends the appropriate response back to the client.
func OperationsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Println("Request Method:", r.Method, "|", "Request API:", r.URL.Path)

	//recursion := r.URL.Query().Get("recursion")
	//allProjects := r.URL.Query().Get("all-projects")

	if len(r.TLS.PeerCertificates) > 0 {

		// if all-projects is "true", List all.  // Not im
		operationsList, err := ListOperations()
		if err != nil {
			response := map[string]any{
				"type":        "sync",
				"status":      "Seccess",
				"status_code": 200,
				"operation":   "",
				"error_code":  0,
				"error":       err.Error(),
				"metadata":    map[string]any{},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		successMetadata := []map[string]any{}
		runningMetadata := []map[string]any{}

		for _, operation := range operationsList {
			operationData := map[string]any{
				"id":          operation.ID,
				"class":       operation.Class,
				"description": operation.Description,
				"created_at":  operation.CreatedAt.Format(time.RFC3339),
				"updated_at":  operation.UpdatedAt.Format(time.RFC3339),
				"status":      operation.Status,
				"status_code": 200,
				"metadata":    nil,
				"may_cancel":  false,
				"err":         operation.Err,
				"location":    "none",
				"resources": map[string]any{
					"instances": []string{"/1.0/instances/" + operation.Instances},
				},
			}

			// May be have
			if operation.Status == "Success" {
				successMetadata = append(successMetadata, operationData)
			} else if operation.Status == "Running" {
				runningMetadata = append(runningMetadata, operationData)
			}
		}

		response := map[string]any{
			"type":        "sync",
			"status":      "Success",
			"status_code": 200,
			"operation":   "",
			"error_code":  0,
			"error":       "",
			"metadata": map[string]any{
				"success": successMetadata,
				"running": runningMetadata,
			},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]any{}
	json.NewEncoder(w).Encode(response)
}
