package cli

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	GitHubRepo  string `yaml:"github_repo"`
	GitHubToken string `yaml:"github_token"`
}

func LoadConfig() (*Config, error) {
	home, _ := os.UserHomeDir()
	return LoadConfigFrom(filepath.Join(home, ".markcloud.yaml"))
}

func LoadConfigFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
