package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

var port = ":10000"

type Message struct {
	Content string
}

func main() {
	// env overriding port
	if os.Getenv("HEALTH_PORT") != "" {
		port = os.Getenv("HEALTH_PORT")
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		      fmt.Fprintf(w, "OK")
	})

  log.Print(fmt.Sprintf("Listening on %s...", port))

	log.Fatal(http.ListenAndServe(port, nil))
}
