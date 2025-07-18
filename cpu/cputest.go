package cpu

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/imroc/req/v3"
	"github.com/oneclickvirt/cputest/model"
	. "github.com/oneclickvirt/defaultset"
)

// logError 记录错误日志
func logError(message string, err error) {
	if model.EnableLoger {
		Logger.Info(message + ": " + err.Error())
	}
}

// SysBenchTest 基于sysbench测试
func SysBenchTest(language, testThread string) string {
	if model.EnableLoger {
		InitLogger()
		defer Logger.Sync()
	}
	// 检查是否是 BSD 系列操作系统
	isBSDSystem := false
	// 检查 /etc/os-release 文件是否存在
	if _, err := os.Stat("/etc/os-release"); err == nil {
		// 读取文件内容
		content, err := os.ReadFile("/etc/os-release")
		if err == nil {
			contentStr := string(content)
			// 检查是否包含 BSD 相关标识
			if strings.Contains(contentStr, "freebsd") ||
				strings.Contains(contentStr, "openbsd") ||
				strings.Contains(contentStr, "netbsd") {
				isBSDSystem = true
			}
		}
	}
	// 如果是 BSD 系统，使用自实现的测试
	if isBSDSystem {
		return runInternalBenchmark(language, testThread)
	}
	// 检查系统是否安装了 sysbench
	comCheck := exec.Command("sysbench", "--version")
	output, err := comCheck.CombinedOutput()
	// 如果系统没有安装 sysbench，使用自实现的测试
	if err != nil {
		return runInternalBenchmark(language, testThread)
	}
	version := string(output)
	result := ""
	// 单线程测试
	singleScore, err := runAndParseSysBench("1", "5", version)
	if err != nil {
		logError("sysbench test single score error", err)
		return ""
	}
	result += formatScoreOutput(language, 1, singleScore)
	// 如果需要多线程测试并且是多核系统
	if testThread == "multi" && runtime.NumCPU() > 1 {
		time.Sleep(1 * time.Second)
		multiScore, err := runAndParseSysBench(fmt.Sprintf("%d", runtime.NumCPU()), "5", version)
		if err != nil {
			logError("sysbench test multi score error", err)
			return result // 返回已有的单线程结果
		}
		result += formatScoreOutput(language, runtime.NumCPU(), multiScore)
	}
	return result
}

// runInternalBenchmark 使用内部实现的基准测试
func runInternalBenchmark(language, testThread string) string {
	config := DefaultConfig()
	result := ""
	// 单线程测试
	if testThread == "single" || testThread == "multi" {
		config.NumThreads = 1
		var singleThreadScore float64
		if runtime.GOOS == "windows" || (runtime.GOOS == "linux" && runtime.GOARCH == "arm") {
			_, singleThreadScore, _ = RunBenchmarkCustom(config)
		} else {
			_, singleThreadScore, _ = RunBenchmark(config)
		}
		result += formatScoreOutput(language, 1, fmt.Sprintf("%.2f", singleThreadScore))
	}
	// 多线程测试（如果需要且是多核系统）
	if testThread == "multi" && runtime.NumCPU() > 1 {
		time.Sleep(1 * time.Second)
		config.NumThreads = runtime.NumCPU()
		var multiThreadScore float64
		if runtime.GOOS == "windows" || (runtime.GOOS == "linux" && runtime.GOARCH == "arm") {
			_, multiThreadScore, _ = RunBenchmarkCustom(config)
		} else {
			_, multiThreadScore, _ = RunBenchmark(config)
		}
		result += formatScoreOutput(language, runtime.NumCPU(), fmt.Sprintf("%.2f", multiThreadScore))
	}
	return result
}

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

// runAndParseSysBench 运行sysbench命令并解析结果
func runAndParseSysBench(threads, timeValue, version string) (string, error) {
	result, err := runSysBenchCommand(threads, timeValue, version)
	if err != nil {
		return "", err
	}
	var score, totalTime, totalEvents string
	// 解析结果
	tempList := strings.Split(result, "\n")
	for _, line := range tempList {
		if strings.Contains(line, "events per second:") {
			temp := strings.Split(line, ":")
			if len(temp) == 2 {
				score = temp[1]
				break
			}
		} else if score == "" && totalTime == "" && strings.Contains(line, "total time:") {
			temp := strings.Split(line, ":")
			if len(temp) == 2 {
				totalTime = strings.ReplaceAll(temp[1], "s", "")
			}
		} else if score == "" && totalEvents == "" && strings.Contains(line, "total number of events:") {
			temp := strings.Split(line, ":")
			if len(temp) == 2 {
				totalEvents = temp[1]
			}
		}
	}
	// 如果没有直接找到分数，但有总事件数和总时间，则计算分数
	if score == "" && totalTime != "" && totalEvents != "" {
		totalEventsFloat, err1 := strconv.ParseFloat(totalEvents, 64)
		if err1 != nil {
			logError("parse total events error", err1)
			return "", err1
		}
		totalTimeFloat, err2 := strconv.ParseFloat(totalTime, 64)
		if err2 != nil {
			logError("parse total time error", err2)
			return "", err2
		}
		scoreFloat := totalEventsFloat / totalTimeFloat
		score = strconv.FormatFloat(scoreFloat, 'f', 2, 64)
	}
	return score, nil
}

