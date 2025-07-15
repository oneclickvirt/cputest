package cpu

// #cgo CFLAGS: -std=c11
// #include "cpu_benchmark.h"
import "C"
import (
	"time"
	"unsafe"
)

type Config struct {
	MaxPrime   int
	Duration   time.Duration
	NumThreads int
	MaxEvents  int
}

func DefaultConfig() Config {
	return Config{
		MaxPrime:   10000,
		Duration:   5 * time.Second,
		NumThreads: 1,
		MaxEvents:  1000000,
	}
}

func RunBenchmark(config Config) (uint64, float64, []float64) {
	cConfig := C.Config{
		max_prime:   C.int(config.MaxPrime),
		duration_ms: C.int(config.Duration.Milliseconds()),
		num_threads: C.int(config.NumThreads),
		max_events:  C.int(config.MaxEvents),
	}
	cResult := C.run_benchmark(cConfig)
	if cResult == nil {
		return 0, 0, []float64{}
	}
	defer C.free_benchmark_result(cResult)
	totalEvents := uint64(cResult.total_events)
	eventsPerSecond := float64(cResult.events_per_second)
	latencies := make([]float64, int(cResult.latency_count))
	if cResult.latency_count > 0 {
		cLatencies := (*[1 << 30]C.double)(unsafe.Pointer(cResult.latencies))[:cResult.latency_count:cResult.latency_count]
		for i, lat := range cLatencies {
			latencies[i] = float64(lat)
		}
	}
	return totalEvents, eventsPerSecond, latencies
}
