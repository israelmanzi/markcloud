package web

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/israelmanzi/markcloud/internal/store"
	"github.com/israelmanzi/markcloud/internal/sync"
)

var funcMap = template.FuncMap{
	"humanDate": func(t time.Time) string {
		now := time.Now()
		diff := now.Sub(t)

		switch {
		case diff < time.Minute:
			return "just now"
		case diff < time.Hour:
			m := int(diff.Minutes())
			if m == 1 {
				return "1 min ago"
			}
			return fmt.Sprintf("%d mins ago", m)
		case diff < 24*time.Hour:
			h := int(diff.Hours())
			if h == 1 {
				return "1 hour ago"
			}
			return fmt.Sprintf("%d hours ago", h)
		case diff < 7*24*time.Hour:
			d := int(diff.Hours() / 24)
			if d == 1 {
				return "yesterday"
			}
			return fmt.Sprintf("%d days ago", d)
		default:
			return t.Format("Jan 2, 2006")
		}
	},
	"trimSuffix": func(suffix, s string) string {
		return strings.TrimSuffix(s, suffix)
	},
}

type Server struct {
	store        *store.Store
	syncHandler  *sync.Handler
	templates    map[string]*template.Template
	apiKey       string
	deploySecret string
}

type Config struct {
	Store        *store.Store
	APIKey       string
	DeploySecret string
	TemplatesDir string
}

func NewServer(cfg Config) *Server {
	templates := make(map[string]*template.Template)
	layout := filepath.Join(cfg.TemplatesDir, "layout.html")

	pages := []string{"login.html", "directory.html", "document.html", "404.html"}
	for _, page := range pages {
		templates[page] = template.Must(
			template.New("").Funcs(funcMap).ParseFiles(layout, filepath.Join(cfg.TemplatesDir, page)),
		)
	}

	return &Server{
		store:        cfg.Store,
		syncHandler:  sync.NewHandler(cfg.Store),
		templates:    templates,
		apiKey:       cfg.APIKey,
		deploySecret: cfg.DeploySecret,
	}
}

func (s *Server) renderTemplate(w http.ResponseWriter, name string, data any) {
	if s.templates == nil {
		return
	}
	tmpl, ok := s.templates[name]
	if !ok {
		log.Printf("template not found: %s", name)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		log.Printf("template error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/login", s.handleLogin)
	mux.HandleFunc("/logout", s.handleLogout)
	mux.HandleFunc("/api/sync/manifest", s.handleSyncManifest)
	mux.HandleFunc("/api/sync/upload", s.handleSyncUpload)
	mux.HandleFunc("/feed.xml", s.handleFeed)
	mux.HandleFunc("/sitemap.xml", s.handleSitemap)
	mux.HandleFunc("/robots.txt", s.handleRobots)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("/", s.handlePage)

	return mux
}
