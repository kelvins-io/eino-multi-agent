package agent

import (
	"context"
	"unicode/utf8"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// ContextReducer truncates older tool results so long tasks stay within the model window.
type ContextReducer struct {
	*adk.BaseChatModelAgentMiddleware
	MaxRunes int
	KeepLast int
}

func NewContextReducer() adk.ChatModelAgentMiddleware {
	return &ContextReducer{
		BaseChatModelAgentMiddleware: &adk.BaseChatModelAgentMiddleware{},
		MaxRunes:                     1200,
		KeepLast:                     12,
	}
}

func (h *ContextReducer) BeforeModelRewriteState(ctx context.Context, state *adk.ChatModelAgentState, mc *adk.ModelContext) (context.Context, *adk.ChatModelAgentState, error) {
	if state == nil || len(state.Messages) == 0 {
		return ctx, state, nil
	}
	keep := h.KeepLast
	if keep <= 0 {
		keep = 12
	}
	maxRunes := h.MaxRunes
	if maxRunes <= 0 {
		maxRunes = 1200
	}
	cutoff := len(state.Messages) - keep
	if cutoff < 0 {
		cutoff = 0
	}
	for i := 0; i < cutoff; i++ {
		msg := state.Messages[i]
		if msg == nil || msg.Role != schema.Tool {
			continue
		}
		if utf8.RuneCountInString(msg.Content) <= maxRunes {
			continue
		}
		cloned := *msg
		cloned.Content = truncateRunes(msg.Content, maxRunes) + "\n...[已压缩，完整过程见任务时间线]"
		state.Messages[i] = &cloned
	}
	return ctx, state, nil
}

func truncateRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}
