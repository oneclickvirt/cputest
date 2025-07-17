//go:build (
//     ((linux || windows) && (amd64 || arm64 || riscv64 || mips64 || mips64le || ppc64le)) ||
//     (darwin && (amd64 || arm64))
//   ) &&
//   !( (freebsd || openbsd || netbsd) && (amd64 || arm64) ) &&
//   !(windows && arm64) &&
//   !(linux && arm)
package cpu

// #cgo CFLAGS: -std=c11
// #cgo LDFLAGS: -lm
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
		const maxArraySize = 1 << 28
		arraySize := int(cResult.latency_count)
		if arraySize > maxArraySize {
			arraySize = maxArraySize
		}
		cLatencies := (*[1 << 28]C.double)(unsafe.Pointer(cResult.latencies))[:arraySize:arraySize]
		copyCount := int(cResult.latency_count)
		if copyCount > len(cLatencies) {
			copyCount = len(cLatencies)
		}
		for i := 0; i < copyCount; i++ {
			latencies[i] = float64(cLatencies[i])
		}
	}
	return totalEvents, eventsPerSecond, latencies
}
