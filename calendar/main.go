package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	log.Print("calendar service ready")
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "calendar service response")
	})
	http.ListenAndServe(":50051", nil)
}
