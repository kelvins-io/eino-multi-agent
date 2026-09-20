package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/cloudwego/eino/components/tool"
)

func TestSelect(t *testing.T) {
	all := []Skill{
		{Name: "weekly-report", Description: "周报"},
		{Name: "research", Description: "调研"},
	}
	got := Select(all, []string{"research"})
	if len(got) != 1 || got[0].Name != "research" {
		t.Fatalf("select: %#v", got)
	}
	if len(Select(all, nil)) != 2 {
		t.Fatal("empty names should return all")
	}
}

func TestSkillToolLoadsContent(t *testing.T) {
	tl, err := newSkillTool([]Skill{{Name: "research", Description: "调研", Content: "先搜索再总结"}})
	if err != nil {
		t.Fatal(err)
	}
	info, err := tl.Info(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if info.Name != "skill" || !strings.Contains(info.Desc, "research") {
		t.Fatalf("info: %#v", info)
	}
	inv, ok := tl.(tool.InvokableTool)
	if !ok {
		t.Fatal("skill tool should be invokable")
	}
	out, err := inv.InvokableRun(context.Background(), `{"name":"research"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "先搜索再总结") {
		t.Fatalf("content: %s", out)
	}
}
