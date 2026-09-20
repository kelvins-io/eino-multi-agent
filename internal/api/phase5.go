package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kelvins-io/eino-multi-agent/internal/eval"
	"github.com/kelvins-io/eino-multi-agent/internal/store"
)

func (s *Server) requireAuth() gin.HandlerFunc {
	token := strings.TrimSpace(s.cfg.Server.AuthToken)
	return func(c *gin.Context) {
		if token == "" || !strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.Next()
			return
		}
		if c.Request.URL.Path == "/api/v1/health" || c.Request.URL.Path == "/api/v1/hooks/echo" {
			c.Next()
			return
		}
		got := bearerToken(c.GetHeader("Authorization"))
		if got == "" {
			got = c.Query("token")
		}
		if got != token {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Next()
	}
}

func bearerToken(h string) string {
	const p = "Bearer "
	if strings.HasPrefix(h, p) {
		return strings.TrimSpace(h[len(p):])
	}
	return ""
}

func operatorOf(c *gin.Context) string {
	if v := strings.TrimSpace(c.GetHeader("X-Operator")); v != "" {
		return v
	}
	return "local"
}

func (s *Server) audit(c *gin.Context, action, targetType, targetID, detail string) {
	_ = s.store.CreateAudit(c.Request.Context(), &store.AuditLog{
		Actor:      operatorOf(c),
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Detail:     detail,
	})
}

func (s *Server) listAudit(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	items, err := s.store.ListAudit(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (s *Server) getEval(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "200"))
	report, err := eval.Run(c.Request.Context(), s.store, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, report)
}
