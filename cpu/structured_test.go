package cpu

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sync/atomic"
	"testing"
	"time"
)

func TestReadLinuxTemperaturesFiltersNonCPUSensors(t *testing.T) {
	root := t.TempDir()
	write := func(path, value string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(root, "class/thermal/thermal_zone0/type"), "x86_pkg_temp\n")
	write(filepath.Join(root, "class/thermal/thermal_zone0/temp"), "50000\n")
	write(filepath.Join(root, "class/thermal/thermal_zone1/type"), "gpu\n")
	write(filepath.Join(root, "class/thermal/thermal_zone1/temp"), "90000\n")
	write(filepath.Join(root, "class/hwmon/hwmon0/name"), "coretemp\n")
	write(filepath.Join(root, "class/hwmon/hwmon0/temp1_label"), "Package id 0\n")
	write(filepath.Join(root, "class/hwmon/hwmon0/temp1_input"), "55000\n")
	write(filepath.Join(root, "class/hwmon/hwmon1/name"), "nvme\n")
	write(filepath.Join(root, "class/hwmon/hwmon1/temp1_input"), "80000\n")
	got := readLinuxTemperaturesFrom(root)
	if len(got) != 2 || got[0] != 50 || got[1] != 55 {
		t.Fatalf("unexpected CPU temperatures: %#v", got)
	}
}

func TestRunStructuredUsesEffectiveThreadsAndTemperature(t *testing.T) {
	result := RunStructured(context.Background(), StructuredConfig{
		Threads: 1000, Duration: 30 * time.Millisecond, MaxPrime: 100,
		TemperatureReader: func() []float64 { return []float64{40, 45} },
	})
	if result.Status != "ok" || result.EffectiveThreads <= 0 || result.EffectiveThreads >= result.RequestedThreads || result.Events == 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if !result.Temperature.Available || result.Temperature.MaxC == nil || *result.Temperature.MaxC != 45 {
		t.Fatalf("unexpected temperature: %+v", result.Temperature)
	}
}

func TestRunStructuredClampsExtremePrimeLimit(t *testing.T) {
	config := StructuredConfig{Threads: 1, Duration: time.Millisecond, MaxPrime: math.MaxInt}
	config.MaxPrime = normalizeMaxPrime(config.MaxPrime, DefaultStructuredPrime)
	if config.MaxPrime != MaxPrimeLimit {
		t.Fatalf("structured prime limit was not clamped: %d", config.MaxPrime)
	}
}

func TestRunStructuredCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result := RunStructured(ctx, StructuredConfig{})
	if result.Status != "canceled" {
		t.Fatalf("unexpected status: %+v", result)
	}
}

func TestSampleTemperaturesRecordsOnePeakPerSnapshot(t *testing.T) {
	var calls atomic.Int32
	reader := func() []float64 {
		if calls.Add(1) == 1 {
			return []float64{40, 70, 151}
		}
		return []float64{55, 60, -21}
	}
	done := make(chan struct{})
	result := make(chan TemperatureResult, 1)
	go sampleTemperatures(context.Background(), reader, done, result)
	time.Sleep(250 * time.Millisecond)
	close(done)
	got := <-result
	if !reflect.DeepEqual(got.Samples, []float64{70, 60}) {
		t.Fatalf("unexpected samples: %+v", got)
	}
	if got.BaselineC == nil || got.MinC == nil || got.MaxC == nil || got.DeltaC == nil ||
		*got.BaselineC != 70 || *got.MinC != 60 || *got.MaxC != 70 || *got.DeltaC != 0 {
		t.Fatalf("unexpected summary: %+v", got)
	}
}

func TestRunStructuredDoesNotWaitForBlockedTemperatureReader(t *testing.T) {
	started := time.Now()
	result := RunStructured(context.Background(), StructuredConfig{
		Threads: 1, Duration: 20 * time.Millisecond, MaxPrime: 100,
		TemperatureReader: func() []float64 {
			select {}
		},
	})
	if elapsed := time.Since(started); elapsed > 500*time.Millisecond {
		t.Fatalf("blocked temperature reader delayed result by %s", elapsed)
	}
	if result.Status != "ok" || result.Temperature.Available {
		t.Fatalf("unexpected result: %+v", result)
	}
}
