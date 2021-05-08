package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type Response struct {
	Version string `json:"version"`
}

func main() {
	log.Print("calendar service ready")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")

		response := &Response{
			Version: "1",
		}

		json.NewEncoder(w).Encode(response)
	})

	http.ListenAndServe(":50051", nil)
}
