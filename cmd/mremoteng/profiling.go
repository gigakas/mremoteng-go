package main

import (
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"strings"
)

// profilingEnv names the variable that turns the profiling endpoint on. Without it
// nothing listens: a diagnostic port must never appear on its own.
const profilingEnv = "MREMOTENG_PPROF"

// profilingAddress is the address to serve the Go profiling endpoints on, or "" when
// profiling is off. Only a loopback address is accepted, because a profiling endpoint
// exposes goroutine stacks and heap contents.
func profilingAddress(value string) (string, error) {
	address := strings.TrimSpace(value)
	if address == "" {
		return "", nil
	}
	if strings.HasPrefix(address, ":") {
		address = "127.0.0.1" + address
	}
	if !strings.HasPrefix(address, "127.0.0.1:") && !strings.HasPrefix(address, "localhost:") {
		return "", fmt.Errorf("%s must be a loopback address, got %q", profilingEnv, value)
	}
	return address, nil
}

// startProfiling serves /debug/pprof on a loopback address when the environment asks for
// it, and reports a broken request on stderr instead of failing to start: a diagnostic
// switch must not stop the application from running.
func startProfiling() {
	address, err := profilingAddress(os.Getenv(profilingEnv))
	if err != nil {
		fmt.Fprintln(os.Stderr, "mremoteng:", err)
		return
	}
	if address == "" {
		return
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/heap", pprof.Handler("heap").ServeHTTP)
	mux.HandleFunc("/debug/pprof/goroutine", pprof.Handler("goroutine").ServeHTTP)
	go func() {
		if serveErr := http.ListenAndServe(address, mux); serveErr != nil {
			fmt.Fprintln(os.Stderr, "mremoteng: profiling endpoint:", serveErr)
		}
	}()
	fmt.Fprintf(os.Stderr, "mremoteng: profiling on http://%s/debug/pprof/\n", address)
}
