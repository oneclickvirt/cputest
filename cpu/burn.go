package cpu

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

type BurnConfig struct {
	Threads  int
	Duration time.Duration
	MaxPrime int
}

type BurnResult struct {
	SchemaVersion    string  `json:"schema_version"`
	Status           string  `json:"status"`
	EffectiveThreads int     `json:"effective_threads"`
	DurationMS       int64   `json:"duration_ms"`
	Events           uint64  `json:"events"`
	EventsPerSecond  float64 `json:"events_per_second"`
	Error            string  `json:"error,omitempty"`
}

func RunBurn(ctx context.Context, config BurnConfig) BurnResult {
	if ctx == nil {
		ctx = context.Background()
	}
	result := BurnResult{SchemaVersion: "goecs.cpu/burn-v1", Status: "skipped"}
	if config.Duration <= 0 {
		// Preserve the historical zero-value skip while allowing an explicitly
		// configured burn to use the bounded default duration.
		if config.Threads <= 0 && config.MaxPrime <= 0 {
			result.Error = "explicit burn duration is not configured"
			return result
		}
		config.Duration = DefaultBurnDuration
	}
	if config.Duration > MaxBurnDuration {
		config.Duration = MaxBurnDuration
	}
	if config.Threads <= 0 {
		config.Threads = runtime.NumCPU()
	}
	config.MaxPrime = normalizeMaxPrime(config.MaxPrime, DefaultBurnMaxPrime)
	result.EffectiveThreads = min(config.Threads, effectiveCPUThreads())
	runCtx, cancel := context.WithTimeout(ctx, config.Duration)
	defer cancel()
	started := time.Now()
	var events atomic.Uint64
	var wait sync.WaitGroup
	wait.Add(result.EffectiveThreads)
	for range result.EffectiveThreads {
		go func() {
			defer wait.Done()
			for runCtx.Err() == nil {
				verifyPrimes(config.MaxPrime)
				events.Add(1)
			}
		}()
	}
	wait.Wait()
	result.DurationMS = time.Since(started).Milliseconds()
	result.Events = events.Load()
	if result.DurationMS > 0 {
		result.EventsPerSecond = float64(result.Events) / (float64(result.DurationMS) / 1000)
	}
	if err := ctx.Err(); err != nil {
		result.Status, result.Error = "canceled", err.Error()
	} else {
		result.Status = "ok"
	}
	return result
}
