package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kelvins-io/eino-multi-agent/internal/config"
	"github.com/kelvins-io/eino-multi-agent/internal/store"
)

func TestBearerToken(t *testing.T) {
	if bearerToken("Bearer secret") != "secret" {
		t.Fatal("bearer")
	}
	if bearerToken("secret") != "" {
		t.Fatal("non-bearer should be empty")
	}
}

func TestNormalizeAuth(t *testing.T) {
	if _, _, err := normalizeAuth("ab", "123456"); err == nil {
		t.Fatal("short username")
	}
	if _, _, err := normalizeAuth("alice", "123"); err == nil {
		t.Fatal("short password")
	}
	if _, _, err := normalizeAuth("alice!", "123456"); err == nil {
		t.Fatal("invalid username chars")
	}
	u, p, err := normalizeAuth(" Alice ", " 123456 ")
	if err != nil || u != "Alice" || p != "123456" {
		t.Fatalf("got %q %q %v", u, p, err)
	}
}

func TestNormalizeDisplayName(t *testing.T) {
	if _, err := normalizeDisplayName("  "); err == nil {
		t.Fatal("empty")
	}
	got, err := normalizeDisplayName(" 小明 ")
	if err != nil || got != "小明" {
		t.Fatalf("got %q %v", got, err)
	}
}

func TestIssueAndParseToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := &Server{cfg: &config.Config{Server: config.ServerConfig{
		JWTSecret: "test-secret",
		JWTExpire: time.Hour,
	}}}
	user := &store.User{ID: "uid-1", Username: "alice"}
	token, exp, err := s.issueToken(user)
	if err != nil || token == "" || exp.Before(time.Now()) {
		t.Fatalf("issue: %v", err)
	}
	claims, err := s.parseToken(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != "uid-1" || claims.Username != "alice" {
		t.Fatalf("claims: %+v", claims)
	}
}

func TestRequireAuthJWT(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := &Server{cfg: &config.Config{Server: config.ServerConfig{
		JWTSecret: "test-secret",
		JWTExpire: time.Hour,
	}}}
	r := gin.New()
	r.Use(s.requireAuth())
	r.GET("/api/v1/meta", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"user": c.GetString(ctxUsername)})
	})
	r.POST("/api/v1/auth/login", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/meta", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{}`))
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("login should be public, got %d", w.Code)
	}

	token, _, err := s.issueToken(&store.User{ID: "u1", Username: "bob"})
	if err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/meta", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "bob") {
		t.Fatalf("body=%s", w.Body.String())
	}
}
