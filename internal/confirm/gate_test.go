package confirm

import (
	"context"
	"errors"
	"testing"
)

func TestGateNeverAllows(t *testing.T) {
	if err := Gate(context.Background(), Never, "overwrite", "output/a.md"); err != nil {
		t.Fatal(err)
	}
}

func TestGateOnRiskFirstWriteAllows(t *testing.T) {
	if err := Gate(context.Background(), OnRisk, "write", "output/a.md"); err != nil {
		t.Fatal(err)
	}
}

func TestGateOnRiskOverwriteInterrupts(t *testing.T) {
	err := Gate(context.Background(), OnRisk, "overwrite", "output/a.md")
	if err == nil {
		t.Fatal("expected interrupt")
	}
	if !errors.Is(err, ErrRejected) && err.Error() == "" {
		t.Fatal(err)
	}
}
