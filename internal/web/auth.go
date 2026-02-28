package web

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

type session struct {
	token     string
	expiresAt time.Time
}

var (
	sessions   = make(map[string]session)
	sessionsMu sync.RWMutex
)

func (s *Server) generateSession() string {
	b := make([]byte, 32)
	rand.Read(b)
	mac := hmac.New(sha256.New, []byte(s.apiKey))
	mac.Write(b)
	token := hex.EncodeToString(mac.Sum(nil))

	sessionsMu.Lock()
	sessions[token] = session{token: token, expiresAt: time.Now().Add(7 * 24 * time.Hour)}
	sessionsMu.Unlock()

	return token
}

func (s *Server) isAuthenticated(r *http.Request) bool {
	cookie, err := r.Cookie("session")
	if err != nil {
		return false
	}

	sessionsMu.RLock()
	sess, ok := sessions[cookie.Value]
	sessionsMu.RUnlock()

	return ok && time.Now().Before(sess.expiresAt)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		s.renderTemplate(w, "login.html", nil)
		return
	}

	key := r.FormValue("api_key")
	if key != s.apiKey {
		w.WriteHeader(http.StatusUnauthorized)
		s.renderTemplate(w, "login.html", map[string]any{"Error": "Invalid API key"})
		return
	}

	token := s.generateSession()
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   7 * 24 * 60 * 60,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err == nil {
		sessionsMu.Lock()
		delete(sessions, cookie.Value)
		sessionsMu.Unlock()
	}

	http.SetCookie(w, &http.Cookie{
		Name:   "session",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
