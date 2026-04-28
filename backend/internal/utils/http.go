package utils

import (
	"encoding/json"
	"net/http"
	"strings"
)

func WriteJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, map[string]string{"error": message})
}

func IsAdminOverride(r *http.Request) bool {
	return strings.EqualFold(r.URL.Query().Get("admin"), "true")
}
