package cli

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/israelmanzi/markcloud/internal/frontmatter"
	gosync "github.com/israelmanzi/markcloud/internal/sync"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "mc",
		Short: "markcloud CLI - manage your markdown documents",
	}

	root.AddCommand(newSyncCmd())
	root.AddCommand(newUploadCmd())
	root.AddCommand(newLsCmd())
	root.AddCommand(newInfoCmd())
	root.AddCommand(newPublicCmd())
	root.AddCommand(newPrivateCmd())
	root.AddCommand(newRmCmd())
	root.AddCommand(newMvCmd())

	return root
}

func getConfig() (*Config, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("load config: %w (create ~/.markcloud.yaml)", err)
	}
	return cfg, nil
}

func getServerClient(cfg *Config) *ServerClient {
	return NewServerClient(cfg.ServerURL, cfg.DeploySecret)
}

// walkContent collects all .md files under content_dir, returning manifest entries,
// file entries for those needing upload, and all relative paths.
func walkContent(dir string) ([]gosync.ManifestEntry, map[string][]byte, []string, error) {
	var manifest []gosync.ManifestEntry
	contents := make(map[string][]byte)
	var allPaths []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if info.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}

		rel, _ := filepath.Rel(dir, path)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		sha := fmt.Sprintf("%x", sha256.Sum256(data))
		manifest = append(manifest, gosync.ManifestEntry{Path: rel, SHA: sha})
		contents[rel] = data
		allPaths = append(allPaths, rel)
		return nil
	})

	return manifest, contents, allPaths, err
}

func syncAll(cfg *Config, client *ServerClient) error {
	manifest, contents, allPaths, err := walkContent(cfg.ContentDir)
	if err != nil {
		return fmt.Errorf("walking content: %w", err)
	}

	if len(manifest) == 0 {
		fmt.Println("No markdown files found")
		return nil
	}

	needContent, err := client.Manifest(manifest)
	if err != nil {
		return err
	}

	var files []gosync.FileEntry
	for _, path := range needContent {
		data, ok := contents[path]
		if !ok {
			continue
		}
		sha := fmt.Sprintf("%x", sha256.Sum256(data))
		files = append(files, gosync.FileEntry{
			Path:    path,
			Content: string(data),
			SHA:     sha,
		})
	}

	if len(files) == 0 && len(allPaths) > 0 {
		// Still send all_paths so server can clean orphans
		return client.Upload(nil, allPaths)
	}

	if len(files) > 0 {
		if err := client.Upload(files, allPaths); err != nil {
			return err
		}
		fmt.Printf("Synced %d file(s)\n", len(files))
	}

	return nil
}

func newSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Sync content directory to server",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := getConfig()
			if err != nil {
				return err
			}

			if err := gitInit(cfg.ContentDir); err != nil {
				return fmt.Errorf("git init: %w", err)
			}
			if err := gitCommit(cfg.ContentDir, "auto-commit before sync"); err != nil {
				return fmt.Errorf("git commit: %w", err)
			}

			fmt.Println("Syncing...")
			return syncAll(cfg, getServerClient(cfg))
		},
	}
}

func newUploadCmd() *cobra.Command {
	var filePath, name, dir, tags string
	var public bool

	cmd := &cobra.Command{
		Use:   "upload",
		Short: "Upload a markdown file",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := getConfig()
			if err != nil {
				return err
			}

			content, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("reading file: %w", err)
			}

			if name == "" {
				name = strings.TrimSuffix(filepath.Base(filePath), ".md")
			}

			meta, body, _ := frontmatter.Parse(content)
			if tags != "" {
				meta.Tags = strings.Split(tags, ",")
			}
			if cmd.Flags().Changed("public") {
				meta.Public = public
			}

			fullContent := frontmatter.Serialize(meta, body)

			relPath := name + ".md"
			if dir != "" {
				relPath = strings.Trim(dir, "/") + "/" + name + ".md"
			}

			destPath := filepath.Join(cfg.ContentDir, relPath)
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return err
			}
			if err := os.WriteFile(destPath, fullContent, 0644); err != nil {
				return err
			}

			if err := gitInit(cfg.ContentDir); err != nil {
				return fmt.Errorf("git init: %w", err)
			}
			if err := gitCommit(cfg.ContentDir, fmt.Sprintf("upload %s", relPath)); err != nil {
				return fmt.Errorf("git commit: %w", err)
			}

			fmt.Printf("Uploaded → %s\n", relPath)
			fmt.Println("Syncing...")
			return syncAll(cfg, getServerClient(cfg))
		},
	}

	cmd.Flags().StringVar(&filePath, "path", "", "Path to markdown file (required)")
	cmd.Flags().StringVar(&name, "name", "", "Document name (defaults to filename)")
	cmd.Flags().StringVar(&dir, "dir", "", "Target directory")
	cmd.Flags().StringVar(&tags, "tags", "", "Comma-separated tags")
	cmd.Flags().BoolVar(&public, "public", false, "Make document public")
	cmd.MarkFlagRequired("path")

	return cmd
}

func newLsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ls [dir]",
		Short: "List documents",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := getConfig()
			if err != nil {
				return err
			}

			root := cfg.ContentDir
			if len(args) > 0 {
				root = filepath.Join(cfg.ContentDir, args[0])
			}

			entries, err := os.ReadDir(root)
			if err != nil {
				return fmt.Errorf("listing %s: %w", root, err)
			}

			for _, e := range entries {
				name := e.Name()
				if e.IsDir() {
					if name == ".git" {
						continue
					}
					fmt.Printf("  %s/\n", name)
				} else if strings.HasSuffix(name, ".md") {
					fmt.Printf("  %s\n", strings.TrimSuffix(name, ".md"))
				}
			}
			return nil
		},
	}
}

func newInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info <path>",
		Short: "Show document metadata",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := getConfig()
			if err != nil {
				return err
			}

			path := args[0]
			if !strings.HasSuffix(path, ".md") {
				path += ".md"
			}

			data, err := os.ReadFile(filepath.Join(cfg.ContentDir, path))
			if err != nil {
				return fmt.Errorf("reading %s: %w", path, err)
			}

			meta, _, err := frontmatter.Parse(data)
			if err != nil {
				return err
			}

			fmt.Printf("Path:   %s\n", args[0])
			fmt.Printf("Public: %v\n", meta.Public)
			fmt.Printf("Tags:   %s\n", strings.Join(meta.Tags, ", "))
			return nil
		},
	}
}

func newPublicCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "public <path>",
		Short: "Make a document public",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return setVisibility(args[0], true)
		},
	}
}

func newPrivateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "private <path>",
		Short: "Make a document private",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return setVisibility(args[0], false)
		},
	}
}

func setVisibility(path string, public bool) error {
	cfg, err := getConfig()
	if err != nil {
		return err
	}

	if !strings.HasSuffix(path, ".md") {
		path += ".md"
	}

	fullPath := filepath.Join(cfg.ContentDir, path)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	meta, body, err := frontmatter.Parse(data)
	if err != nil {
		return err
	}

	meta.Public = public
	if err := os.WriteFile(fullPath, frontmatter.Serialize(meta, body), 0644); err != nil {
		return err
	}

	action := "private"
	if public {
		action = "public"
	}

	if err := gitInit(cfg.ContentDir); err != nil {
		return fmt.Errorf("git init: %w", err)
	}
	if err := gitCommit(cfg.ContentDir, fmt.Sprintf("set %s %s", path, action)); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}

	fmt.Printf("Set %s → %s\n", path, action)
	fmt.Println("Syncing...")
	return syncAll(cfg, getServerClient(cfg))
}

func newRmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <path>",
		Short: "Delete a document",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := getConfig()
			if err != nil {
				return err
			}

			path := args[0]
			if !strings.HasSuffix(path, ".md") {
				path += ".md"
			}

			fullPath := filepath.Join(cfg.ContentDir, path)
			if err := os.Remove(fullPath); err != nil {
				return fmt.Errorf("deleting %s: %w", path, err)
			}

			if err := gitInit(cfg.ContentDir); err != nil {
				return fmt.Errorf("git init: %w", err)
			}
			if err := gitCommit(cfg.ContentDir, fmt.Sprintf("delete %s", path)); err != nil {
				return fmt.Errorf("git commit: %w", err)
			}

			fmt.Printf("Deleted %s\n", path)
			fmt.Println("Syncing...")
			return syncAll(cfg, getServerClient(cfg))
		},
	}
}

func newMvCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mv <source> <dest>",
		Short: "Move or rename a document",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := getConfig()
			if err != nil {
				return err
			}

			src := args[0]
			dst := args[1]
			if !strings.HasSuffix(src, ".md") {
				src += ".md"
			}
			if !strings.HasSuffix(dst, ".md") {
				dst += ".md"
			}

			srcPath := filepath.Join(cfg.ContentDir, src)
			dstPath := filepath.Join(cfg.ContentDir, dst)

			if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
				return err
			}
			if err := os.Rename(srcPath, dstPath); err != nil {
				return fmt.Errorf("moving %s to %s: %w", src, dst, err)
			}

			if err := gitInit(cfg.ContentDir); err != nil {
				return fmt.Errorf("git init: %w", err)
			}
			if err := gitCommit(cfg.ContentDir, fmt.Sprintf("move %s to %s", src, dst)); err != nil {
				return fmt.Errorf("git commit: %w", err)
			}

			fmt.Printf("Moved %s → %s\n", src, dst)
			fmt.Println("Syncing...")
			return syncAll(cfg, getServerClient(cfg))
		},
	}
}
