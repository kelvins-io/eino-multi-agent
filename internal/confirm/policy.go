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

func NeedsConfirm(policy, action, path string) bool {
	switch Normalize(policy) {
	case Never:
		return false
	case Always:
		return true
	default:
		return IsRisky(action, path)
	}
}

func IsRisky(action, path string) bool {
	action = strings.ToLower(strings.TrimSpace(action))
	path = strings.ToLower(path)
	switch action {
	case "delete", "remove", "overwrite", "shell", "execute", "http_post", "publish", "send":
		return true
	case "write", "edit":
		return strings.Contains(path, "/output/") || strings.HasSuffix(path, ".sh")
	default:
		return false
	}
}
