// Command fluxtube is the single-binary YouTube -> HTTP streaming bridge.
// It resolves a video on demand and exposes a seekable HTTP stream plus a
// headless discovery API, with the web UI embedded via go:embed.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	fluxtube "github.com/jodacame/fluxtube"
)

// Version is the running build version, overridable at link time.
var Version = "dev"

func main() {
	host := getenv("FT_LISTEN_HOST", "0.0.0.0")
	port := getenv("FT_LISTEN_PORT", "7002")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"version":%q,"status":"ok"}`, Version)
	})
	mux.Handle("/", spaHandler(fluxtube.DistFS()))

	addr := net.JoinHostPort(host, port)
	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 15 * time.Second,
	}

	go func() {
		log.Printf("FluxTube %s listening on http://%s", Version, addr)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	fmt.Println()
	log.Println("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(ctx)
	log.Println("bye")
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// portInt is a small helper kept for future numeric port needs.
func portInt(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
