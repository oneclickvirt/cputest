package cpu

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type StructuredConfig struct {
	Threads           int
	Duration          time.Duration
	MaxPrime          int
	TemperatureReader func() []float64
}

type TemperatureResult struct {
	Available bool      `json:"available"`
	Source    string    `json:"source,omitempty"`
	BaselineC *float64  `json:"baseline_c,omitempty"`
	MinC      *float64  `json:"min_c,omitempty"`
	MaxC      *float64  `json:"max_c,omitempty"`
	DeltaC    *float64  `json:"delta_c,omitempty"`
	Samples   []float64 `json:"samples,omitempty"`
}

type StructuredResult struct {
	SchemaVersion    string            `json:"schema_version"`
	Status           string            `json:"status"`
	RequestedThreads int               `json:"requested_threads"`
	EffectiveThreads int               `json:"effective_threads"`
	Events           uint64            `json:"events"`
	EventsPerSecond  float64           `json:"events_per_second"`
	DurationMS       int64             `json:"duration_ms"`
	Temperature      TemperatureResult `json:"temperature"`
	Error            string            `json:"error,omitempty"`
}

func RunStructured(ctx context.Context, config StructuredConfig) StructuredResult {
	if ctx == nil {
		ctx = context.Background()
	}
	if config.Threads <= 0 {
		config.Threads = runtime.NumCPU()
	}
	if config.Duration <= 0 {
		config.Duration = 5 * time.Second
	}
	if config.Duration > 20*time.Second {
		config.Duration = 20 * time.Second
	}
	config.MaxPrime = normalizeMaxPrime(config.MaxPrime, DefaultStructuredPrime)
	effective := min(config.Threads, effectiveCPUThreads())
	result := StructuredResult{SchemaVersion: "goecs.cpu/v1", Status: "ok", RequestedThreads: config.Threads, EffectiveThreads: effective}
	if err := ctx.Err(); err != nil {
		result.Status, result.Error = structuredCPUStop(err)
		return result
	}
	reader := config.TemperatureReader
	if reader == nil {
		reader = readLinuxTemperatures
	}
	temperatureDone := make(chan struct{})
	temperatureResult := make(chan TemperatureResult, 1)
	go sampleTemperatures(ctx, reader, temperatureDone, temperatureResult)

	runCtx, cancel := context.WithTimeout(ctx, config.Duration)
	started := time.Now()
	var events atomic.Uint64
	var workers sync.WaitGroup
	workers.Add(effective)
	for range effective {
		go func() {
			defer workers.Done()
			for runCtx.Err() == nil {
				verifyPrimes(config.MaxPrime)
				events.Add(1)
			}
		}()
	}
	workers.Wait()
	cancel()
	close(temperatureDone)
	result.Temperature = <-temperatureResult
	duration := time.Since(started)
	result.DurationMS = duration.Milliseconds()
	result.Events = events.Load()
	if duration > 0 {
		result.EventsPerSecond = float64(result.Events) / duration.Seconds()
	}
	if err := ctx.Err(); err != nil {
		result.Status, result.Error = structuredCPUStop(err)
	}
	return result
}

func structuredCPUStop(err error) (string, string) {
	if errors.Is(err, context.Canceled) {
		return "canceled", "canceled"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout", "timeout"
	}
	return "error", "benchmark_failed"
}

func sampleTemperatures(ctx context.Context, reader func() []float64, done <-chan struct{}, result chan<- TemperatureResult) {
	values := make([]float64, 0, 16)
	readResult := make(chan []float64, 1)
	reading := false
	startRead := func() {
		if reading {
			return
		}
		reading = true
		go func() {
			readResult <- append([]float64(nil), reader()...)
		}()
	}
	record := func(snapshot []float64) {
		if peak, ok := temperaturePeak(snapshot); ok {
			values = append(values, peak)
		}
	}
	finish := func() {
		select {
		case snapshot := <-readResult:
			record(snapshot)
		default:
		}
		result <- summarizeTemperatures(values)
	}
	startRead()
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case snapshot := <-readResult:
			reading = false
			record(snapshot)
		case <-ticker.C:
			startRead()
		case <-done:
			finish()
			return
		case <-ctx.Done():
			finish()
			return
		}
	}
}

func temperaturePeak(values []float64) (float64, bool) {
	var peak float64
	found := false
	for _, value := range values {
		if value < -20 || value > 150 {
			continue
		}
		if !found || value > peak {
			peak = value
			found = true
		}
	}
	return peak, found
}

func summarizeTemperatures(values []float64) TemperatureResult {
	if len(values) == 0 {
		return TemperatureResult{}
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	baseline, minimum, maximum := values[0], sorted[0], sorted[len(sorted)-1]
	delta := maximum - baseline
	return TemperatureResult{Available: true, Source: "sysfs", BaselineC: &baseline, MinC: &minimum, MaxC: &maximum, DeltaC: &delta, Samples: values}
}

func readLinuxTemperatures() []float64 {
	if runtime.GOOS != "linux" {
		return nil
	}
	return readLinuxTemperaturesFrom("/sys")
}

func readLinuxTemperaturesFrom(sysRoot string) []float64 {
	var result []float64
	thermalPaths, _ := filepath.Glob(filepath.Join(sysRoot, "class/thermal/thermal_zone*/temp"))
	for _, path := range thermalPaths {
		sensorType := readSensorText(filepath.Join(filepath.Dir(path), "type"))
		if !isCPUTemperatureSensor(sensorType, "") {
			continue
		}
		if value, ok := readTemperatureValue(path); ok {
			result = append(result, value)
		}
	}
	hwmonPaths, _ := filepath.Glob(filepath.Join(sysRoot, "class/hwmon/hwmon*/temp*_input"))
	for _, path := range hwmonPaths {
		directory := filepath.Dir(path)
		driver := readSensorText(filepath.Join(directory, "name"))
		base := strings.TrimSuffix(filepath.Base(path), "_input")
		label := readSensorText(filepath.Join(directory, base+"_label"))
		if !isCPUTemperatureSensor(driver, label) {
			continue
		}
		if value, ok := readTemperatureValue(path); ok {
			result = append(result, value)
		}
	}
	return result
}

func readSensorText(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(string(data)))
}

func readTemperatureValue(path string) (float64, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	value, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64)
	if err != nil {
		return 0, false
	}
	if value > 1000 {
		value /= 1000
	}
	return value, true
}

func isCPUTemperatureSensor(driver, label string) bool {
	text := strings.ToLower(strings.TrimSpace(driver + " " + label))
	for _, excluded := range []string{"nvme", "gpu", "amdgpu", "wifi", "battery", "pch", "ssd"} {
		if strings.Contains(text, excluded) {
			return false
		}
	}
	for _, marker := range []string{"coretemp", "k10temp", "zenpower", "cpu", "x86_pkg_temp", "package id", "tctl", "tdie", "soc_thermal"} {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}
