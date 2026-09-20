package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

func TestContextReducerTruncatesOldToolResults(t *testing.T) {
	h := NewContextReducer().(*ContextReducer)
	h.KeepLast = 2
	h.MaxRunes = 8
	long := strings.Repeat("字", 40)
	state := &adk.ChatModelAgentState{
		Messages: []*schema.Message{
			{Role: schema.Tool, Content: long},
			{Role: schema.Assistant, Content: "ok"},
			{Role: schema.Tool, Content: "short"},
			{Role: schema.User, Content: "go"},
		},
	}
	_, out, err := h.BeforeModelRewriteState(context.Background(), state, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.Messages[0].Content, "已压缩") {
		t.Fatalf("old tool result should shrink: %q", out.Messages[0].Content)
	}
	if out.Messages[2].Content != "short" {
		t.Fatal("recent tool result should stay")
	}
}
