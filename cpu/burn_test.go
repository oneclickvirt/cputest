package cpu

import (
	"context"
	"testing"
	"time"
)

func TestRunBurnSkipsWithoutExplicitDuration(t *testing.T) {
	result := RunBurn(context.Background(), BurnConfig{})
	if result.Status != "skipped" || result.Error == "" {
		t.Fatalf("unexpected skipped burn: %+v", result)
	}
}

func TestRunBurnCompletesExplicitDuration(t *testing.T) {
	result := RunBurn(context.Background(), BurnConfig{Threads: 1, Duration: 10 * time.Millisecond, MaxPrime: 100})
	if result.Status != "ok" || result.Events == 0 || result.DurationMS < 1 {
		t.Fatalf("unexpected burn result: %+v", result)
	}
}

func TestRunBurnHonorsCanceledParent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result := RunBurn(ctx, BurnConfig{Duration: time.Minute})
	if result.Status != "canceled" {
		t.Fatalf("unexpected canceled burn: %+v", result)
	}
}
