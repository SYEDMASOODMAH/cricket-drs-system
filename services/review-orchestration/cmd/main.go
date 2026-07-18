// Command review-orchestration is the entrypoint for the review-orchestration service.
//
// This is a Phase 1 scaffold: it starts an HTTP server with a health check
// endpoint only. Domain logic lands in internal/ as each subsequent phase
// builds it out (see /docs/phases.md).
package main

import (
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	log.Println("review-orchestration listening on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
