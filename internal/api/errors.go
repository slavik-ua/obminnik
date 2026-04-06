package api

import (
	"encoding/json"
	"net/http"
)

type APIError struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Detail string `json:"detail,omitempty"`
	Status int    `json:"status"`
}

func WriteError(w http.ResponseWriter, errorType, title, detail string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	json.NewEncoder(w).Encode(APIError{
		Type:   errorType,
		Title:  title,
		Detail: detail,
		Status: status,
	})
}
