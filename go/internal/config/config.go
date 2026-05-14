package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	ServerURL string `json:"server_url"`
	APIKey    string `json:"api_key"`
	VerifySSL bool   `json:"verify_ssl"`
	Language  string `json:"language"`
}

const configFile = "config.json"

func Load() (*Config, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.Language == "" {
		cfg.Language = "en"
	}
	return &cfg, nil
}

func Save(cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configFile, data, 0644)
}

func Setup() *Config {
	fmt.Println("--- Miniflux TUI Setup ---")

	var serverURL string
	fmt.Print("Server URL (e.g., https://miniflux.example.com): ")
	fmt.Scanln(&serverURL)

	var apiKey string
	fmt.Print("API Key: ")
	fmt.Scanln(&apiKey)

	var verifyStr string
	fmt.Print("Verify SSL certificates? (Y/n): ")
	fmt.Scanln(&verifyStr)
	verifySSL := strings.ToLower(strings.TrimSpace(verifyStr)) != "n"

	var lang string
	fmt.Print("Interface language (en/cn) [default: en]: ")
	fmt.Scanln(&lang)
	lang = strings.ToLower(strings.TrimSpace(lang))
	if lang != "cn" && lang != "en" {
		lang = "en"
	}

	cfg := &Config{
		ServerURL: strings.TrimRight(serverURL, "/"),
		APIKey:    strings.TrimSpace(apiKey),
		VerifySSL: verifySSL,
		Language:  lang,
	}

	if err := Save(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Configuration saved to %s\n", configFile)
	return cfg
}
