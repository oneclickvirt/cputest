#define _POSIX_C_SOURCE 200809L
#include <stdio.h>
#include <stdlib.h>
#include <math.h>
#include <time.h>
#include <stdatomic.h>
#include <stdint.h>
#ifdef _WIN32
#include <windows.h>
#include <process.h>
void portable_sleep_ms(int milliseconds) {
    Sleep(milliseconds);
}
typedef struct {
    LARGE_INTEGER frequency;
    LARGE_INTEGER start;
} win_timer_t;
static win_timer_t win_timer;
static int win_timer_initialized = 0;
void win_timer_init() {
    if (!win_timer_initialized) {
        QueryPerformanceFrequency(&win_timer.frequency);
        win_timer_initialized = 1;
    }
}
void win_clock_gettime(struct timespec *ts) {
    LARGE_INTEGER counter;
    QueryPerformanceCounter(&counter);
    long long nanoseconds = (counter.QuadPart * 1000000000LL) / win_timer.frequency.QuadPart;
    ts->tv_sec = nanoseconds / 1000000000LL;
    ts->tv_nsec = nanoseconds % 1000000000LL;
}
typedef struct {
    HANDLE handle;
    void *(*start_routine)(void *);
    void *arg;
    void *result;
} win_thread_t;
typedef CRITICAL_SECTION win_mutex_t;
static DWORD WINAPI win_thread_proc(LPVOID lpParam) {
    win_thread_t *thread = (win_thread_t *)lpParam;
    thread->result = thread->start_routine(thread->arg);
    return 0;
}
int win_thread_create(win_thread_t *thread, void *(*start_routine)(void *), void *arg) {
    thread->start_routine = start_routine;
    thread->arg = arg;
    thread->handle = CreateThread(NULL, 0, win_thread_proc, thread, 0, NULL);
    return thread->handle ? 0 : -1;
}
int win_thread_join(win_thread_t *thread, void **retval) {
    WaitForSingleObject(thread->handle, INFINITE);
    if (retval) {
        *retval = thread->result;
    }
    CloseHandle(thread->handle);
    return 0;
}
void win_mutex_init(win_mutex_t *mutex) {
    InitializeCriticalSection(mutex);
}
void win_mutex_lock(win_mutex_t *mutex) {
    EnterCriticalSection(mutex);
}
void win_mutex_unlock(win_mutex_t *mutex) {
    LeaveCriticalSection(mutex);
}
void win_mutex_destroy(win_mutex_t *mutex) {
    DeleteCriticalSection(mutex);
}
#define pthread_t win_thread_t
#define pthread_mutex_t win_mutex_t
#define pthread_create(thread, attr, start_routine, arg) win_thread_create(thread, start_routine, arg)
#define pthread_join(thread, retval) win_thread_join(&thread, retval)
#define pthread_mutex_init(mutex, attr) win_mutex_init(mutex)
#define pthread_mutex_lock(mutex) win_mutex_lock(mutex)
#define pthread_mutex_unlock(mutex) win_mutex_unlock(mutex)
#define pthread_mutex_destroy(mutex) win_mutex_destroy(mutex)
#define PTHREAD_MUTEX_INITIALIZER {0}
#define clock_gettime(clk_id, tp) win_clock_gettime(tp)
#define CLOCK_MONOTONIC 0
#else
#include <pthread.h>
#include <unistd.h>
extern int usleep(useconds_t __useconds);
void portable_sleep_ms(int milliseconds) {
    usleep(milliseconds * 1000);
}
#endif
#if defined(__LP64__) || defined(_WIN64) || (defined(__WORDSIZE) && __WORDSIZE == 64) || defined(__x86_64__) || defined(__amd64__) || defined(__aarch64__)
    #define MAX_LATENCY_SAMPLES 100000000  // 100M samples
#else
    #define MAX_LATENCY_SAMPLES 1000000    // 1M samples
