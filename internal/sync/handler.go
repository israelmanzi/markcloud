package sync

import (
	"strings"

	"github.com/israelmanzi/markcloud/internal/frontmatter"
	"github.com/israelmanzi/markcloud/internal/markdown"
	"github.com/israelmanzi/markcloud/internal/store"
)

type ManifestEntry struct {
	Path string `json:"path"`
	SHA  string `json:"sha"`
}

type FileEntry struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	SHA     string `json:"sha"`
}

type DiffResult struct {
	NeedContent []string `json:"need_content"`
}

type Handler struct {
	store *store.Store
}

func NewHandler(s *store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Diff(manifest []ManifestEntry) (*DiffResult, error) {
	existing, err := h.store.GetAllSHAs()
	if err != nil {
		return nil, err
	}

	result := &DiffResult{}
	for _, entry := range manifest {
		if sha, ok := existing[entry.Path]; !ok || sha != entry.SHA {
			result.NeedContent = append(result.NeedContent, entry.Path)
		}
	}

	return result, nil
}

func (h *Handler) Apply(files []FileEntry, allPaths []string) error {
	for _, f := range files {
		meta, body, err := frontmatter.Parse([]byte(f.Content))
		if err != nil {
			return err
		}

		title := markdown.ExtractTitle(body)
		result, err := markdown.RenderFull(body)
		if err != nil {
			return err
		}

		doc := &store.Document{
			Path:        f.Path,
			Title:       title,
			ContentMD:   string(body),
			ContentHTML: result.HTML,
			SHA:         f.SHA,
			Public:      meta.Public,
			Tags:        meta.Tags,
			TOCHTML:     result.TOC,
			Description: result.Description,
		}

		if err := h.store.Upsert(doc); err != nil {
			return err
		}

		// Normalize link targets to store paths (e.g. /some-page → some-page.md)
		var targetPaths []string
		for _, link := range result.Links {
			target := strings.TrimPrefix(link, "/")
			if target == "" {
				continue
			}
			if !strings.HasSuffix(target, ".md") {
				target += ".md"
			}
			targetPaths = append(targetPaths, target)
		}
		if err := h.store.UpsertBacklinks(f.Path, targetPaths); err != nil {
			return err
		}
	}

	return h.store.DeleteAllExcept(allPaths)
}
