package cpu

import (
	"context"
	"math"
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

func TestRunBurnUsesBoundedDefaultForConfiguredBurn(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	result := RunBurn(ctx, BurnConfig{Threads: 1, MaxPrime: 100})
	if result.Status != "canceled" || result.DurationMS <= 0 || result.DurationMS > 500 {
		t.Fatalf("unexpected default burn result: %+v", result)
	}
}

func TestRunBurnNormalizesExtremeDurationAndPrimeLimit(t *testing.T) {
	config := BurnConfig{Duration: time.Hour, MaxPrime: math.MaxInt}
	if config.Duration > MaxBurnDuration {
		config.Duration = MaxBurnDuration
	}
	config.MaxPrime = normalizeMaxPrime(config.MaxPrime, DefaultBurnMaxPrime)
	if config.Duration != MaxBurnDuration || config.MaxPrime != MaxPrimeLimit {
		t.Fatalf("burn limits were not applied: %+v", config)
	}
}

func TestVerifyPrimesBoundsExtremeInput(t *testing.T) {
	if got := verifyPrimes(0); got != 0 {
		t.Fatalf("non-positive prime input changed legacy behavior: %d", got)
	}
	if got := verifyPrimes(math.MaxInt); got == 0 {
		t.Fatal("extreme prime input did not execute the bounded workload")
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
