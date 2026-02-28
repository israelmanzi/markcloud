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
		"BaseURL":       "https://" + r.Host,
	}
}

// BreadcrumbItem represents a single breadcrumb link.
type BreadcrumbItem struct {
	Name string
	Href string
	Last bool
}

func buildBreadcrumbs(urlPath string) []BreadcrumbItem {
	path := strings.Trim(urlPath, "/")
	if path == "" {
		return nil
	}

	parts := strings.Split(path, "/")
	crumbs := make([]BreadcrumbItem, 0, len(parts)+1)
	crumbs = append(crumbs, BreadcrumbItem{Name: "home", Href: "/"})

	href := ""
	for i, part := range parts {
		href += "/" + part
		crumbs = append(crumbs, BreadcrumbItem{
			Name: part,
			Href: href,
			Last: i == len(parts)-1,
		})
	}
	return crumbs
}

func (s *Server) handlePage(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	data := s.baseData(r)
	authenticated := data["Authenticated"].(bool)
	query := data["Query"].(string)

	// Tag filter: /?tag=X
	if tag := r.URL.Query().Get("tag"); tag != "" {
		s.handleTagFilter(w, r, tag, data)
		return
	}

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
		data["TOC"] = template.HTML(doc.TOCHTML)
		data["IsDocument"] = true
		data["OGTitle"] = doc.Title
		data["OGDescription"] = doc.Description
		data["DateModified"] = doc.UpdatedAt.UTC().Format(time.RFC3339)
		data["Breadcrumbs"] = buildBreadcrumbs(r.URL.Path)

		// Backlinks
		backlinks, err := s.store.GetBacklinks(path + ".md")
		if err == nil && len(backlinks) > 0 {
			var filtered []store.Document
			for _, bl := range backlinks {
				if bl.Public || authenticated {
					filtered = append(filtered, bl)
				}
			}
			if len(filtered) > 0 {
				data["Backlinks"] = filtered
			}
		}

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

	data["Breadcrumbs"] = buildBreadcrumbs(r.URL.Path)
	s.renderListing(w, prefix, docs, data)
}

func (s *Server) handleListing(w http.ResponseWriter, r *http.Request, prefix string, data map[string]any) {
	docs, err := s.store.List(prefix)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	s.renderListing(w, prefix, docs, data)
}

func (s *Server) renderListing(w http.ResponseWriter, prefix string, docs []store.Document, data map[string]any) {
	authenticated := data["Authenticated"].(bool)
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

	var entries []DirEntry
	for _, r := range results {
		doc, _ := s.store.Get(r.Path)
		if doc == nil {
			continue
		}
		if !authenticated && !doc.Public {
			continue
		}
		path := strings.TrimSuffix(r.Path, ".md")
		entries = append(entries, DirEntry{
			Name:    r.Title,
			Path:    "/" + path,
			Snippet: cleanSnippet(r.Snippet),
			Public:  doc.Public,
			Tags:    doc.Tags,
			Date:    doc.UpdatedAt,
		})
	}

	data["Path"] = ""
	data["Entries"] = entries
	data["SearchResults"] = true
	s.renderTemplate(w, "directory.html", data)
}

func (s *Server) handleTagFilter(w http.ResponseWriter, r *http.Request, tag string, data map[string]any) {
	authenticated := data["Authenticated"].(bool)

	docs, err := s.store.ListByTag(tag)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	var entries []DirEntry
	for _, d := range docs {
		if !authenticated && !d.Public {
			continue
		}
		path := strings.TrimSuffix(d.Path, ".md")
		entries = append(entries, DirEntry{
			Name:   d.Title,
			Path:   "/" + path,
			Tags:   d.Tags,
			Date:   d.UpdatedAt,
			Public: d.Public,
		})
	}

	data["Path"] = ""
	data["Entries"] = entries
	data["TagFilter"] = tag
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
