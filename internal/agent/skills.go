package agent

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Skill struct {
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description" json:"description"`
	Content     string `json:"-"`
}

func LoadSkills(dir string) ([]Skill, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var skills []Skill
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name(), "SKILL.md"))
		if err != nil {
			continue
		}
		skill := parseSkill(e.Name(), string(raw))
		skills = append(skills, skill)
	}
	return skills, nil
}

func parseSkill(fallback string, raw string) Skill {
	skill := Skill{Name: fallback}
	body := raw
	if strings.HasPrefix(raw, "---") {
		rest := strings.TrimPrefix(raw, "---")
		idx := strings.Index(rest, "\n---")
		if idx >= 0 {
			meta := rest[:idx]
			body = strings.TrimSpace(rest[idx+4:])
			_ = yaml.NewDecoder(bytes.NewReader([]byte(meta))).Decode(&skill)
			if skill.Name == "" {
				skill.Name = fallback
			}
		}
	}
	skill.Content = strings.TrimSpace(body)
	if skill.Description == "" {
		skill.Description = firstLine(skill.Content)
	}
	return skill
}

func firstLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(strings.TrimLeft(line, "# "))
		if line != "" {
			return line
		}
	}
	return ""
}

func Select(all []Skill, names []string) []Skill {
	if len(names) == 0 {
		return all
	}
	want := map[string]struct{}{}
	for _, n := range names {
		want[strings.TrimSpace(n)] = struct{}{}
	}
	var out []Skill
	for _, s := range all {
		if _, ok := want[s.Name]; ok {
			out = append(out, s)
		}
	}
	return out
}
