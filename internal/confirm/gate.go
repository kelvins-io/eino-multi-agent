package confirm

import (
	"context"
	"errors"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
)

var ErrRejected = errors.New("用户已拒绝该操作")

// Gate pauses the current tool when the confirm policy requires it.
// On resume, approved=true continues; approved=false returns ErrRejected.
func Gate(ctx context.Context, policy, action, target string) error {
	if isResume, hasData, approved := tool.GetResumeContext[bool](ctx); isResume {
		if hasData && !approved {
			return fmt.Errorf("%w: %s %s", ErrRejected, action, target)
		}
		return nil
	}
	if !NeedsConfirm(policy, action, target) {
		return nil
	}
	return tool.Interrupt(ctx, fmt.Sprintf("请求确认：%s %s", action, target))
}
