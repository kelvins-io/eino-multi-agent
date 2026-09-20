package confirm

import "strings"

const (
	Always = "always"
	OnRisk = "on_risk"
	Never  = "never"
)

func Normalize(policy string) string {
	switch strings.ToLower(strings.TrimSpace(policy)) {
	case Always:
		return Always
	case Never:
		return Never
	default:
		return OnRisk
	}
}

func NeedsConfirm(policy, action, target string) bool {
	action = Classify(action, target, false)
	switch Normalize(policy) {
	case Never:
		return false
	case Always:
		return true
	default:
		return IsRisky(action, target)
	}
}

// Classify maps a raw tool action into a stable risk class.
// exists is only used for write/edit; pass true when the target file already exists.
func Classify(action, target string, exists bool) string {
	action = strings.ToLower(strings.TrimSpace(action))
	switch action {
	case "write", "edit":
		if exists || isInputPath(target) {
			return "overwrite"
		}
		return "write"
	case "shell", "execute":
		return ClassifyShell(target)
	default:
		return action
	}
}

func ClassifyShell(cmd string) string {
	lower := strings.ToLower(cmd)
	switch {
	case containsAny(lower, "rm ", "rm\t", "unlink ", "rmdir "):
		return "delete"
	case containsAny(lower, "curl -x post", "curl -x put", "curl -x patch", "curl -x delete",
		"wget --post", "--data "):
		return "http_post"
	case containsAny(lower, "mv ", "mv\t"):
		return "overwrite"
	case containsAny(lower, "chmod ", "chown ", "kill ", "pkill ", "mkfs", "dd ", "sudo ", "ssh ", "scp "):
		return "shell"
	default:
		return "execute"
	}
}

func IsRisky(action, target string) bool {
	action = strings.ToLower(strings.TrimSpace(action))
	switch action {
	case "delete", "remove", "overwrite", "shell", "http_post", "publish", "send":
		return true
	case "write", "edit":
		return strings.HasSuffix(strings.ToLower(target), ".sh")
	default:
		return false
	}
}

func isInputPath(path string) bool {
	p := strings.ToLower(strings.ReplaceAll(path, "\\", "/"))
	return strings.Contains(p, "/input/") || strings.HasPrefix(p, "input/")
}

func containsAny(s string, parts ...string) bool {
	for _, p := range parts {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}
