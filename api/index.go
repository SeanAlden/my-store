package handler

import (
	"encoding/json"
	"net/http"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := map[string]string{
		"message": "Hello from Golang on Vercel!",
	}

	json.NewEncoder(w).Encode(response)
}