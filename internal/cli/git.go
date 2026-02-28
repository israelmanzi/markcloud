package cli

import (
	"os"
	"os/exec"
	"strings"
)

func gitRun(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func gitOutput(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

func gitInit(dir string) error {
	if isGitRepo(dir) {
		return nil
	}
	return gitRun(dir, "init")
}

func gitCommit(dir, message string) error {
	if err := gitRun(dir, "add", "-A"); err != nil {
		return err
	}
	// Check if there's anything to commit
	status, err := gitOutput(dir, "status", "--porcelain")
	if err != nil {
		return err
	}
	if status == "" {
		return nil
	}
	return gitRun(dir, "commit", "-m", message)
}

func isGitRepo(dir string) bool {
	info, err := os.Stat(dir + "/.git")
	return err == nil && info.IsDir()
}
