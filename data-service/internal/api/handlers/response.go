package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

type apiError struct {
	Error   string      `json:"error"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if v == nil {
		return
	}

	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string, details interface{}) {
	writeJSON(w, status, apiError{
		Error:   code,
		Message: message,
		Details: details,
	})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst interface{}) bool {

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid json body", map[string]any{"error": err.Error()})
		return false
	}

	if err := dec.Decode(&struct{}{}); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid json body", map[string]any{"error": "extra data after json"})
		return false
	}

	return true
}
