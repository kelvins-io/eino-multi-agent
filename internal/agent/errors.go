package agent

import (
	"strings"
)

func ExplainError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	switch {
	case containsAny(msg, "SetLimitExceeded", "inference limit", "Safe Experience Mode", "安全体验模式"):
		return "豆包模型已达到「安全体验模式」推理额度，该接入点已被暂停。请到火山方舟控制台「模型开通」关闭或调整安全体验模式，或把 EINO_LLM_MODEL 换成仍有额度的接入点后再重试。"
	case containsAny(msg, "Error code: 429", `"code":"TooManyRequests"`, "TooManyRequests"):
		return "模型返回 429：请求过于频繁或额度已用尽。请稍后重试、降低并发，或更换 EINO_LLM_MODEL。"
	case containsAny(msg, "Error code: 401", "Unauthorized", "invalid api key", "InvalidApiKey"):
		return "模型鉴权失败，请检查 EINO_LLM_API_KEY 是否有效。"
	case containsAny(msg, "Error code: 404", "ModelNotOpen", "not found"):
		return "当前模型接入点不存在或未开通，请在火山方舟核对 EINO_LLM_MODEL。"
	case containsAny(msg, "context deadline exceeded", "Client.Timeout", "context cancellation while reading body"):
		return "模型响应超时。推理模型（如 Doubao Seed）单次生成经常超过 2 分钟。请把 llm.timeout / EINO_LLM_TIMEOUT 调到 10m 或更长后重试。"
	default:
		return msg
	}
}

func IsTransient(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return containsAny(msg,
		"context deadline exceeded",
		"Client.Timeout",
		"context cancellation while reading body",
		"Error code: 500",
		"Error code: 503",
		"connection reset",
		"EOF",
	)
}

func IsQuotaOrRetryable(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return IsTransient(err) || containsAny(msg, "SetLimitExceeded", "inference limit", "Error code: 429", "TooManyRequests")
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
