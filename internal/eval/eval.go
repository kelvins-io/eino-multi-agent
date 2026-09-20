package eval

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/kelvins-io/eino-multi-agent/internal/store"
)

type Case struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Skill       string   `json:"skill"`
	Fixture     string   `json:"fixture,omitempty"`
	ExpectFiles []string `json:"expect_files"`
}

type TaskScore struct {
	CaseID        string `json:"case_id"`
	TaskID        string `json:"task_id"`
	Title         string `json:"title"`
	Status        string `json:"status"`
	Passed        bool   `json:"passed"`
	Reason        string `json:"reason"`
	Confirms      int    `json:"confirms"`
	DurationMS    int64  `json:"duration_ms,omitempty"`
	ArtifactCount int    `json:"artifact_count"`
}

type CaseSummary struct {
	Case       Case        `json:"case"`
	Total      int         `json:"total"`
	Passed     int         `json:"passed"`
	Failed     int         `json:"failed"`
	Confirms   int         `json:"confirms"`
	Completion float64     `json:"completion"`
	Items      []TaskScore `json:"items"`
}

type Report struct {
	Cases      []CaseSummary `json:"cases"`
	Total      int           `json:"total"`
	Passed     int           `json:"passed"`
	Completion float64       `json:"completion"`
	Confirms   int           `json:"confirms"`
}

func Catalog() []Case {
	return []Case{
		{
			ID:          "weekly-report",
			Name:        "CSV 销售周报",
			Skill:       "weekly-report",
			Fixture:     "testdata/sample/sales.csv",
			ExpectFiles: []string{"weekly-report.md"},
		},
		{
			ID:          "research",
			Name:        "公开资料调研",
			Skill:       "research",
			ExpectFiles: []string{"research-report.md"},
		},
		{
			ID:          "file-organize",
			Name:        "文件整理归档",
			Skill:       "file-organize",
			ExpectFiles: []string{"file-index.md"},
		},
	}
}

func Run(ctx context.Context, st *store.Store, limit int, userID string) (*Report, error) {
	tasks, err := st.ListTasks(ctx, limit, "", userID)
	if err != nil {
		return nil, err
	}
	report := &Report{Cases: make([]CaseSummary, 0, len(Catalog()))}
	for _, c := range Catalog() {
		sum := CaseSummary{Case: c}
		for _, task := range tasks {
			if !hasSkill(task.Skills, c.Skill) {
				continue
			}
			events, _ := st.ListEvents(ctx, task.ID, 0)
			arts, _ := st.ListArtifacts(ctx, task.ID)
			score := Score(c, task, events, arts)
			sum.Items = append(sum.Items, score)
			sum.Total++
			sum.Confirms += score.Confirms
			if score.Passed {
				sum.Passed++
			} else {
				sum.Failed++
			}
		}
		if sum.Total > 0 {
			sum.Completion = float64(sum.Passed) / float64(sum.Total)
		}
		report.Cases = append(report.Cases, sum)
		report.Total += sum.Total
		report.Passed += sum.Passed
		report.Confirms += sum.Confirms
	}
	if report.Total > 0 {
		report.Completion = float64(report.Passed) / float64(report.Total)
	}
	return report, nil
}

func Score(c Case, task store.Task, events []store.TaskEvent, arts []store.Artifact) TaskScore {
	out := TaskScore{
		CaseID:        c.ID,
		TaskID:        task.ID,
		Title:         task.Title,
		Status:        task.Status,
		ArtifactCount: len(arts),
	}
	for _, ev := range events {
		if ev.Type == "interrupt" {
			out.Confirms++
		}
	}
	if task.StartedAt != nil && task.FinishedAt != nil {
		out.DurationMS = task.FinishedAt.Sub(*task.StartedAt).Milliseconds()
		if out.DurationMS < 0 {
			out.DurationMS = 0
		}
	}
	if task.Status != store.StatusSucceeded {
		out.Reason = "任务未成功完成"
		return out
	}
	if !hasExpected(arts, c.ExpectFiles) {
		out.Reason = "缺少预期产物 " + strings.Join(c.ExpectFiles, ", ")
		return out
	}
	out.Passed = true
	out.Reason = "通过"
	return out
}

func hasSkill(skills []string, want string) bool {
	for _, s := range skills {
		if s == want {
			return true
		}
	}
	return false
}

func hasExpected(arts []store.Artifact, names []string) bool {
	if len(names) == 0 {
		return len(arts) > 0
	}
	for _, want := range names {
		found := false
		for _, a := range arts {
			base := filepath.Base(a.Name)
			if strings.EqualFold(base, want) || strings.HasSuffix(strings.ToLower(a.RelPath), strings.ToLower(want)) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
