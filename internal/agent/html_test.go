package agent

import (
	"strings"
	"testing"
)

func TestReadableHTML(t *testing.T) {
	raw := `<html><head><style>p{color:red}</style></head><body><h1>调研标题</h1><script>alert(1)</script><p>第一段内容</p></body></html>`
	got := ReadableHTML(raw)
	if !strings.Contains(got, "调研标题") || !strings.Contains(got, "第一段内容") {
		t.Fatalf("got %q", got)
	}
	if strings.Contains(got, "alert") || strings.Contains(got, "color:red") {
		t.Fatalf("should strip script/style: %q", got)
	}
}
