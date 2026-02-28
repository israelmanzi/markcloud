package web

import (
	"html/template"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/israelmanzi/markcloud/internal/store"
)

var mdSyntaxRegex = regexp.MustCompile(`(?:^#{1,6}\s+)|(?:\*{1,3})|(?:_{1,3})|(?:` + "`" + `+)|(?:^\s*[-*+]\s)|(?:^\s*\d+\.\s)|(?:^\s*>+\s?)`)

func cleanSnippet(s string) template.HTML {
	cleaned := mdSyntaxRegex.ReplaceAllString(s, "")
	cleaned = strings.ReplaceAll(cleaned, "\n", " ")
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	return template.HTML(cleaned)
}

func (s *Server) notFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	s.renderTemplate(w, "404.html", s.baseData(r))
}

func (s *Server) baseData(r *http.Request) map[string]any {
	return map[string]any{
		"Authenticated": s.isAuthenticated(r),
		"Query":         r.URL.Query().Get("q"),
		"CurrentPath":   r.URL.Path,
	}
}

func (s *Server) handlePage(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	data := s.baseData(r)
	authenticated := data["Authenticated"].(bool)
	query := data["Query"].(string)

	// Search: /?q=... or /notes/?q=...
	if query != "" {
		s.handleSearch(w, r, data)
		return
	}

	// Root listing
	if path == "" {
		s.handleListing(w, r, "", data)
		return
	}

	// Try as document
	doc, err := s.store.Get(path + ".md")
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	if doc != nil {
		if !doc.Public && !authenticated {
			s.notFound(w, r)
			return
		}
		data["Doc"] = doc
		data["Content"] = template.HTML(doc.ContentHTML)
		s.renderTemplate(w, "document.html", data)
		return
	}

	// Try as directory
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
		s.notFound(w, r)
		return
	}

	s.handleListing(w, r, prefix, data)
}

func (s *Server) handleListing(w http.ResponseWriter, r *http.Request, prefix string, data map[string]any) {
	authenticated := data["Authenticated"].(bool)

	docs, err := s.store.List(prefix)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	entries := buildDirectoryEntries(docs, prefix, authenticated)

	data["Path"] = strings.TrimSuffix(prefix, "/")
	data["Entries"] = entries
	s.renderTemplate(w, "directory.html", data)
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request, data map[string]any) {
	query := data["Query"].(string)
	authenticated := data["Authenticated"].(bool)

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

	// Convert search results to dir entries for the same template
	var entries []DirEntry
	for _, r := range results {
		path := strings.TrimSuffix(r.Path, ".md")
		entry := DirEntry{
			Name:    r.Title,
			Path:    "/" + path,
			Snippet: cleanSnippet(r.Snippet),
		}
		if doc, _ := s.store.Get(r.Path); doc != nil {
			entry.Public = doc.Public
			entry.Tags = doc.Tags
			entry.Date = doc.UpdatedAt
		}
		entries = append(entries, entry)
	}

	data["Path"] = ""
	data["Entries"] = entries
	data["SearchResults"] = true
	s.renderTemplate(w, "directory.html", data)
}

type DirEntry struct {
	Name    string
	Path    string
	IsDir   bool
	Title   string
	Tags    []string
	Date    time.Time
	Public  bool
	Snippet template.HTML
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
				Name:   name,
				Path:   "/" + strings.TrimSuffix(d.Path, ".md"),
				Title:  d.Title,
				Tags:   d.Tags,
				Date:   d.UpdatedAt,
				Public: d.Public,
			})
		}
	}

	return entries
}
