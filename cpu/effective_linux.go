//go:build linux

package cpu

import (
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

type cgroupMembership struct {
	v2          string
	controllers map[string]string
}

type cgroupMount struct {
	root        string
	mountPoint  string
	fsType      string
	controllers map[string]struct{}
}

func effectiveCPUThreads() int {
	affinityCount := 0
	var affinity unix.CPUSet
	if err := unix.SchedGetaffinity(0, &affinity); err == nil {
		affinityCount = affinity.Count()
	}
	return effectiveCPUThreadsFrom(os.ReadFile, runtime.GOMAXPROCS(0), affinityCount)
}

func effectiveCPUThreadsFrom(readFile func(string) ([]byte, error), runtimeLimit, affinityCount int) int {
	effective := max(runtimeLimit, 1)
	if affinityCount > 0 {
		effective = min(effective, affinityCount)
	}
	membership := parseCgroupMembership(readText(readFile, "/proc/self/cgroup"))
	mounts := parseCgroupMounts(readText(readFile, "/proc/self/mountinfo"))
	found := false
	for _, mount := range mounts {
		switch mount.fsType {
		case "cgroup2":
			if membership.v2 == "" {
				continue
			}
			current := resolveCgroupPath(mount, membership.v2)
			if current == "" {
				continue
			}
			found = true
			effective = applyCPUHierarchy(readFile, effective, current, mount.mountPoint, true, true)
		case "cgroup":
			if _, ok := mount.controllers["cpu"]; ok {
				path := membership.controllers["cpu"]
				if path == "" {
					path = membership.controllers["cpuacct"]
				}
				if current := resolveCgroupPath(mount, path); current != "" {
					found = true
					effective = applyCPUHierarchy(readFile, effective, current, mount.mountPoint, true, false)
				}
			}
			if _, ok := mount.controllers["cpuset"]; ok {
				if current := resolveCgroupPath(mount, membership.controllers["cpuset"]); current != "" {
					found = true
					effective = applyCPUHierarchy(readFile, effective, current, mount.mountPoint, false, true)
				}
			}
		}
	}
	if !found {
		// Compatibility fallback for minimal containers that hide mountinfo.
		effective = applyCPUHierarchy(readFile, effective, "/sys/fs/cgroup", "/sys/fs/cgroup", true, true)
		effective = applyCPUHierarchy(readFile, effective, "/sys/fs/cgroup/cpu", "/sys/fs/cgroup/cpu", true, false)
		effective = applyCPUHierarchy(readFile, effective, "/sys/fs/cgroup/cpuset", "/sys/fs/cgroup/cpuset", false, true)
	}
	return max(effective, 1)
}

func applyCPUHierarchy(readFile func(string) ([]byte, error), effective int, current, mountPoint string, quota, cpuset bool) int {
	for _, directory := range cgroupAncestors(current, mountPoint) {
		if cpuset {
			value := strings.TrimSpace(readText(readFile, filepath.Join(directory, "cpuset.cpus.effective")))
			if value == "" {
				value = strings.TrimSpace(readText(readFile, filepath.Join(directory, "cpuset.cpus")))
			}
			if count := countCPUSet(value); count > 0 {
				effective = min(effective, count)
			}
		}
		if quota {
			if limit := readCPUQuota(readFile, directory); limit > 0 {
				effective = min(effective, limit)
			}
		}
	}
	return max(effective, 1)
}

func readCPUQuota(readFile func(string) ([]byte, error), directory string) int {
	if fields := strings.Fields(readText(readFile, filepath.Join(directory, "cpu.max"))); len(fields) >= 2 {
		if fields[0] == "max" {
			return 0
		}
		return quotaThreads(fields[0], fields[1])
	}
	quota := strings.TrimSpace(readText(readFile, filepath.Join(directory, "cpu.cfs_quota_us")))
	period := strings.TrimSpace(readText(readFile, filepath.Join(directory, "cpu.cfs_period_us")))
	if quota == "" || quota == "-1" || period == "" {
		return 0
	}
	return quotaThreads(quota, period)
}

func quotaThreads(quotaValue, periodValue string) int {
	quota, quotaErr := strconv.ParseFloat(strings.TrimSpace(quotaValue), 64)
	period, periodErr := strconv.ParseFloat(strings.TrimSpace(periodValue), 64)
	if quotaErr != nil || periodErr != nil || quota <= 0 || period <= 0 {
		return 0
	}
	return max(int(math.Ceil(quota/period)), 1)
}

func parseCgroupMembership(value string) cgroupMembership {
	result := cgroupMembership{controllers: make(map[string]string)}
	for _, line := range strings.Split(value, "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), ":", 3)
		if len(parts) != 3 || !strings.HasPrefix(parts[2], "/") {
			continue
		}
		if parts[0] == "0" && parts[1] == "" {
			result.v2 = parts[2]
			continue
		}
		for _, controller := range strings.Split(parts[1], ",") {
			controller = strings.TrimSpace(controller)
			if controller != "" {
				result.controllers[controller] = parts[2]
			}
		}
	}
	return result
}

func parseCgroupMounts(value string) []cgroupMount {
	var result []cgroupMount
	for _, line := range strings.Split(value, "\n") {
		fields := strings.Fields(line)
		separator := -1
		for index, field := range fields {
			if field == "-" {
				separator = index
				break
			}
		}
		if separator < 6 || separator+3 >= len(fields) {
			continue
		}
		fsType := fields[separator+1]
		if fsType != "cgroup" && fsType != "cgroup2" {
			continue
		}
		mount := cgroupMount{
			root: decodeMountField(fields[3]), mountPoint: decodeMountField(fields[4]),
			fsType: fsType, controllers: make(map[string]struct{}),
		}
		if fsType == "cgroup" {
			options := fields[5] + "," + fields[separator+3]
			for _, option := range strings.Split(options, ",") {
				switch option {
				case "cpu", "cpuacct", "cpuset":
					mount.controllers[option] = struct{}{}
				}
			}
		}
		result = append(result, mount)
	}
	return result
}

func resolveCgroupPath(mount cgroupMount, membership string) string {
	if membership == "" || !strings.HasPrefix(membership, "/") {
		return ""
	}
	root := filepath.Clean(mount.root)
	membership = filepath.Clean(membership)
	var relative string
	switch {
	case root == "/":
		relative = strings.TrimPrefix(membership, "/")
	case membership == root:
		relative = ""
	case strings.HasPrefix(membership, root+string(filepath.Separator)):
		relative = strings.TrimPrefix(strings.TrimPrefix(membership, root), "/")
	default:
		return ""
	}
	return filepath.Join(mount.mountPoint, relative)
}

func cgroupAncestors(current, mountPoint string) []string {
	current = filepath.Clean(current)
	mountPoint = filepath.Clean(mountPoint)
	if current != mountPoint && !strings.HasPrefix(current, mountPoint+string(filepath.Separator)) {
		return nil
	}
	var result []string
	for {
		result = append(result, current)
		if current == mountPoint {
			break
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return result
}

func readText(readFile func(string) ([]byte, error), path string) string {
	data, err := readFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

func decodeMountField(value string) string {
	replacer := strings.NewReplacer(`\040`, " ", `\011`, "\t", `\134`, `\`)
	return replacer.Replace(value)
}

func countCPUSet(value string) int {
	total := 0
	for _, item := range strings.Split(value, ",") {
		bounds := strings.SplitN(strings.TrimSpace(item), "-", 2)
		if len(bounds) == 0 || bounds[0] == "" {
			continue
		}
		start, err := strconv.Atoi(bounds[0])
		if err != nil || start < 0 {
			continue
		}
		end := start
		if len(bounds) == 2 {
			end, err = strconv.Atoi(bounds[1])
			if err != nil || end < start {
				continue
			}
		}
		total += end - start + 1
	}
	return total
}
