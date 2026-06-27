package response

import (
	"encoding/json"
	"net/http"
)

type Body struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
	Error   any    `json:"error,omitempty"`
}

func JSON(w http.ResponseWriter, statusCode int, body Body) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(body)
}

func Success(w http.ResponseWriter, statusCode int, message string, data any) {
	JSON(w, statusCode, Body{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func Error(w http.ResponseWriter, statusCode int, message string, err any) {
	JSON(w, statusCode, Body{
		Success: false,
		Message: message,
		Error:   err,
	})
}
