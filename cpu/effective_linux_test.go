//go:build linux

package cpu

import (
	"errors"
	"testing"
)

func TestCountCPUSet(t *testing.T) {
	if got := countCPUSet("0-3,8,10-11"); got != 7 {
		t.Fatalf("got %d CPUs", got)
	}
}

func TestEffectiveCPUThreadsV2NestedLimits(t *testing.T) {
	files := map[string]string{
		"/proc/self/cgroup":                               "0::/tenant/job\n",
		"/proc/self/mountinfo":                            "36 25 0:32 / /sys/fs/cgroup rw - cgroup2 cgroup rw\n",
		"/sys/fs/cgroup/tenant/job/cpu.max":               "max 100000\n",
		"/sys/fs/cgroup/tenant/job/cpuset.cpus.effective": "0-7\n",
		"/sys/fs/cgroup/tenant/cpu.max":                   "150000 100000\n",
		"/sys/fs/cgroup/tenant/cpuset.cpus.effective":     "0-3\n",
		"/sys/fs/cgroup/cpu.max":                          "max 100000\n",
		"/sys/fs/cgroup/cpuset.cpus.effective":            "0-15\n",
	}
	if got := effectiveCPUThreadsFrom(mapCPUFileReader(files), 16, 12); got != 2 {
		t.Fatalf("got %d effective threads, want 2", got)
	}
}

func TestEffectiveCPUThreadsV1NestedInheritedLimits(t *testing.T) {
	files := map[string]string{
		"/proc/self/cgroup": "2:cpu,cpuacct:/docker/job\n3:cpuset:/docker/job\n",
		"/proc/self/mountinfo": "29 23 0:26 /docker /sys/fs/cgroup/cpu rw - cgroup cgroup rw,cpu,cpuacct\n" +
			"30 23 0:27 /docker /sys/fs/cgroup/cpuset rw - cgroup cgroup rw,cpuset\n",
		"/sys/fs/cgroup/cpu/job/cpu.cfs_quota_us": "-1\n",
		"/sys/fs/cgroup/cpu/cpu.cfs_quota_us":     "200000\n",
		"/sys/fs/cgroup/cpu/cpu.cfs_period_us":    "100000\n",
		"/sys/fs/cgroup/cpuset/job/cpuset.cpus":   "\n",
		"/sys/fs/cgroup/cpuset/cpuset.cpus":       "2-5\n",
	}
	if got := effectiveCPUThreadsFrom(mapCPUFileReader(files), 16, 8); got != 2 {
		t.Fatalf("got %d effective threads, want 2", got)
	}
}

func TestEffectiveCPUThreadsFallsBackToRuntimeAndAffinity(t *testing.T) {
	if got := effectiveCPUThreadsFrom(mapCPUFileReader(nil), 12, 6); got != 6 {
		t.Fatalf("got %d effective threads, want 6", got)
	}
}

func mapCPUFileReader(files map[string]string) func(string) ([]byte, error) {
	return func(path string) ([]byte, error) {
		value, ok := files[path]
		if !ok {
			return nil, errors.New("not found")
		}
		return []byte(value), nil
	}
}
