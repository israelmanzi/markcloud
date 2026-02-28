package cli

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ServerURL    string `yaml:"server_url"`
	DeploySecret string `yaml:"deploy_secret"`
	ContentDir   string `yaml:"content_dir"`
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
	if strings.HasPrefix(cfg.ContentDir, "~/") {
		home, _ := os.UserHomeDir()
		cfg.ContentDir = filepath.Join(home, cfg.ContentDir[2:])
	}
	return &cfg, nil
}
