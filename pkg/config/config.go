package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Confluence ConfluenceConfig `yaml:"confluence"`
}

// ConfluenceConfig holds Confluence-specific configuration
type ConfluenceConfig struct {
	URL          string `yaml:"url"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
	Space        string `yaml:"space"`
	ParentPageID string `yaml:"parent_page_id,omitempty"`
}

// LoadConfig loads configuration with priority handling:
// 1. Command-line arguments (highest priority)
// 2. Environment variables
// 3. Config file (lowest priority)
func LoadConfig(configPath string, cliArgs map[string]string) (*Config, error) {
	config := &Config{
		Confluence: ConfluenceConfig{},
	}

	// 1. Try to load from config file (lowest priority)
	if configPath != "" {
		err := loadFromFile(configPath, config)
		if err != nil {
			fmt.Printf("⚠️ Warning: Unable to read config file %s: %s\n", configPath, err)
			fmt.Println("Will try using environment variables...")
		}
	}

	// 2. Load from environment variables (medium priority)
	loadFromEnv(config)

	// 3. Load from CLI arguments (highest priority)
	loadFromCLI(config, cliArgs)

	// Validate required configuration
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	return config, nil
}

// loadFromFile loads configuration from YAML file
func loadFromFile(configPath string, config *Config) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, config)
}

// loadFromEnv loads configuration from environment variables
func loadFromEnv(config *Config) {
	if url := os.Getenv("KMS_URL"); url != "" {
		config.Confluence.URL = url
	}

	if username := os.Getenv("KMS_USERNAME"); username != "" {
		config.Confluence.Username = username
	}

	if password := os.Getenv("KMS_PASSWORD"); password != "" {
		config.Confluence.Password = password
	}

	if space := os.Getenv("KMS_SPACE"); space != "" {
		config.Confluence.Space = space
	}
}

// loadFromCLI loads configuration from command-line arguments
func loadFromCLI(config *Config, cliArgs map[string]string) {
	if url := cliArgs["url"]; url != "" {
		config.Confluence.URL = url
	}

	if username := cliArgs["username"]; username != "" {
		config.Confluence.Username = username
	}

	if password := cliArgs["password"]; password != "" {
		config.Confluence.Password = password
	}

	if space := cliArgs["space"]; space != "" {
		config.Confluence.Space = space
	}
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	var missingKeys []string

	if config.Confluence.URL == "" {
		missingKeys = append(missingKeys, "url")
	}

	if config.Confluence.Username == "" {
		missingKeys = append(missingKeys, "username")
	}

	if config.Confluence.Password == "" {
		missingKeys = append(missingKeys, "password")
	}

	if config.Confluence.Space == "" {
		missingKeys = append(missingKeys, "space")
	}

	if len(missingKeys) > 0 {
		return fmt.Errorf("missing required configuration items: %s\n"+
			"Please provide configuration through one of:\n"+
			"1. Command-line arguments:\n"+
			"   --url, --username, --password, --space\n"+
			"2. Environment variables:\n"+
			"   KMS_URL, KMS_USERNAME, KMS_PASSWORD, KMS_SPACE\n"+
			"3. Configuration file", strings.Join(missingKeys, ", "))
	}

	return nil
} 