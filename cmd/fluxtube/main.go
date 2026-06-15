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
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	fluxtube "github.com/jodacame/fluxtube"
	"github.com/jodacame/fluxtube/internal/api"
	"github.com/jodacame/fluxtube/internal/config"
	"github.com/jodacame/fluxtube/internal/extractor"
	"github.com/jodacame/fluxtube/internal/stream"
)

func main() {
	configDir := getenv("FT_CONFIG_DIR", "/config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		log.Fatalf("config dir: %v", err)
	}

	store, err := config.Open(filepath.Join(configDir, "fluxtube.db"))
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer store.Close()

	cfg := applyEnv(store)

	ex := extractor.New(extractor.Options{
		YtDlpPath:    getenv("FT_YTDLP", "yt-dlp"),
		CookiesFile:  cfg.YouTube.CookiesFile,
		ExtractorArg: cfg.YouTube.ExtractorArg,
		AutoSubLangs: cfg.Quality.AutoSubLangs,
	})

	eng, err := stream.New(ex, stream.Options{
		FFmpegPath:       getenv("FT_FFMPEG", "ffmpeg"),
		CacheRoot:        cfg.Cache.Path,
		SegmentSeconds:   cfg.Cache.SegmentSeconds,
		IdleTimeout:      time.Duration(cfg.Limits.IdleTimeoutSec) * time.Second,
		MaxSessions:      cfg.Limits.MaxSessions,
		MaxFFmpeg:        cfg.Limits.MaxFFmpeg,
		DefaultMaxHeight: cfg.Quality.DefaultMaxHeight,
	})
	if err != nil {
		log.Fatalf("engine: %v", err)
	}
	defer eng.Close()

	srv := api.New(store, ex, eng, fluxtube.DistFS())

	addr := net.JoinHostPort(cfg.Net.ListenHost, strconv.Itoa(cfg.Net.ListenPort))
	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 15 * time.Second,
	}

	go func() {
		log.Printf("FluxTube %s listening on http://%s", api.Version, addr)
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

// applyEnv overlays environment variables onto persisted settings (env wins for
// the few boot-critical values) and returns the effective configuration.
func applyEnv(store *config.Store) config.Settings {
	cfg := store.Get()
	changed := false
	if v := os.Getenv("FT_LISTEN_HOST"); v != "" {
		cfg.Net.ListenHost = v
		changed = true
	}
	if v := os.Getenv("FT_LISTEN_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.Net.ListenPort = p
			changed = true
		}
	}
	if v := os.Getenv("FT_CACHE_PATH"); v != "" {
		cfg.Cache.Path = v
		changed = true
	}
	if v := os.Getenv("FT_API_TOKEN"); v != "" {
		cfg.APIToken = v
		changed = true
	}
	if v := os.Getenv("FT_COOKIES"); v != "" {
		cfg.YouTube.CookiesFile = v
		changed = true
	}
	if changed {
		_ = store.PutSettings(cfg)
	}
	return cfg
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
