package workspace

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	localbk "github.com/cloudwego/eino-ext/adk/backend/local"
	"github.com/cloudwego/eino/adk/filesystem"
	"github.com/cloudwego/eino/schema"
	"github.com/kelvins-io/eino-multi-agent/internal/confirm"
)

// absPathTokenRe matches filesystem paths that start at a token boundary.
// It must not match the "/file" suffix inside relative paths like tmp/compute.py.
var absPathTokenRe = regexp.MustCompile(`(^|[\s'"=])(/[^\s'"]+)`)

type BoundBackend struct {
	inner  *localbk.Local
	root   string
	policy string
}

func NewBackend(ctx context.Context, sb *Sandbox, policy string) (*BoundBackend, error) {
	inner, err := localbk.NewBackend(ctx, &localbk.Config{
		ValidateCommand: func(cmd string) error {
			return validateCommand(sb.Root, cmd)
		},
	})
	if err != nil {
		return nil, err
	}
	return &BoundBackend{inner: inner, root: sb.Root, policy: confirm.Normalize(policy)}, nil
}

func (b *BoundBackend) rewrite(p string) (string, error) {
	return Confine(b.root, p)
}

func (b *BoundBackend) LsInfo(ctx context.Context, req *filesystem.LsInfoRequest) ([]filesystem.FileInfo, error) {
	if req == nil {
		req = &filesystem.LsInfoRequest{}
	}
	p, err := b.rewrite(req.Path)
	if err != nil {
		return nil, err
	}
	cloned := *req
	cloned.Path = p
	return b.inner.LsInfo(ctx, &cloned)
}

func (b *BoundBackend) Read(ctx context.Context, req *filesystem.ReadRequest) (*filesystem.FileContent, error) {
	p, err := b.rewrite(req.FilePath)
	if err != nil {
		return nil, err
	}
	cloned := *req
	cloned.FilePath = p
	return b.inner.Read(ctx, &cloned)
}

func (b *BoundBackend) GrepRaw(ctx context.Context, req *filesystem.GrepRequest) ([]filesystem.GrepMatch, error) {
	if req == nil {
		req = &filesystem.GrepRequest{}
	}
	p, err := b.rewrite(req.Path)
	if err != nil {
		return nil, err
	}
	cloned := *req
	cloned.Path = p
	return b.inner.GrepRaw(ctx, &cloned)
}

func (b *BoundBackend) GlobInfo(ctx context.Context, req *filesystem.GlobInfoRequest) ([]filesystem.FileInfo, error) {
	if req == nil {
		req = &filesystem.GlobInfoRequest{}
	}
	p, err := b.rewrite(req.Path)
	if err != nil {
		return nil, err
	}
	cloned := *req
	cloned.Path = p
	return b.inner.GlobInfo(ctx, &cloned)
}

func (b *BoundBackend) Write(ctx context.Context, req *filesystem.WriteRequest) error {
	p, err := b.rewrite(req.FilePath)
	if err != nil {
		return err
	}
	action := confirm.Classify("write", b.displayPath(p), fileExists(p))
	if err := confirm.Gate(ctx, b.policy, action, b.displayPath(p)); err != nil {
		return err
	}
	cloned := *req
	cloned.FilePath = p
	return b.inner.Write(ctx, &cloned)
}

func (b *BoundBackend) Edit(ctx context.Context, req *filesystem.EditRequest) error {
	p, err := b.rewrite(req.FilePath)
	if err != nil {
		return err
	}
	action := confirm.Classify("edit", b.displayPath(p), fileExists(p))
	if err := confirm.Gate(ctx, b.policy, action, b.displayPath(p)); err != nil {
		return err
	}
	cloned := *req
	cloned.FilePath = p
	return b.inner.Edit(ctx, &cloned)
}

func (b *BoundBackend) ExecuteStreaming(ctx context.Context, input *filesystem.ExecuteRequest) (*schema.StreamReader[*filesystem.ExecuteResponse], error) {
	if input == nil {
		return nil, fmt.Errorf("command is required")
	}
	action := confirm.ClassifyShell(input.Command)
	if err := confirm.Gate(ctx, b.policy, action, truncate(input.Command, 180)); err != nil {
		return nil, err
	}
	rewritten, err := rewriteCommand(b.root, input.Command)
	if err != nil {
		return nil, err
	}
	cloned := *input
	cloned.Command = fmt.Sprintf("cd %s && { %s ; }", shellQuote(b.root), rewritten)
	return b.inner.ExecuteStreaming(ctx, &cloned)
}

func (b *BoundBackend) displayPath(abs string) string {
	rel, err := filepath.Rel(b.root, abs)
	if err != nil {
		return abs
	}
	return filepath.ToSlash(rel)
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func validateCommand(root, cmd string) error {
	_, err := rewriteCommand(root, cmd)
	return err
}

func rewriteCommand(root, cmd string) (string, error) {
	if err := denyDangerous(cmd); err != nil {
		return "", err
	}
	matches := absPathTokenRe.FindAllStringSubmatchIndex(cmd, -1)
	if len(matches) == 0 {
		return cmd, nil
	}
	var b strings.Builder
	last := 0
	for _, m := range matches {
		pathStart, pathEnd := m[4], m[5]
		path := cmd[pathStart:pathEnd]
		mapped, err := confineCommandPath(root, path)
		if err != nil {
			return "", err
		}
		b.WriteString(cmd[last:pathStart])
		quoted := pathStart > 0 && (cmd[pathStart-1] == '\'' || cmd[pathStart-1] == '"')
		if !quoted && needsShellQuote(mapped) {
			b.WriteString(shellQuote(mapped))
		} else {
			b.WriteString(mapped)
		}
		last = pathEnd
	}
	b.WriteString(cmd[last:])
	return b.String(), nil
}

func denyDangerous(cmd string) error {
	lower := strings.ToLower(cmd)
	denied := []string{
		"sudo ", "mkfs", "shutdown", "reboot", "dd if=", "chmod -r /",
		"rm -rf /", "rm -fr /", "curl | sh", "wget | sh", "mkfs.",
		":(){", "/etc/passwd", "/etc/shadow",
	}
	for _, d := range denied {
		if strings.Contains(lower, d) {
			return fmt.Errorf("command not allowed: contains %q", d)
		}
	}
	return nil
}

func confineCommandPath(root, p string) (string, error) {
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	if allowedSystemPath(abs) {
		return p, nil
	}
	if mapped, err := Confine(root, p); err == nil {
		return mapped, nil
	}
	rel := strings.TrimPrefix(filepath.Clean(p), string(filepath.Separator))
	first, _, found := strings.Cut(rel, string(filepath.Separator))
	sandboxRelative := first == "input" || first == "output" || first == "tmp" || !found
	if sandboxRelative && rel != "" && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		mapped := filepath.Join(root, rel)
		mapped, err := filepath.Abs(mapped)
		if err == nil && inside(root, mapped) {
			return mapped, nil
		}
	}
	return "", fmt.Errorf("command path outside workspace: %s", p)
}

func needsShellQuote(s string) bool {
	return strings.ContainsAny(s, " \t\n'\"")
}

func allowedSystemPath(p string) bool {
	prefixes := []string{"/bin", "/usr", "/opt/homebrew", "/Library/Developer", "/System/Library"}
	for _, prefix := range prefixes {
		if p == prefix || strings.HasPrefix(p, prefix+"/") {
			return true
		}
	}
	return false
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}
