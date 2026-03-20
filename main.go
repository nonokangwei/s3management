package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/kangwe/s3management/config"
	"github.com/kangwe/s3management/gcs"
	"github.com/kangwe/s3management/middleware"
	"github.com/kangwe/s3management/server"
)

func main() {
	cfg := config.Load()

	if cfg.GCSProjectID == "" {
		log.Fatal("GCS project ID is required. Set GCS_PROJECT_ID env var or use --project flag.")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize GCS client with Application Default Credentials
	gcsClient, err := gcs.NewClient(ctx, cfg.GCSProjectID)
	if err != nil {
		log.Fatalf("Failed to create GCS client: %v", err)
	}
	defer gcsClient.Close()

	// Build handler chain: recovery -> request log -> body limit -> signature bypass -> router
	router := server.NewRouter(gcsClient, cfg.GCSRequestTimeout)

	// Health check mux: route /healthz separately from the proxy handler
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.Handle("/", middleware.Recovery(
		middleware.RequestLog(
			middleware.BodyLimit(cfg.MaxRequestBodyKB*1024)(
				middleware.SignatureBypass(router),
			),
		),
	))

	srv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      mux,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	// Graceful shutdown on SIGINT/SIGTERM with timeout
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("Shutting down server...")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer shutdownCancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	log.Printf("S3-to-GCS proxy starting on %s (project: %s)", cfg.ListenAddr, cfg.GCSProjectID)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
	log.Println("Server stopped")
}
