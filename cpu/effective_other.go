//go:build !linux

package cpu

import "runtime"

func effectiveCPUThreads() int { return max(runtime.GOMAXPROCS(0), 1) }
