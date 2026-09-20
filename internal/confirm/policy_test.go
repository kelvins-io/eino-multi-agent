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
