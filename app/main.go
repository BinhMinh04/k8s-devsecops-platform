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
	// nosemgrep: go.lang.security.audit.net.use-tls.use-tls
	// TLS is terminated at the Ingress layer (Traefik/Nginx). Internal service communication uses plain HTTP per K8s best practice.
	http.ListenAndServe(":8080", nil)
}