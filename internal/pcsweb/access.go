package pcsweb

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"
)

const accessCookieName = "pcsweb_access"

type accessState struct {
	mu       sync.RWMutex
	sessions map[string]time.Time
}

func newAccessState() *accessState {
	return &accessState{sessions: make(map[string]time.Time)}
}

func (s *Server) accessRequired() bool {
	return strings.TrimSpace(s.accessPassword) != ""
}

func (s *Server) accessAuthenticated(r *http.Request) bool {
	if !s.accessRequired() {
		return true
	}
	cookie, err := r.Cookie(accessCookieName)
	if err != nil || cookie.Value == "" {
		return false
	}

	s.access.mu.RLock()
	expiresAt, ok := s.access.sessions[cookie.Value]
	s.access.mu.RUnlock()
	if !ok {
		return false
	}
	if time.Now().Before(expiresAt) {
		return true
	}

	s.access.mu.Lock()
	delete(s.access.sessions, cookie.Value)
	s.access.mu.Unlock()
	return false
}

func newAccessToken() string {
	data := make([]byte, 32)
	if _, err := rand.Read(data); err == nil {
		return hex.EncodeToString(data)
	}
	return hex.EncodeToString([]byte(time.Now().Format("20060102150405.000000000")))
}

func (s *Server) issueAccessCookie(w http.ResponseWriter) {
	token := newAccessToken()
	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	s.access.mu.Lock()
	s.access.sessions[token] = expiresAt
	s.access.mu.Unlock()
	http.SetCookie(w, &http.Cookie{
		Name:     accessCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   30 * 24 * 60 * 60,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *Server) handleAccessStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{
		"required":      s.accessRequired(),
		"authenticated": s.accessAuthenticated(r),
	})
}

func (s *Server) handleAccessLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if !s.accessRequired() {
		writeJSON(w, http.StatusOK, map[string]bool{"authenticated": true})
		return
	}
	var request struct {
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	if subtle.ConstantTimeCompare([]byte(request.Password), []byte(s.accessPassword)) != 1 {
		writeError(w, http.StatusUnauthorized, errors.New("访问密码错误"))
		return
	}
	s.issueAccessCookie(w)
	writeJSON(w, http.StatusOK, map[string]bool{"authenticated": true})
}

func (s *Server) withAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.accessRequired() || r.URL.Path == "/api/access/status" || r.URL.Path == "/api/access/login" || s.accessAuthenticated(r) || !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		writeError(w, http.StatusUnauthorized, errors.New("需要先输入 Web 访问密码"))
	})
}
