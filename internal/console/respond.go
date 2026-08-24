package console

import (
	"encoding/json"
	"net/http"
)

// writeJSON encodes a payload as JSON.
func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// writeError encodes a plain error message.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// decodeJSON reads a JSON request body.
func decodeJSON(r *http.Request, target any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

// ok writes a small success envelope.
func ok(w http.ResponseWriter, detail string) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "detail": detail})
}
