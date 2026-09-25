package utils

import (
	"encoding/json"
	"net/http"
)

type AppError struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func Success(w http.ResponseWriter, r *http.Request, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func Error(w http.ResponseWriter, r *http.Request, statusCode int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	json.NewEncoder(w).Encode(AppError{
		Success: false,
		Message: err.Error(),
	})
}
