package cputest

import (
	"os/exec"
	"strings"
)

// runSysBenchCommand 执行 sysbench 命令进行测试
func runSysBenchCommand(numThreads, maxTime, version string) (string, error) {
	// version <= 1.0.17
	// sysbench --test=cpu --num-threads=1 --cpu-max-prime=10000 --max-requests=1000000 --max-time=5 run
	// version >= 1.0.18
	// sysbench cpu --threads=1 --cpu-max-prime=10000 --events=1000000 --time=5 run
	var command *exec.Cmd
	if strings.Contains(version, "1.0.18") || strings.Contains(version, "1.0.19") || strings.Contains(version, "1.0.20") {
		command = exec.Command("sysbench", "cpu", "--threads="+numThreads, "--cpu-max-prime=10000", "--events=1000000", "--time="+maxTime, "run")
	} else {
		command = exec.Command("sysbench", "--test=cpu", "--num-threads="+numThreads, "--cpu-max-prime=10000", "--max-requests=1000000", "--max-time="+maxTime, "run")
	}
	output, err := command.CombinedOutput()
	return string(output), err
}
