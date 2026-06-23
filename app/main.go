package main

import (
	"fmt"
	"net/http"
)

func main() {
	// Endpoint checking status k8s
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	})

	// Endpoint metrics format to Prometheus (Phase 5 usage)
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "# HELP app_up Service is up")
		fmt.Fprintln(w, "# TYPE app_up gauge")
		fmt.Fprintln(w, "app_up 1")
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello DevSecOps")
	})
	
	http.ListenAndServe(":8080", nil) // nosemgrep: go.lang.security.audit.net.use-tls.use-tls
}AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
