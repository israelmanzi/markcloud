package web

import (
	"encoding/xml"
	"net/http"
	"strings"
	"time"
)

type rssChannel struct {
	XMLName       xml.Name  `xml:"channel"`
	Title         string    `xml:"title"`
	Link          string    `xml:"link"`
	Description   string    `xml:"description"`
	LastBuildDate string    `xml:"lastBuildDate"`
	Items         []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description,omitempty"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
}

type rssFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel rssChannel `xml:"channel"`
}

func (s *Server) handleFeed(w http.ResponseWriter, r *http.Request) {
	docs, err := s.store.ListPublic(50)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	scheme := "https"
	host := r.Host
	baseURL := scheme + "://" + host

	var items []rssItem
	for _, d := range docs {
		path := strings.TrimSuffix(d.Path, ".md")
		items = append(items, rssItem{
			Title:       d.Title,
			Link:        baseURL + "/" + path,
			Description: d.Description,
			PubDate:     d.UpdatedAt.UTC().Format(time.RFC1123Z),
			GUID:        baseURL + "/" + path,
		})
	}

	feed := rssFeed{
		Version: "2.0",
		Channel: rssChannel{
			Title:         "markcloud",
			Link:          baseURL,
			Description:   "markcloud documents",
			LastBuildDate: time.Now().UTC().Format(time.RFC1123Z),
			Items:         items,
		},
	}

	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.Write([]byte(xml.Header))
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	enc.Encode(feed)
}
