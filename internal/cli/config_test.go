package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte("server_url: https://example.com\ndeploy_secret: secret123\ncontent_dir: ~/notes\n"), 0644)

	cfg, err := LoadConfigFrom(path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if cfg.ServerURL != "https://example.com" {
		t.Errorf("expected https://example.com, got %s", cfg.ServerURL)
	}
	if cfg.DeploySecret != "secret123" {
		t.Errorf("expected secret123, got %s", cfg.DeploySecret)
	}
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, "notes")
	if cfg.ContentDir != expected {
		t.Errorf("expected %s, got %s", expected, cfg.ContentDir)
	}
}
