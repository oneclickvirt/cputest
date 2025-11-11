package cpu

import (
	"math"
	"sync"
	"sync/atomic"
	"time"
)


// 完全按照 sysbench 的实现来验证质数 见 https://github.com/akopytov/sysbench/blob/master/src/tests/cpu/sb_cpu.c
func verifyPrimes(maxPrime int) uint64 {
	var n uint64 = 0
	// 从3开始验证到最大值
	for c := 3; c < maxPrime; c++ {
		t := math.Sqrt(float64(c))
		isPrime := true
		for l := 2; float64(l) <= t; l++ {
			if c%l == 0 {
				isPrime = false
				break
			}
		}
		if isPrime {
			n++
		}
	}
	return n
}

func worker(config Config, counter *uint64, wg *sync.WaitGroup, done chan bool, latencies chan<- float64) {
	defer wg.Done()
	for atomic.LoadUint64(counter) < uint64(config.MaxEvents) {
		select {
		case <-done:
			return
		default:
			start := time.Now()
			// 执行质数验证
			verifyPrimes(config.MaxPrime)
			// 计算延迟（毫秒）
			duration := float64(time.Since(start).Nanoseconds()) / 1e6
			latencies <- duration
			atomic.AddUint64(counter, 1)
		}
	}
}

func RunBenchmark(config Config) (uint64, float64, []float64) {
	var counter uint64
	var wg sync.WaitGroup
	done := make(chan bool)
	latencyChan := make(chan float64, 1000)
	var latencies []float64
	var collectorWg sync.WaitGroup
	startTime := time.Now()
	// 启动工作线程
	for i := 0; i < config.NumThreads; i++ {
		wg.Add(1)
		go worker(config, &counter, &wg, done, latencyChan)
	}
	// 收集延迟数据
	collectorWg.Add(1)
	go func() {
		defer collectorWg.Done()
		for latency := range latencyChan {
			latencies = append(latencies, latency)
		}
	}()
	// 运行指定时间
	time.Sleep(config.Duration)
	close(done)
	wg.Wait()
	close(latencyChan)
	// 等待收集器完成
	collectorWg.Wait()
	duration := time.Since(startTime).Seconds()
	eventsPerSecond := float64(counter) / duration
	return counter, eventsPerSecond, latencies
}
