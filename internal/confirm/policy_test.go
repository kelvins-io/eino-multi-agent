package confirm

import "testing"

func TestNormalize(t *testing.T) {
	if Normalize("") != OnRisk {
		t.Fatalf("empty should default to on_risk")
	}
	if Normalize("ALWAYS") != Always {
		t.Fatalf("always")
	}
	if Normalize("never") != Never {
		t.Fatalf("never")
	}
}

func TestNeedsConfirm(t *testing.T) {
	if NeedsConfirm(Never, "delete", "") {
		t.Fatal("never should skip")
	}
	if !NeedsConfirm(Always, "read", "") {
		t.Fatal("always should confirm")
	}
	if !NeedsConfirm(OnRisk, "delete", "a.txt") {
		t.Fatal("delete is risky")
	}
	if NeedsConfirm(OnRisk, "read", "a.txt") {
		t.Fatal("read is not risky")
	}
}

func TestOverwriteIsRiskyFirstWriteIsNot(t *testing.T) {
	action := Classify("write", "output/report.md", true)
	if action != "overwrite" {
		t.Fatalf("got %s", action)
	}
	if !NeedsConfirm(OnRisk, action, "output/report.md") {
		t.Fatal("overwrite should confirm")
	}
	first := Classify("write", "output/report.md", false)
	if first != "write" {
		t.Fatalf("got %s", first)
	}
	if NeedsConfirm(OnRisk, first, "output/report.md") {
		t.Fatal("first write to output should not confirm on_risk")
	}
}

func TestClassifyShell(t *testing.T) {
	if ClassifyShell("python3 tmp/compute.py") != "execute" {
		t.Fatal("python should be execute")
	}
	if ClassifyShell("rm -rf output/old.md") != "delete" {
		t.Fatal("rm is delete")
	}
	if ClassifyShell("curl -X POST https://example.com") != "http_post" {
		t.Fatal("post is http_post")
	}
	if !NeedsConfirm(OnRisk, ClassifyShell("rm tmp/a"), "rm tmp/a") {
		t.Fatal("delete shell should confirm")
	}
	if NeedsConfirm(OnRisk, ClassifyShell("python3 tmp/compute.py"), "python3 tmp/compute.py") {
		t.Fatal("python should not confirm on_risk")
	}
}

func TestInputWriteIsOverwrite(t *testing.T) {
	if Classify("write", "input/sales.csv", false) != "overwrite" {
		t.Fatal("mutating uploads is overwrite")
	}
}
