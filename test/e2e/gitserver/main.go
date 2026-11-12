/*
Copyright 2026, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

// gitserver serves the bare git repositories below GIT_PROJECT_ROOT over the smart HTTP protocol,
// using `git http-backend`. It is only meant for e2e tests: it has no authentication and is read-only
// (receive-pack is disabled by git http-backend unless explicitly enabled).
package main

import (
	"log"
	"net/http"
	"net/http/cgi"
	"os"
	"os/exec"
	"time"
)

func main() {
	root := getenv("GIT_PROJECT_ROOT", "/srv/git")
	addr := getenv("LISTEN_ADDRESS", ":8080")

	gitPath, err := exec.LookPath("git")
	if err != nil {
		log.Fatalf("git binary not found: %v", err)
	}

	handler := &cgi.Handler{
		Path: gitPath,
		Args: []string{"http-backend"},
		// Root is empty so PATH_INFO is the full request path, e.g. /capabilities.git/info/refs
		Env: []string{
			"GIT_PROJECT_ROOT=" + root,
			"GIT_HTTP_EXPORT_ALL=1",
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.Handle("/", handler)

	server := &http.Server{
		Addr:              addr,
		Handler:           logRequests(mux),
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("serving git repositories from %s on %s", root, addr)
	log.Fatal(server.ListenAndServe())
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" {
			log.Printf("%s %s", r.Method, r.URL.String())
		}
		next.ServeHTTP(w, r)
	})
}

func getenv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}
