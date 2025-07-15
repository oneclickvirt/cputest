#ifndef CPU_BENCHMARK_H
#define CPU_BENCHMARK_H

#include <stdint.h>

typedef struct {
    int max_prime;
    int duration_ms;
    int num_threads;
    int max_events;
} Config;

typedef struct {
    uint64_t total_events;
    double events_per_second;
    double *latencies;
    int latency_count;
} BenchmarkResult;

BenchmarkResult *run_benchmark(Config config);
void free_benchmark_result(BenchmarkResult *result);

#endif // CPU_BENCHMARK_H