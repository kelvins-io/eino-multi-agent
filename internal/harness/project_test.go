package harness

import (
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestProjectWriteTodos(t *testing.T) {
	msg := &schema.Message{
		Role: schema.Assistant,
		ToolCalls: []schema.ToolCall{{
			Function: schema.FunctionCall{
				Name:      "write_todos",
				Arguments: `{"todos":[{"content":"读 CSV","status":"completed"},{"content":"写周报","status":"in_progress"}]}`,
			},
		}},
	}
	steps := projectMessage(msg)
	if len(steps) != 1 || steps[0].Type != "todos" || len(steps[0].Todos) != 2 {
		t.Fatalf("%#v", steps)
	}
	if !strings.Contains(steps[0].Message, "1/2") {
		t.Fatalf("summary %q", steps[0].Message)
	}
}

func TestProjectWriteFile(t *testing.T) {
	msg := &schema.Message{
		Role: schema.Assistant,
		ToolCalls: []schema.ToolCall{{
			Function: schema.FunctionCall{Name: "write_file", Arguments: `{"file_path":"output/report.md"}`},
		}},
	}
	steps := projectMessage(msg)
	if len(steps) != 1 || steps[0].Type != "write" || !steps[0].RefreshArtifacts {
		t.Fatalf("%#v", steps)
	}
	if steps[0].Message != "写入 output/report.md" {
		t.Fatalf("msg %q", steps[0].Message)
	}
}

func TestProjectThinkSkipsToolResults(t *testing.T) {
	if steps := projectMessage(&schema.Message{Role: schema.Tool, Content: "ok"}); len(steps) != 0 {
		t.Fatal(steps)
	}
	steps := projectMessage(&schema.Message{Role: schema.Assistant, Content: "接下来读取销售表"})
	if len(steps) != 1 || steps[0].Type != "think" {
		t.Fatal(steps)
	}
}

func TestIsPreviewable(t *testing.T) {
	if !Previewable("a.md", "text/markdown") || !Previewable("a.png", "image/png") {
		t.Fatal("should preview")
	}
	if Previewable("a.pptx", "application/vnd.openxmlformats-officedocument.presentationml.presentation") {
		t.Fatal("pptx is download only")
	}
}
