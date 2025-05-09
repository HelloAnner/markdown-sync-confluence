package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config 配置
type Config struct {
	Confluence ConfluenceConfig `yaml:"confluence"`
}

// ConfluenceConfig confluence 配置
type ConfluenceConfig struct {
	URL          string `yaml:"url"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
	Space        string `yaml:"space"`
	ParentPageID string `yaml:"parent_page_id,omitempty"`
}

// LoadConfig  按照优先级加载配置
// 1. 最高优先级: 命令行参数
// 2. 次高优先级: 环境变量
func LoadConfig(cliArgs map[string]string) (*Config, error) {
	config := &Config {
		Confluence: ConfluenceConfig{},
	}

	// 1. 从环境变量加载 (次高优先级)
	loadFromEnv(config)

	// 2. 从命令行参数加载 (最高优先级)
	loadFromCLI(config, cliArgs)

	// 验证必填配置
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	return config, nil
}

// loadFromFile 从 YAML 文件加载配置
func loadFromFile(configPath string, config *Config) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, config)
}

// loadFromEnv 从环境变量加载配置
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

// loadFromCLI 从命令行参数加载配置
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

// validateConfig 验证配置
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