#endif
typedef struct
{
    int max_prime;
    int duration_ms;
    int num_threads;
    int max_events;
} Config;
typedef struct
{
    Config config;
    atomic_uint_fast64_t *counter;
    int *done;
    double *latencies;
    int *latency_count;
    int latency_capacity;
    pthread_mutex_t *latency_mutex;
} WorkerArgs;
typedef struct
{
    uint64_t total_events;
    double events_per_second;
    double *latencies;
    int latency_count;
} BenchmarkResult;
static uint64_t verify_primes(int max_prime)
{
    uint64_t n = 0;
    for (int c = 3; c < max_prime; c++)
    {
        double t = sqrt((double)c);
        int is_prime = 1;
        for (int l = 2; (double)l <= t; l++)
        {
            if (c % l == 0)
            {
                is_prime = 0;
                break;
            }
        }
        if (is_prime)
        {
            n++;
        }
    }
    return n;
}
static void *worker(void *arg)
{
    WorkerArgs *args = (WorkerArgs *)arg;
#ifdef _WIN32
    win_timer_init();
#endif
    while (atomic_load(args->counter) < (uint64_t)args->config.max_events)
    {
        if (*args->done)
        {
            break;
        }
        struct timespec start, end;
        clock_gettime(CLOCK_MONOTONIC, &start);
        verify_primes(args->config.max_prime);
        clock_gettime(CLOCK_MONOTONIC, &end);
        double duration = (end.tv_sec - start.tv_sec) * 1000.0 +
                          (end.tv_nsec - start.tv_nsec) / 1000000.0;
        pthread_mutex_lock(args->latency_mutex);
        if (*args->latency_count < args->latency_capacity)
        {
            args->latencies[*args->latency_count] = duration;
            (*args->latency_count)++;
        }
        pthread_mutex_unlock(args->latency_mutex);
        atomic_fetch_add(args->counter, 1);
    }
    return NULL;
}
BenchmarkResult *run_benchmark(Config config)
{
#ifdef _WIN32
    win_timer_init();
#endif
    atomic_uint_fast64_t counter = 0;
    int done = 0;
    int latency_count = 0;
    int latency_capacity = config.max_events;
    if (latency_capacity > MAX_LATENCY_SAMPLES) {
        latency_capacity = MAX_LATENCY_SAMPLES;
    }
    size_t required_bytes = (size_t)latency_capacity * sizeof(double);
    size_t max_safe_bytes = SIZE_MAX / 4;
    if (required_bytes > max_safe_bytes) {
        latency_capacity = (int)(max_safe_bytes / sizeof(double));
    }
    if (latency_capacity < 1000) {
        latency_capacity = 1000;
    }
    double *latencies = (double *)malloc(latency_capacity * sizeof(double));
    pthread_mutex_t latency_mutex = PTHREAD_MUTEX_INITIALIZER;
#ifdef _WIN32
    pthread_mutex_init(&latency_mutex, NULL);
#endif
    if (!latencies)
    {
        return NULL;
    }
    pthread_t *threads = (pthread_t *)malloc(config.num_threads * sizeof(pthread_t));
    WorkerArgs *worker_args = (WorkerArgs *)malloc(config.num_threads * sizeof(WorkerArgs));
    if (!threads || !worker_args)
    {
        free(latencies);
        free(threads);
        free(worker_args);
        return NULL;
    }
    struct timespec start_time, end_time;
    clock_gettime(CLOCK_MONOTONIC, &start_time);
    for (int i = 0; i < config.num_threads; i++)
    {
        worker_args[i].config = config;
        worker_args[i].counter = &counter;
        worker_args[i].done = &done;
        worker_args[i].latencies = latencies;
        worker_args[i].latency_count = &latency_count;
        worker_args[i].latency_capacity = latency_capacity;
        worker_args[i].latency_mutex = &latency_mutex;
        pthread_create(&threads[i], NULL, worker, &worker_args[i]);
    }
    portable_sleep_ms(config.duration_ms);
    done = 1;
    for (int i = 0; i < config.num_threads; i++)
    {
        pthread_join(threads[i], NULL);
    }
    clock_gettime(CLOCK_MONOTONIC, &end_time);
    double duration = (end_time.tv_sec - start_time.tv_sec) +
                      (end_time.tv_nsec - start_time.tv_nsec) / 1000000000.0;
    BenchmarkResult *result = (BenchmarkResult *)malloc(sizeof(BenchmarkResult));
    if (!result)
    {
        free(latencies);
        free(threads);
        free(worker_args);
        return NULL;
    }
    result->total_events = atomic_load(&counter);
    result->events_per_second = (double)result->total_events / duration;
    result->latencies = latencies;
    result->latency_count = latency_count;
    free(threads);
    free(worker_args);
    pthread_mutex_destroy(&latency_mutex);
    return result;
}
void free_benchmark_result(BenchmarkResult *result)
{
    if (result)
    {
        free(result->latencies);
        free(result);
    }
}