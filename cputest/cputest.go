package cputest

import (
	"os/exec"
	"strings"
	"runtime"
	"fmt"
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

func SysBenchTest(language, testThread string) string {
	var result, singleScore, multiScore string
	comCheck := exec.Command("sysbench", "--version")
	output, err := comCheck.CombinedOutput()
	if err == nil {
		version := string(output)
		if testThread == "single" {
			singleResult, err := runSysBenchCommand("1", "5", version)
			if err == nil {
				tempList := strings.Split(singleResult, "\n")
				for _, line := range tempList {
					if strings.Contains(line, "events per second:") || strings.Contains(line, "total number of events:") {
						temp1 := strings.Split(line, ":")
						if len(temp1) == 2 {
							singleScore = temp1[1]
						}
					}
				}
			}
			if language == "en" {
				result += "1 Thread(s) Test: "
			} else {
				result += "1 线程测试(单核)得分: "
			}
			result += singleScore + "\n"
		} else if testThread == "multi" {
			singleResult, err := runSysBenchCommand("1", "5", version)
			if err == nil {
				tempList := strings.Split(singleResult, "\n")
				for _, line := range tempList {
					if strings.Contains(line, "events per second:") || strings.Contains(line, "total number of events:") {
						temp1 := strings.Split(line, ":")
						if len(temp1) == 2 {
							singleScore = temp1[1]
						}
					}
				}
			}
			if language == "en" {
				result += "1 Thread(s) Test: "
			} else {
				result += "1 线程测试(单核)得分: "
			}
			result += singleScore + "\n"
			multiResult, err := runSysBenchCommand(fmt.Sprintf("%s", runtime.NumCPU()), "5", version)
			if err == nil {
				tempList := strings.Split(multiResult, "\n")
				for _, line := range tempList {
					if strings.Contains(line, "events per second:") || strings.Contains(line, "total number of events:") {
						temp1 := strings.Split(line, ":")
						if len(temp1) == 2 {
							multiScore = temp1[1]
						}
					}
				}
			}
			if language == "en" {
				result += fmt.Sprintf("%s", runtime.NumCPU()) + " Thread(s) Test: "
			} else {
				result += fmt.Sprintf("%s", runtime.NumCPU()) + " 线程测试(多核)得分: "
			}
			result += multiScore + "\n"
		}
	} else {
		return ""
	}
	return result
}

func GeekBenchTest(language, testThread string) string {
	var result string
	// https://github.com/masonr/yet-another-bench-script/blob/0ad4c4e85694dbcf0958d8045c2399dbd0f9298c/yabs.sh#L894
	return result
}

func WinsatTest(language, testThread string) string {
	var result string
	cmd1 := exec.Command("winsat", "cpu", "-encryption")
	output1, err1 := cmd1.Output()
	if err1 != nil {
		return ""
	} else {
		tempList := strings.Split(string(output1), "\n")
		for _, l := range tempList {
			if strings.Contains(l, "CPU AES256") {
				tempL := strings.Split(l, " ")
				tempText := strings.TrimSpace(tempL[len(tempL)-2])
				if language == "en" {
					result += "CPU AES256 encrypt: "
				} else {
					result += "CPU AES256 加密: "
				}
				result += tempText + "MB/s" + "\n"
			}
		}
	}
	cmd2 := exec.Command("winsat", "cpu", "-compression")
	output2, err2 := cmd2.Output()
	if err2 != nil {
		return ""
	} else {
		tempList := strings.Split(string(output2), "\n")
		for _, l := range tempList {
			if strings.Contains(l, "CPU LZW") {
				tempL := strings.Split(l, " ")
				tempText := strings.TrimSpace(tempL[len(tempL)-2])
				if language == "en" {
					result += "CPU LZW Compression: "
				} else {
					result += "CPU LZW 压缩: "
				}
				result += tempText + "MB/s" + "\n"
			}
		}
	}
	return result
}
