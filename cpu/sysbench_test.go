package cpu

import (
	"fmt"
	"testing"
)

func Test_RunBenchmark(t *testing.T) {
	// sysbench cpu --threads=1 --cpu-max-prime=10000 --events=1000000 --time=5 run
	config := DefaultConfig()
	fmt.Printf("Running the test with following options:\n")
	fmt.Printf("Number of threads: %d\n", config.NumThreads)
	fmt.Printf("Initializing random number generator from current time\n")
	fmt.Printf("Prime numbers limit: %d\n", config.MaxPrime)
	fmt.Printf("Initializing worker threads...\n")
	fmt.Printf("Threads started!\n")
	totalEvents, eps, latencies := RunBenchmark(config)
	// 计算延迟统计
	var minLatency, maxLatency, sumLatency float64
	if len(latencies) > 0 {
		minLatency = latencies[0]
		maxLatency = latencies[0]
		for _, lat := range latencies {
			if lat < minLatency {
				minLatency = lat
			}
			if lat > maxLatency {
				maxLatency = lat
			}
			sumLatency += lat
		}
	}
	avgLatency := sumLatency / float64(len(latencies))
	fmt.Printf("\nCPU speed:\n")
	fmt.Printf("    events per second: %8.2f\n\n", eps)
	fmt.Printf("General statistics:\n")
	fmt.Printf("    total time:                          %.4fs\n", config.Duration.Seconds())
	fmt.Printf("    total number of events:              %d\n\n", totalEvents)
	fmt.Printf("Latency (ms):\n")
	fmt.Printf("         min:                              %.2f\n", minLatency)
	fmt.Printf("         avg:                              %.2f\n", avgLatency)
	fmt.Printf("         max:                              %.2f\n", maxLatency)
	fmt.Printf("         sum:                              %.2f\n", sumLatency)
}
