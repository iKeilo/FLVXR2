package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"go-backend/internal/app"
	"go-backend/internal/config"
	"go-backend/internal/license"
	"go-backend/internal/middleware"
	"go-backend/internal/store/repo"
)

func main() {
	cfg := config.FromEnv()
	if cfg.JWTSecret == "" {
		log.Println("warning: JWT_SECRET is empty")
	}

	// If license env values are missing, recover persisted config from the DB.
	// This keeps upgrades and restarts working when .env has lost those values.
	if cfg.LicenseKey == "" {
		log.Println("license env is not configured, trying to restore from database")
		tempRepo, err := getTempRepository(cfg)
		if err == nil && tempRepo != nil {
			cfg1, _ := tempRepo.GetConfigByName("license_server_url")
			cfg2, _ := tempRepo.GetConfigByName("license_key")
			cfg3, _ := tempRepo.GetConfigByName("server_domain")
			tempRepo.Close()

			if cfg2 != nil && cfg2.Value != "" {
				cfg.LicenseKey = cfg2.Value
			}
			if cfg3 != nil && cfg3.Value != "" {
				middleware.UpdateServerDomainFromConfig(cfg3.Value)
			}
			if cfg1 != nil && cfg1.Value != "" {
				cfg.LicenseServerURL = cfg1.Value
			} else if cfg.LicenseKey != "" {
				cfg.LicenseServerURL = license.DefaultServerURL
				log.Println("license server URL restored from compiled default")
			}
			if cfg.LicenseKey != "" {
				log.Println("license config restored from database")
			}
		}
	}
	if cfg.LicenseKey != "" && cfg.LicenseServerURL == "" {
		cfg.LicenseServerURL = license.DefaultServerURL
	}

	// Verify license at startup.
	if cfg.LicenseServerURL != "" && cfg.LicenseKey != "" {
		log.Printf("starting license verification")
		domain := middleware.GetServerDomain()
		if err := middleware.StartLicenseVerification(cfg.LicenseServerURL, cfg.LicenseKey, domain, domain, "https:"); err != nil {
			log.Printf("license verification failed: %v", err)
		} else {
			valid, expireTime, reason, _ := middleware.GetLicenseState()
			if valid {
				log.Printf("license verification succeeded, expires at %s", time.UnixMilli(expireTime).Format("2006-01-02"))
			} else {
				log.Printf("license is invalid: %s", reason)
			}
		}
	} else {
		log.Println("license server is not configured, running in evaluation mode")
	}

	log.Printf("starting go-backend on %s (db=%s, version=%s)", cfg.Addr, cfg.DBPath, cfg.FluxVersion)

	a, err := app.New(cfg)
	if err != nil {
		log.Fatalf("failed to create app: %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- a.Run()
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		log.Printf("received signal %s, shutting down", sig)
	case runErr := <-errCh:
		if runErr != nil && !errors.Is(runErr, http.ErrServerClosed) {
			log.Fatalf("server stopped unexpectedly: %v", runErr)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := a.Shutdown(ctx); err != nil {
		log.Fatalf("shutdown failed: %v", err)
	}
}

func getTempRepository(cfg config.Config) (*repo.Repository, error) {
	dialectorType := strings.ToLower(strings.TrimSpace(cfg.DBType))
	switch dialectorType {
	case "", "sqlite":
		return repo.Open(cfg.DBPath)
	case "postgres", "postgresql":
		return repo.OpenPostgres(cfg.DatabaseURL)
	}
	return nil, nil
}
