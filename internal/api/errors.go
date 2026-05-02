package api

import (
	"encoding/json"
	"net/http"
)

type Error struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Detail string `json:"detail,omitempty"`
	Status int    `json:"status"`
}

func WriteError(w http.ResponseWriter, errorType, title, detail string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(Error{
		Type:   errorType,
		Title:  title,
		Detail: detail,
		Status: status,
	})
}

func WriteUnauthorizedError(w http.ResponseWriter) {
	WriteError(w, "unauthorized", "Unauthorized", "User ID not found in session", http.StatusUnauthorized)
}
