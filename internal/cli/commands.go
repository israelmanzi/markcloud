package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/israelmanzi/markcloud/internal/frontmatter"
	"github.com/spf13/cobra"
)

const contentPrefix = "content/"

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "mc",
		Short: "markcloud CLI - manage your markdown documents",
	}

	root.AddCommand(newUploadCmd())
	root.AddCommand(newLsCmd())
	root.AddCommand(newInfoCmd())
	root.AddCommand(newPublicCmd())
	root.AddCommand(newPrivateCmd())
	root.AddCommand(newRmCmd())
	root.AddCommand(newMvCmd())
	root.AddCommand(newStatusCmd())

	return root
}

func getClient() (*GitHubClient, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("load config: %w (create ~/.markcloud.yaml)", err)
	}
	return NewGitHubClient(cfg.GitHubToken, cfg.GitHubRepo)
}

func newUploadCmd() *cobra.Command {
	var filePath, name, dir, tags string
	var public, dryRun bool

	cmd := &cobra.Command{
		Use:   "upload",
		Short: "Upload a markdown file",
		RunE: func(cmd *cobra.Command, args []string) error {
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

			remotePath := contentPrefix + name + ".md"
			if dir != "" {
				remotePath = contentPrefix + strings.Trim(dir, "/") + "/" + name + ".md"
			}

			if dryRun {
				cfg, err := LoadConfig()
				if err != nil {
					return fmt.Errorf("load config: %w", err)
				}
				fmt.Printf("curl -X PUT \\\n")
				fmt.Printf("  -H \"Authorization: token %s\" \\\n", cfg.GitHubToken)
				fmt.Printf("  -H \"Content-Type: application/json\" \\\n")
				fmt.Printf("  https://api.github.com/repos/%s/contents/%s \\\n", cfg.GitHubRepo, remotePath)
				fmt.Printf("  -d '{\"message\":\"upload %s\",\"content\":\"<base64-content>\"}'\n", remotePath)
				return nil
			}

			gh, err := getClient()
			if err != nil {
				return err
			}

			ctx := context.Background()
			msg := fmt.Sprintf("upload %s", remotePath)
			if err := gh.CreateOrUpdateFile(ctx, remotePath, fullContent, msg); err != nil {
				return fmt.Errorf("upload failed: %w", err)
			}

			fmt.Printf("Uploaded → %s\n", remotePath)
			return nil
		},
	}

	cmd.Flags().StringVar(&filePath, "path", "", "Path to markdown file (required)")
	cmd.Flags().StringVar(&name, "name", "", "Document name (defaults to filename)")
	cmd.Flags().StringVar(&dir, "dir", "", "Target directory")
	cmd.Flags().StringVar(&tags, "tags", "", "Comma-separated tags")
	cmd.Flags().BoolVar(&public, "public", false, "Make document public")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print curl command instead of uploading")
	cmd.MarkFlagRequired("path")

	return cmd
}

func newLsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ls [dir]",
		Short: "List documents",
		RunE: func(cmd *cobra.Command, args []string) error {
			gh, err := getClient()
			if err != nil {
				return err
			}

			path := contentPrefix
			if len(args) > 0 {
				path = contentPrefix + args[0]
			}

			ctx := context.Background()
			contents, err := gh.ListDir(ctx, strings.TrimSuffix(path, "/"))
			if err != nil {
				return err
			}

			for _, c := range contents {
				name := c.GetName()
				if c.GetType() == "dir" {
					fmt.Printf("  %s/\n", name)
				} else if strings.HasSuffix(name, ".md") {
					fmt.Printf("  %s\n", strings.TrimSuffix(name, ".md"))
				}
			}
			return nil
		},
	}

	return cmd
}

func newInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info <path>",
		Short: "Show document metadata",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			gh, err := getClient()
			if err != nil {
				return err
			}

			path := contentPrefix + args[0]
			if !strings.HasSuffix(path, ".md") {
				path += ".md"
			}

			ctx := context.Background()
			content, err := gh.GetFile(ctx, path)
			if err != nil {
				return err
			}

			meta, _, err := frontmatter.Parse([]byte(content))
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
	gh, err := getClient()
	if err != nil {
		return err
	}

	path = contentPrefix + path
	if !strings.HasSuffix(path, ".md") {
		path += ".md"
	}

	ctx := context.Background()
	content, err := gh.GetFile(ctx, path)
	if err != nil {
		return err
	}

	meta, body, err := frontmatter.Parse([]byte(content))
	if err != nil {
		return err
	}

	meta.Public = public
	updated := frontmatter.Serialize(meta, body)

	action := "private"
	if public {
		action = "public"
	}

	return gh.CreateOrUpdateFile(ctx, path, updated, fmt.Sprintf("set %s %s", path, action))
}

func newRmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <path>",
		Short: "Delete a document",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			gh, err := getClient()
			if err != nil {
				return err
			}

			path := contentPrefix + args[0]
			if !strings.HasSuffix(path, ".md") {
				path += ".md"
			}

			ctx := context.Background()
			return gh.DeleteFile(ctx, path, fmt.Sprintf("delete %s", path))
		},
	}
}

func newMvCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mv <source> <dest>",
		Short: "Move or rename a document",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			gh, err := getClient()
			if err != nil {
				return err
			}

			src := contentPrefix + args[0]
			dst := contentPrefix + args[1]
			if !strings.HasSuffix(src, ".md") {
				src += ".md"
			}
			if !strings.HasSuffix(dst, ".md") {
				dst += ".md"
			}

			ctx := context.Background()

			content, err := gh.GetFile(ctx, src)
			if err != nil {
				return fmt.Errorf("source not found: %w", err)
			}

			if err := gh.CreateOrUpdateFile(ctx, dst, []byte(content), fmt.Sprintf("move %s to %s", src, dst)); err != nil {
				return err
			}

			return gh.DeleteFile(ctx, src, fmt.Sprintf("move %s to %s (delete old)", src, dst))
		},
	}
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check latest deployment status",
		RunE: func(cmd *cobra.Command, args []string) error {
			gh, err := getClient()
			if err != nil {
				return err
			}

			ctx := context.Background()
			frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
			frame := 0

			for {
				run, err := gh.GetLatestWorkflowRun(ctx)
				if err != nil {
					return err
				}

				status := run.GetStatus()
				if status == "completed" {
					conclusion := run.GetConclusion()
					mark := "✓"
					if conclusion != "success" {
						mark = "✗"
					}
					fmt.Printf("\r%s %s — %s\n", mark, run.GetName(), conclusion)
					fmt.Printf("  %s\n", run.GetHTMLURL())
					return nil
				}

				fmt.Printf("\r%s %s — %s...", frames[frame%len(frames)], run.GetName(), status)
				frame++
				time.Sleep(3 * time.Second)
			}
		},
	}
}