// formatScoreOutput 根据语言和线程数格式化输出字符串
func formatScoreOutput(language string, threads int, score string) string {
	if language == "en" {
		return fmt.Sprintf("%d Thread(s) Test: %s\n", threads, score)
	} else {
		if threads == 1 {
			return fmt.Sprintf("1 线程测试(单核)得分: %s\n", score)
		}
		return fmt.Sprintf("%d 线程测试(多核)得分: %s\n", threads, score)
	}
}

// runGeekbenchCommand 执行 geekbench 命令进行测试
func runGeekbenchCommand() (string, error) {
	var command *exec.Cmd
	command = exec.Command("geekbench", "--upload")
	output, err := command.CombinedOutput()
	return string(output), err
}

// GeekBenchTest 调用 geekbench 执行CPU测试
// 调用 geekbench 命令执行
// https://github.com/masonr/yet-another-bench-script/blob/0ad4c4e85694dbcf0958d8045c2399dbd0f9298c/yabs.sh#L894
func GeekBenchTest(language, testThread string) string {
	if model.EnableLoger {
		InitLogger()
		defer Logger.Sync()
	}
	var result, singleScore, multiScore, link string
	comCheck := exec.Command("geekbench", "--version")
	// Geekbench 5.4.5 Tryout Build 503938 (corktown-master-build 6006e737ba)
	output, err := comCheck.CombinedOutput()
	version := string(output)
	if err != nil {
		logError("cannot match geekbench command: ", err)
		return ""
	}
	if strings.Contains(version, "Geekbench 6") {
		// 检测存在 /etc/os-release 文件且含 CentOS Linux 7 时，需要预先下载 GLIBC_2.27 才能使用 geekbench 6
		file, err := os.Open("/etc/os-release")
		defer file.Close()
		if err == nil {
			scanner := bufio.NewScanner(file)
			isCentOS7 := false
			for scanner.Scan() {
				line := scanner.Text()
				if strings.Contains(line, "CentOS Linux 7") {
					isCentOS7 = true
					break
				}
			}
			if err := scanner.Err(); err == nil {
				// 如果文件中包含 CentOS Linux 7，则打印提示信息
				if isCentOS7 && language == "zh" {
					return "需要预先下载 GLIBC_2.27 才能使用 geekbench 6"
				} else if isCentOS7 && language != "zh" {
					return "You need to pre-download GLIBC_2.27 to use geekbench 6."
				}
			}
		}
	}
	tp, err := runGeekbenchCommand()
	if err != nil {
		logError("run geekbench command error: ", err)
		return ""
	}
	// 解析 geekbench 执行结果
	tempList := strings.Split(tp, "\n")
	for _, line := range tempList {
		if strings.Contains(line, "https://browser.geekbench.com") && strings.Contains(line, "cpu") {
			link = strings.TrimSpace(line)
			break
		}
	}
	const (
		userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.75 Safari/537.36"
		accept    = "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9"
		referer   = "browser.geekbench.com"
	)
	client := req.DefaultClient()
	client.SetTimeout(6 * time.Second)
	client.SetCommonHeader("User-Agent", userAgent)
	client.SetCommonHeader("Accept", accept)
	client.SetCommonHeader("Referer", referer)
	resp, err := client.R().Get(link)
	if err != nil {
		logError("geekbench test link error: ", err)
		return ""
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		logError("read response body error: ", err)
		return ""
	}
	body := string(b)
	if resp.StatusCode != http.StatusOK {
		if model.EnableLoger {
			Logger.Info("geekbench test status code not OK")
		}
		return ""
	}
	doc, readErr := goquery.NewDocumentFromReader(strings.NewReader(body))
	if readErr != nil {
		logError("parse response body error: ", err)
		return ""
	}
	textContent := doc.Find(".table-wrapper.cpu").Text()
	resList := strings.Split(textContent, "\n")
	for index, l := range resList {
		if strings.Contains(l, "Single-Core") {
			singleScore = resList[index-1]
		} else if strings.Contains(l, "Multi-Core") {
			multiScore = resList[index-1]
		}
	}
	if link != "" && singleScore != "" {
		result += strings.TrimSpace(strings.ReplaceAll(version, "\n", "")) + "\n"
		result += "Single-Core Score: " + singleScore + "\n"
		if multiScore != "" {
			result += "Multi-Core Score: " + multiScore + "\n"
		}
		result += "Link: " + link + "\n"
	}
	return result
}

func WinsatTest(language, testThread string) string {
	if model.EnableLoger {
		InitLogger()
		defer Logger.Sync()
	}
	var result string
	cmd1 := exec.Command("winsat", "cpu", "-encryption") // winsat cpu -encryption
	output1, err1 := cmd1.Output()
	if err1 != nil {
		logError("winsat cpu encryption error: ", err1)
		return runInternalBenchmark(language, testThread)
	} else if strings.Contains(strings.ToLower(string(output1)), "error") ||
		strings.Contains(string(output1), "错误") {
		return runInternalBenchmark(language, testThread)
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
		logError("winsat cpu compression error: ", err2)
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
