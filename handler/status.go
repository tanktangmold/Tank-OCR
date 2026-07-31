package handler

import (
	"encoding/json"
	"net/http"
)

type StatusResponse struct {
	Status       string `json:"status"`
	Engine       string `json:"engine"`
	Version      string `json:"version"`
	MaxDimension int    `json:"max_dimension"`
	ErrorDetail  string `json:"detail,omitempty"`
}

func HandleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if ActiveEngine == nil {
		json.NewEncoder(w).Encode(StatusResponse{
			Status:      "error",
			ErrorDetail: "OCR Engine is not initialized",
		})
		return
	}

	version, err := ActiveEngine.Version()
	if err != nil {
		json.NewEncoder(w).Encode(StatusResponse{
			Status:      "error",
			ErrorDetail: "Failed to get version: " + err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(StatusResponse{
		Status:       "ready",
		Engine:       ActiveEngine.Name(),
		Version:      version,
		MaxDimension: MaxImageDimension,
	})
}
