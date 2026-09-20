package api

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kelvins-io/eino-multi-agent/internal/agent"
	"github.com/kelvins-io/eino-multi-agent/internal/config"
	"github.com/kelvins-io/eino-multi-agent/internal/connector"
	"github.com/kelvins-io/eino-multi-agent/internal/harness"
	"github.com/kelvins-io/eino-multi-agent/internal/scheduler"
	"github.com/kelvins-io/eino-multi-agent/internal/store"
	"github.com/kelvins-io/eino-multi-agent/internal/workspace"
)

type Server struct {
	cfg        *config.Config
	store      *store.Store
	runtime    *harness.Runtime
	sched      *scheduler.Scheduler
	connectors *connector.Registry
	engine     *gin.Engine
}

func New(cfg *config.Config, st *store.Store, rt *harness.Runtime, sched *scheduler.Scheduler, connectors *connector.Registry) *Server {
	gin.SetMode(cfg.Server.Mode)
	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger(), cors(cfg.Server.CORSOrigins))
	s := &Server{cfg: cfg, store: st, runtime: rt, sched: sched, connectors: connectors, engine: r}
	r.Use(s.requireAuth())
	s.routes()
	return s
}

func (s *Server) Engine() *gin.Engine { return s.engine }

func (s *Server) routes() {
	api := s.engine.Group("/api/v1")
	api.GET("/health", s.health)
	api.GET("/meta", s.meta)
	api.GET("/tasks", s.listTasks)
	api.POST("/tasks", s.createTask)
	api.GET("/tasks/:id", s.getTask)
	api.POST("/tasks/:id/cancel", s.cancelTask)
	api.POST("/tasks/:id/confirm", s.confirmTask)
	api.POST("/tasks/:id/retry", s.retryTask)
	api.GET("/tasks/:id/events", s.taskEvents)
	api.GET("/tasks/:id/artifacts", s.listArtifacts)
	api.GET("/tasks/:id/artifacts/:aid/preview", s.previewArtifact)
	api.GET("/tasks/:id/artifacts/:aid", s.downloadArtifact)

	api.GET("/projects", s.listProjects)
	api.POST("/projects", s.createProject)
	api.GET("/projects/:id", s.getProject)
	api.POST("/projects/:id/files", s.uploadProjectFiles)

	api.GET("/schedules", s.listSchedules)
	api.POST("/schedules", s.createSchedule)
	api.POST("/schedules/:id/run", s.runSchedule)
	api.POST("/schedules/:id/toggle", s.toggleSchedule)

	api.GET("/connectors", s.listConnectors)
	api.POST("/connectors", s.createConnector)
	api.POST("/connectors/:id/toggle", s.toggleConnector)
	api.POST("/connectors/:id/test", s.testConnector)
	api.POST("/hooks/echo", s.hookEcho)
	api.GET("/audit", s.listAudit)
	api.GET("/eval", s.getEval)

	s.engine.NoRoute(s.spa)
}

func (s *Server) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) hookEcho(c *gin.Context) {
	_, _ = io.Copy(io.Discard, io.LimitReader(c.Request.Body, 1<<20))
	c.Status(http.StatusNoContent)
}

func (s *Server) meta(c *gin.Context) {
	type skillDTO struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	skills := make([]skillDTO, 0)
	for _, sk := range s.runtime.Skills() {
		skills = append(skills, skillDTO{Name: sk.Name, Description: sk.Description})
	}
	c.JSON(http.StatusOK, gin.H{
		"llm": gin.H{
			"provider": s.cfg.LLM.Provider,
			"model":    s.cfg.LLM.Model,
			"ready":    agent.Ready(s.cfg.LLM) == nil,
		},
		"confirm_policies": []string{"on_risk", "always", "never"},
		"skills":           skills,
		"auth_required":    s.cfg.Server.AuthToken != "",
	})
}

func (s *Server) listTasks(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	tasks, err := s.store.ListTasks(c.Request.Context(), limit, c.Query("project_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": tasks})
}

