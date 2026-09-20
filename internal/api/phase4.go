package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kelvins-io/eino-multi-agent/internal/confirm"
	"github.com/kelvins-io/eino-multi-agent/internal/connector"
	"github.com/kelvins-io/eino-multi-agent/internal/scheduler"
	"github.com/kelvins-io/eino-multi-agent/internal/store"
	"github.com/kelvins-io/eino-multi-agent/internal/workspace"
)

func (s *Server) listProjects(c *gin.Context) {
	items, err := s.store.ListProjects(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (s *Server) createProject(c *gin.Context) {
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	id := uuid.NewString()
	root, err := workspace.NewProject(s.cfg.Workspace.Root, id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p := &store.Project{ID: id, Name: body.Name, Description: body.Description, Workspace: root}
	if err := s.store.CreateProject(c.Request.Context(), p); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.audit(c, "project.create", "project", p.ID, p.Name)
	c.JSON(http.StatusCreated, p)
}

func (s *Server) getProject(c *gin.Context) {
	p, err := s.store.GetProject(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	files, _ := workspace.ListShared(p.Workspace)
	tasks, _ := s.store.ListTasks(c.Request.Context(), 50, p.ID)
	c.JSON(http.StatusOK, gin.H{"project": p, "files": files, "tasks": tasks})
}

func (s *Server) uploadProjectFiles(c *gin.Context) {
	p, err := s.store.GetProject(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	for _, fh := range form.File["files"] {
		f, err := fh.Open()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		_, err = workspace.SaveShared(p.Workspace, fh.Filename, f)
		_ = f.Close()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	files, _ := workspace.ListShared(p.Workspace)
	c.JSON(http.StatusOK, gin.H{"files": files})
}

func (s *Server) listSchedules(c *gin.Context) {
	items, err := s.store.ListSchedules(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (s *Server) createSchedule(c *gin.Context) {
	var body struct {
		ProjectID     string   `json:"project_id"`
		Title         string   `json:"title"`
		Goal          string   `json:"goal"`
		Skills        []string `json:"skills"`
		ConfirmPolicy string   `json:"confirm_policy"`
		Kind          string   `json:"kind"`
		Weekday       int      `json:"weekday"`
		Hour          int      `json:"hour"`
		Minute        int      `json:"minute"`
		RunAt         string   `json:"run_at"`
		Timezone      string   `json:"timezone"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Goal == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "goal is required"})
		return
	}
	if body.Timezone == "" {
		body.Timezone = "Asia/Shanghai"
	}
	loc := scheduler.LoadLocation(body.Timezone)
	var runAt *time.Time
	if body.Kind == scheduler.KindOnce && body.RunAt != "" {
		t, err := time.ParseInLocation(time.RFC3339, body.RunAt, loc)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "run_at must be RFC3339"})
			return
		}
		runAt = &t
	}
	next, err := scheduler.NextRun(time.Now(), loc, body.Kind, body.Weekday, body.Hour, body.Minute, runAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	title := body.Title
	if title == "" {
		title = body.Goal
		if len([]rune(title)) > 40 {
			title = string([]rune(title)[:40])
		}
	}
	item := &store.Schedule{
		ID:            uuid.NewString(),
		ProjectID:     body.ProjectID,
		Title:         title,
		Goal:          body.Goal,
		Skills:        compact(body.Skills),
		ConfirmPolicy: confirm.Normalize(body.ConfirmPolicy),
		Kind:          body.Kind,
		Weekday:       body.Weekday,
		Hour:          body.Hour,
		Minute:        body.Minute,
		RunAt:         runAt,
		Timezone:      body.Timezone,
		Enabled:       true,
		NextRunAt:     &next,
	}
	if err := s.store.CreateSchedule(c.Request.Context(), item); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.audit(c, "schedule.create", "schedule", item.ID, item.Title)
	c.JSON(http.StatusCreated, item)
}

func (s *Server) runSchedule(c *gin.Context) {
	if s.sched == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "scheduler not ready"})
		return
	}
	task, err := s.sched.FireNow(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.audit(c, "schedule.run", "schedule", c.Param("id"), task.ID)
	c.JSON(http.StatusOK, task)
}

func (s *Server) toggleSchedule(c *gin.Context) {
	item, err := s.store.GetSchedule(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	item.Enabled = !item.Enabled
	if err := s.store.SaveSchedule(c.Request.Context(), item); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (s *Server) listConnectors(c *gin.Context) {
	items, err := s.store.ListConnectors(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (s *Server) createConnector(c *gin.Context) {
	var body struct {
		Name   string            `json:"name"`
		Kind   string            `json:"kind"`
		Config map[string]string `json:"config"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	if err := connector.ValidateKind(body.Kind); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.Config == nil {
		body.Config = map[string]string{}
	}
	item := &store.Connector{
		ID:      uuid.NewString(),
		Name:    body.Name,
		Kind:    body.Kind,
		Config:  body.Config,
		Enabled: true,
	}
	if err := s.store.CreateConnector(c.Request.Context(), item); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.audit(c, "connector.create", "connector", item.ID, item.Name)
	c.JSON(http.StatusCreated, item)
}

func (s *Server) toggleConnector(c *gin.Context) {
	item, err := s.store.GetConnector(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	item.Enabled = !item.Enabled
	if err := s.store.SaveConnector(c.Request.Context(), item); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (s *Server) testConnector(c *gin.Context) {
	item, err := s.store.GetConnector(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	if s.connectors == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "connectors not ready"})
		return
	}
	ev := connector.Event{Task: &store.Task{ID: "test", Title: "连接器测试", Status: store.StatusSucceeded, Workspace: s.cfg.Workspace.Root}}
	if err := s.connectors.Invoke(c.Request.Context(), item, ev); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.audit(c, "connector.test", "connector", item.ID, item.Name)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
