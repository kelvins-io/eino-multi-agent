package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kelvins-io/eino-multi-agent/internal/config"
)

func TestRequireUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := &Server{cfg: &config.Config{}}
	r := gin.New()
	r.GET("/x", func(c *gin.Context) {
		if _, ok := s.requireUser(c); !ok {
			return
		}
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 got %d", w.Code)
	}

	r2 := gin.New()
	r2.GET("/x", func(c *gin.Context) {
		c.Set(ctxUserID, "u1")
		if id, ok := s.requireUser(c); !ok || id != "u1" {
			t.Fatalf("user=%q ok=%v", id, ok)
			return
		}
		c.Status(http.StatusOK)
	})
	w = httptest.NewRecorder()
	r2.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("want 200 got %d", w.Code)
	}
}