func (s *Server) createTask(c *gin.Context) {
	var title, goal, policy, projectID string
	var skills []string
	var uploads []harness.Upload

	ct := c.ContentType()
	if strings.HasPrefix(ct, "multipart/form-data") {
		title = c.PostForm("title")
		goal = c.PostForm("goal")
		policy = c.PostForm("confirm_policy")
		projectID = c.PostForm("project_id")
		if raw := c.PostForm("skills"); raw != "" {
			_ = json.Unmarshal([]byte(raw), &skills)
			if len(skills) == 0 {
				skills = strings.Split(raw, ",")
			}
		}
		form, err := c.MultipartForm()
		if err == nil && form != nil {
			for _, fh := range form.File["files"] {
				f, err := fh.Open()
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				uploads = append(uploads, harness.Upload{Name: fh.Filename, Reader: f})
			}
		}
		defer func() {
			for _, u := range uploads {
				if rc, ok := u.Reader.(io.Closer); ok {
					_ = rc.Close()
				}
			}
		}()
	} else {
		var body struct {
			Title         string   `json:"title"`
			Goal          string   `json:"goal"`
			ConfirmPolicy string   `json:"confirm_policy"`
			Skills        []string `json:"skills"`
			ProjectID     string   `json:"project_id"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		title, goal, policy, skills, projectID = body.Title, body.Goal, body.ConfirmPolicy, body.Skills, body.ProjectID
	}

	task, err := s.runtime.Create(c.Request.Context(), harness.CreateInput{
		Title:         title,
		Goal:          goal,
		ConfirmPolicy: policy,
		Skills:        compact(skills),
		ProjectID:     projectID,
		Files:         uploads,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.audit(c, "task.create", "task", task.ID, task.Title)
	c.JSON(http.StatusCreated, task)
}

func (s *Server) getTask(c *gin.Context) {
	task, err := s.store.GetTask(c.Request.Context(), c.Param("id"))
	if err != nil {
		status := http.StatusInternalServerError
		if harness.IsNotFound(err) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	events, _ := s.store.ListEvents(c.Request.Context(), task.ID, 0)
	artifacts, _ := s.store.ListArtifacts(c.Request.Context(), task.ID)
	c.JSON(http.StatusOK, gin.H{"task": task, "events": events, "artifacts": artifacts})
}

func (s *Server) cancelTask(c *gin.Context) {
	if err := s.runtime.Cancel(c.Request.Context(), c.Param("id")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.audit(c, "task.cancel", "task", c.Param("id"), "")
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) retryTask(c *gin.Context) {
	task, err := s.runtime.Retry(c.Request.Context(), c.Param("id"))
	if err != nil {
		status := http.StatusBadRequest
		if harness.IsNotFound(err) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	s.audit(c, "task.retry", "task", task.ID, "")
	c.JSON(http.StatusOK, task)
}

func (s *Server) confirmTask(c *gin.Context) {
	var body struct {
		Approved bool `json:"approved"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := s.runtime.Confirm(c.Request.Context(), c.Param("id"), body.Approved); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	detail := "rejected"
	if body.Approved {
		detail = "approved"
	}
	s.audit(c, "task.confirm", "task", c.Param("id"), detail)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) taskEvents(c *gin.Context) {
	id := c.Param("id")
	afterID, _ := strconv.ParseUint(c.DefaultQuery("after", "0"), 10, 64)
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Flush()

	history, err := s.store.ListEvents(c.Request.Context(), id, afterID)
	if err != nil {
		c.SSEvent("error", err.Error())
		return
	}
	for _, ev := range history {
		writeSSE(c, ev)
		if ev.ID > afterID {
			afterID = ev.ID
		}
	}
	ch, unsub := s.runtime.Bus().Subscribe(id)
	defer unsub()
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
			c.Writer.Write([]byte(": ping\n\n"))
			c.Writer.Flush()
		case ev, ok := <-ch:
			if !ok {
				return
			}
			if ev.ID <= afterID {
				continue
			}
			writeSSE(c, store.TaskEvent{
				ID:      ev.ID,
				TaskID:  ev.TaskID,
				Type:    ev.Type,
				Agent:   ev.Agent,
				Message: ev.Message,
				Payload: ev.Payload,
			})
		}
	}
}

func (s *Server) listArtifacts(c *gin.Context) {
	items, err := s.store.ListArtifacts(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (s *Server) downloadArtifact(c *gin.Context) {
	task, err := s.store.GetTask(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	aid, _ := strconv.ParseUint(c.Param("aid"), 10, 64)
	item, err := s.store.GetArtifact(c.Request.Context(), task.ID, aid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	sb, err := workspace.Open(task.Workspace)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	abs, err := sb.ReadArtifact(item.RelPath)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	if c.Query("inline") == "1" {
		c.File(abs)
		return
	}
	c.FileAttachment(abs, item.Name)
}

func (s *Server) previewArtifact(c *gin.Context) {
	task, err := s.store.GetTask(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	aid, _ := strconv.ParseUint(c.Param("aid"), 10, 64)
	item, err := s.store.GetArtifact(c.Request.Context(), task.ID, aid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	if !harness.Previewable(item.Name, item.Mime) {
		c.JSON(http.StatusOK, gin.H{"previewable": false, "name": item.Name, "mime": item.Mime})
		return
	}
	sb, err := workspace.Open(task.Workspace)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	abs, err := sb.ReadArtifact(item.RelPath)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	if strings.HasPrefix(item.Mime, "image/") || isImageName(item.Name) {
		c.JSON(http.StatusOK, gin.H{
			"previewable": true,
			"kind":        "image",
			"name":        item.Name,
			"mime":        item.Mime,
			"url":         "/api/v1/tasks/" + task.ID + "/artifacts/" + strconv.FormatUint(item.ID, 10) + "?inline=1",
		})
		return
	}
	raw, err := os.ReadFile(abs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	text := string(raw)
	truncated := false
	if len([]rune(text)) > 20000 {
		text = string([]rune(text)[:20000])
		truncated = true
	}
	c.JSON(http.StatusOK, gin.H{
		"previewable": true,
		"kind":        "text",
		"name":        item.Name,
		"mime":        item.Mime,
		"text":        text,
		"truncated":   truncated,
	})
}

func isImageName(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg":
		return true
	}
	return false
}

func (s *Server) spa(c *gin.Context) {
	if strings.HasPrefix(c.Request.URL.Path, "/api/") {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	dist := filepath.Join("web", "dist")
	target := filepath.Join(dist, c.Request.URL.Path)
	if info, err := os.Stat(target); err == nil && !info.IsDir() {
		c.File(target)
		return
	}
	index := filepath.Join(dist, "index.html")
	if _, err := os.Stat(index); err == nil {
		c.File(index)
		return
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "frontend is not built; run npm install && npm run build in ./web"})
}

func writeSSE(c *gin.Context, ev store.TaskEvent) {
	b, _ := json.Marshal(ev)
	c.Writer.Write([]byte("id: " + strconv.FormatUint(ev.ID, 10) + "\n"))
	c.Writer.Write([]byte("event: task\n"))
	c.Writer.Write([]byte("data: " + string(b) + "\n\n"))
	c.Writer.Flush()
}

func cors(origins []string) gin.HandlerFunc {
	allow := map[string]struct{}{}
	for _, o := range origins {
		allow[o] = struct{}{}
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if _, ok := allow[origin]; ok || len(allow) == 0 {
			if origin == "" {
				origin = "*"
			}
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func compact(in []string) []string {
	var out []string
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}
