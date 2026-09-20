package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kelvins-io/eino-multi-agent/internal/eval"
	"github.com/kelvins-io/eino-multi-agent/internal/store"
)

func (s *Server) audit(c *gin.Context, action, targetType, targetID, detail string) {
	_ = s.store.CreateAudit(c.Request.Context(), &store.AuditLog{
		UserID:     c.GetString(ctxUserID),
		Actor:      operatorOf(c),
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Detail:     detail,
	})
}

func (s *Server) listAudit(c *gin.Context) {
	userID, ok := s.requireUser(c)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	items, err := s.store.ListAudit(c.Request.Context(), limit, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (s *Server) getEval(c *gin.Context) {
	userID, ok := s.requireUser(c)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "200"))
	report, err := eval.Run(c.Request.Context(), s.store, limit, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, report)
}
