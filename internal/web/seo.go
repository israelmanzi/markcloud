package web

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type urlSet struct {
	XMLName xml.Name     `xml:"urlset"`
	XMLNS   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

type sitemapURL struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod"`
}

func (s *Server) handleSitemap(w http.ResponseWriter, r *http.Request) {
	docs, err := s.store.List("")
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	baseURL := "https://" + r.Host

	var urls []sitemapURL
	for _, d := range docs {
		if !d.Public {
			continue
		}
		path := strings.TrimSuffix(d.Path, ".md")
		urls = append(urls, sitemapURL{
			Loc:     baseURL + "/" + path,
			LastMod: d.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}

	set := urlSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  urls,
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Write([]byte(xml.Header))
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	enc.Encode(set)
}

func (s *Server) handleRobots(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "User-agent: *\nAllow: /\nDisallow: /login\nDisallow: /logout\nDisallow: /api/\nSitemap: https://%s/sitemap.xml\n", r.Host)
}
