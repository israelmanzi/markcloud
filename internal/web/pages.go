package web

import (
	"html/template"
	"net/http"
	"strings"

	"github.com/israelmanzi/markcloud/internal/store"
)

func (s *Server) handlePage(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	authenticated := s.isAuthenticated(r)

	if path == "" {
		if authenticated {
			s.handleDashboard(w, r)
		} else {
			http.Redirect(w, r, "/public", http.StatusSeeOther)
		}
		return
	}

	doc, err := s.store.Get(path + ".md")
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	if doc != nil {
		if !doc.Public && !authenticated {
			http.NotFound(w, r)
			return
		}
		s.renderTemplate(w, "document.html", map[string]any{
			"Doc":           doc,
			"Content":       template.HTML(doc.ContentHTML),
			"Breadcrumbs":   buildBreadcrumbs(path),
			"Authenticated": authenticated,
		})
		return
	}

	prefix := path
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	docs, err := s.store.List(prefix)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	if len(docs) == 0 {
		http.NotFound(w, r)
		return
	}

	entries := buildDirectoryEntries(docs, prefix, authenticated)
	if len(entries) == 0 {
		http.NotFound(w, r)
		return
	}

	s.renderTemplate(w, "directory.html", map[string]any{
		"Path":          path,
		"Entries":       entries,
		"Breadcrumbs":   buildBreadcrumbs(path),
		"Authenticated": authenticated,
	})
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	docs, err := s.store.List("")
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	entries := buildDirectoryEntries(docs, "", true)

	s.renderTemplate(w, "directory.html", map[string]any{
		"Path":          "",
		"Entries":       entries,
		"Breadcrumbs":   nil,
		"Authenticated": true,
		"IsRoot":        true,
	})
}

func (s *Server) handlePublicIndex(w http.ResponseWriter, r *http.Request) {
	docs, err := s.store.List("")
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	var publicDocs []map[string]any
	for _, d := range docs {
		if d.Public {
			path := strings.TrimSuffix(d.Path, ".md")
			publicDocs = append(publicDocs, map[string]any{
				"Path":  path,
				"Title": d.Title,
				"Tags":  d.Tags,
				"Date":  d.UpdatedAt.Format("2006-01-02"),
			})
		}
	}

	s.renderTemplate(w, "public.html", map[string]any{
		"Docs": publicDocs,
	})
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	authenticated := s.isAuthenticated(r)

	if query == "" {
		s.renderTemplate(w, "search.html", map[string]any{
			"Authenticated": authenticated,
		})
		return
	}

	results, err := s.store.Search(query)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	if !authenticated {
		var filtered []store.SearchResult
		for _, r := range results {
			doc, _ := s.store.Get(r.Path)
			if doc != nil && doc.Public {
				filtered = append(filtered, r)
			}
		}
		results = filtered
	}

	s.renderTemplate(w, "search.html", map[string]any{
		"Query":         query,
		"Results":       results,
		"Authenticated": authenticated,
	})
}

type Breadcrumb struct {
	Name string
	Path string
}

func buildBreadcrumbs(path string) []Breadcrumb {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	crumbs := []Breadcrumb{{Name: "home", Path: "/"}}
	for i, part := range parts {
		crumbs = append(crumbs, Breadcrumb{
			Name: part,
			Path: "/" + strings.Join(parts[:i+1], "/"),
		})
	}
	return crumbs
}

type DirEntry struct {
	Name  string
	Path  string
	IsDir bool
	Title string
	Tags  []string
	Date  string
}

func buildDirectoryEntries(docs []store.Document, prefix string, authenticated bool) []DirEntry {
	seen := make(map[string]bool)
	var entries []DirEntry

	for _, d := range docs {
		if !authenticated && !d.Public {
			continue
		}

		rel := strings.TrimPrefix(d.Path, prefix)
		parts := strings.SplitN(rel, "/", 2)

		if len(parts) > 1 {
			dirName := parts[0]
			if !seen[dirName] {
				seen[dirName] = true
				entries = append(entries, DirEntry{
					Name:  dirName + "/",
					Path:  "/" + prefix + dirName,
					IsDir: true,
				})
			}
		} else {
			name := strings.TrimSuffix(parts[0], ".md")
			entries = append(entries, DirEntry{
				Name:  name,
				Path:  "/" + strings.TrimSuffix(d.Path, ".md"),
				Title: d.Title,
				Tags:  d.Tags,
				Date:  d.UpdatedAt.Format("2006-01-02"),
			})
		}
	}

	return entries
}
