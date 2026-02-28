package web

import (
	"html/template"
	"log"
	"net/http"
	"path/filepath"

	"github.com/israelmanzi/markcloud/internal/store"
	"github.com/israelmanzi/markcloud/internal/sync"
)

type Server struct {
	store        *store.Store
	syncHandler  *sync.Handler
	templates    *template.Template
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
	tmpl := template.Must(template.ParseGlob(filepath.Join(cfg.TemplatesDir, "*.html")))

	return &Server{
		store:        cfg.Store,
		syncHandler:  sync.NewHandler(cfg.Store),
		templates:    tmpl,
		apiKey:       cfg.APIKey,
		deploySecret: cfg.DeploySecret,
	}
}

func (s *Server) renderTemplate(w http.ResponseWriter, name string, data any) {
	if s.templates == nil {
		return
	}
	if err := s.templates.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("template error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/login", s.handleLogin)
	mux.HandleFunc("/logout", s.handleLogout)
	mux.HandleFunc("/search", s.handleSearch)
	mux.HandleFunc("/public", s.handlePublicIndex)
	mux.HandleFunc("/api/sync/manifest", s.handleSyncManifest)
	mux.HandleFunc("/api/sync/upload", s.handleSyncUpload)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("/", s.handlePage)

	return mux
}
