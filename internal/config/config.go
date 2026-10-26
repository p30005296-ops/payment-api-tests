// Package config loads test-run settings from environment variables,
// falling back to a .env file in the repository root.
package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const defaultBaseURL = "https://gateway-recruitment-sdet-f67brx13-675279313328.europe-west1.run.app"

type Config struct {
	BaseURL      string
	Email        string
	ClientSecret string
	Timeout      time.Duration
	// BrowserChannel optionally runs the UI test in an installed browser
	// ("msedge", "chrome") instead of Playwright's bundled Chromium.
	BrowserChannel string
}

// Load reads configuration. Real environment variables always win over .env values.
func Load() (Config, error) {
	loadDotEnv()

	timeout := 30 * time.Second
	if v := os.Getenv("API_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return Config{}, fmt.Errorf("API_TIMEOUT: %w", err)
		}
		timeout = d
	}

	cfg := Config{
		BaseURL:        strings.TrimRight(getenv("API_BASE_URL", defaultBaseURL), "/"),
		Email:          os.Getenv("API_EMAIL"),
		ClientSecret:   os.Getenv("API_CLIENT_SECRET"),
		Timeout:        timeout,
		BrowserChannel: os.Getenv("PW_BROWSER_CHANNEL"),
	}
	if cfg.Email == "" || cfg.ClientSecret == "" {
		return Config{}, errors.New("API_EMAIL and API_CLIENT_SECRET must be set (copy .env.example to .env and fill them in)")
	}
	return cfg, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// loadDotEnv walks up from the working directory to the module root (go.mod)
// and applies KEY=VALUE pairs from .env without overriding existing variables.
func loadDotEnv() {
	dir, err := os.Getwd()
	if err != nil {
		return
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return
		}
		dir = parent
	}

	f, err := os.Open(filepath.Join(dir, ".env"))
	if err != nil {
		return
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if _, set := os.LookupEnv(key); !set {
			os.Setenv(key, strings.Trim(strings.TrimSpace(val), `"'`))
		}
	}
}
