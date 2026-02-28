package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte("github_repo: user/repo\ngithub_token: ghp_test\n"), 0644)

	cfg, err := LoadConfigFrom(path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if cfg.GitHubRepo != "user/repo" {
		t.Errorf("expected user/repo, got %s", cfg.GitHubRepo)
	}
	if cfg.GitHubToken != "ghp_test" {
		t.Errorf("expected ghp_test, got %s", cfg.GitHubToken)
	}
}
