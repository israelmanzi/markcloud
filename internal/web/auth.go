package web

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"
)

const sessionDuration = 7 * 24 * time.Hour

func (s *Server) generateSession() string {
	b := make([]byte, 32)
	rand.Read(b)
	mac := hmac.New(sha256.New, []byte(s.apiKey))
	mac.Write(b)
	token := hex.EncodeToString(mac.Sum(nil))

	s.store.CreateSession(token, time.Now().Add(sessionDuration))
	return token
}

func (s *Server) isAuthenticated(r *http.Request) bool {
	cookie, err := r.Cookie("session")
	if err != nil {
		return false
	}
	return s.store.GetSession(cookie.Value)
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
		MaxAge:   int(sessionDuration.Seconds()),
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err == nil {
		s.store.DeleteSession(cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:   "session",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
