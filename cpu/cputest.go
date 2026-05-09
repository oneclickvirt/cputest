package cpu

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/imroc/req/v3"
	"github.com/oneclickvirt/cputest/model"
	. "github.com/oneclickvirt/defaultset"
)

type Config struct {
	MaxPrime   int
	Duration   time.Duration
	NumThreads int
	MaxEvents  int
}

func DefaultConfig() Config {
	return Config{
		MaxPrime:   10000,
		Duration:   5 * time.Second,
		NumThreads: 1,
		MaxEvents:  1000000,
	}
}

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
	isBSDSystem := runtime.GOOS == "freebsd" || runtime.GOOS == "openbsd" || runtime.GOOS == "netbsd"
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
	if err != nil || singleScore == "" {
		logError("sysbench test single score error", fmt.Errorf("score extraction failed"))
		return runInternalBenchmark(language, testThread)
	}
	result += formatScoreOutput(language, 1, singleScore)
	// 如果需要多线程测试并且是多核系统
	if testThread == "multi" && runtime.NumCPU() > 1 {
		time.Sleep(1 * time.Second)
		multiScore, err := runAndParseSysBench(fmt.Sprintf("%d", runtime.NumCPU()), "5", version)
		if err != nil || multiScore == "" {
			logError("sysbench test multi score error", fmt.Errorf("score extraction failed"))
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
		_, singleThreadScore, _ = RunBenchmark(config)
		result += formatScoreOutput(language, 1, fmt.Sprintf("%.2f", singleThreadScore))
	}
	// 多线程测试（如果需要且是多核系统）
	if testThread == "multi" && runtime.NumCPU() > 1 {
		time.Sleep(1 * time.Second)
		config.NumThreads = runtime.NumCPU()
		var multiThreadScore float64
		_, multiThreadScore, _ = RunBenchmark(config)
		result += formatScoreOutput(language, runtime.NumCPU(), fmt.Sprintf("%.2f", multiThreadScore))
	}
	return result
}

// isNewSysbenchFormat 判断 sysbench 是否使用新的命令行格式（>= 1.0.18）
func isNewSysbenchFormat(version string) bool {
	// version string example: "sysbench 1.0.20 (using system LuaJIT 2.1.0-beta3)"
	fields := strings.Fields(version)
	if len(fields) < 2 {
		return false
	}
	parts := strings.Split(fields[1], ".")
	if len(parts) < 3 {
		return false
	}
	major, errM := strconv.Atoi(parts[0])
	minor, errMi := strconv.Atoi(parts[1])
	patch, errP := strconv.Atoi(parts[2])
	if errM != nil || errMi != nil || errP != nil {
		return false
	}
	return major > 1 || (major == 1 && minor > 0) || (major == 1 && minor == 0 && patch >= 18)
}

// runSysBenchCommand 执行 sysbench 命令进行测试
func runSysBenchCommand(numThreads, maxTime, version string) (string, error) {
	// version <= 1.0.17
	// sysbench --test=cpu --num-threads=1 --cpu-max-prime=10000 --max-requests=1000000 --max-time=5 run
	// version >= 1.0.18
	// sysbench cpu --threads=1 --cpu-max-prime=10000 --events=1000000 --time=5 run
	var command *exec.Cmd
	if isNewSysbenchFormat(version) {
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
				score = strings.TrimSpace(temp[1])
				break
			}
		} else if score == "" && totalTime == "" && strings.Contains(line, "total time:") {
			temp := strings.Split(line, ":")
			if len(temp) == 2 {
				// 去掉末尾的 "s" 时间单位，同时去除首尾空格
				totalTime = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(temp[1]), "s"))
			}
		} else if score == "" && totalEvents == "" && strings.Contains(line, "total number of events:") {
			temp := strings.Split(line, ":")
			if len(temp) == 2 {
				totalEvents = strings.TrimSpace(temp[1])
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
	if score == "" {
		return "", fmt.Errorf("could not extract sysbench score from output")
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

// fileExists reports whether a file exists and is not a directory.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// resolveGeekbenchBinary returns the path to the geekbench binary and, when the
// embedded binary is used, the temp directory that the caller must remove.
// It checks the system PATH first, then falls back to the embedded binary.
func resolveGeekbenchBinary() (binPath, tmpDir string, err error) {
	if path, lookErr := exec.LookPath("geekbench"); lookErr == nil {
		return path, "", nil
	}
	return extractEmbeddedGeekbench()
}

// fetchGeekbenchScores fetches single-core and multi-core scores from the
// geekbench results page. Returns empty strings on any error so callers can
// still display the link even when score extraction fails.
func fetchGeekbenchScores(link string) (singleScore, multiScore string) {
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
		logError("geekbench fetch scores error", err)
		return "", ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if model.EnableLoger {
			Logger.Info("geekbench results page status code not OK")
		}
		return "", ""
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		logError("geekbench fetch scores read body error", err)
		return "", ""
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(b)))
	if err != nil {
		logError("geekbench fetch scores parse body error", err)
		return "", ""
	}
	resList := strings.Split(doc.Find(".table-wrapper.cpu").Text(), "\n")
	for i, l := range resList {
		if i == 0 {
			continue
		}
		if strings.Contains(l, "Single-Core") {
			singleScore = strings.TrimSpace(resList[i-1])
		} else if strings.Contains(l, "Multi-Core") {
			multiScore = strings.TrimSpace(resList[i-1])
		}
	}
	return singleScore, multiScore
}

// GeekBenchTest 调用 geekbench 执行CPU测试
// https://github.com/masonr/yet-another-bench-script/blob/0ad4c4e85694dbcf0958d8045c2399dbd0f9298c/yabs.sh#L894
func GeekBenchTest(language, testThread string) string {
	if model.EnableLoger {
		InitLogger()
		defer Logger.Sync()
	}

	// Resolve geekbench binary: PATH first, then embedded binary.
	geekbenchBin, tmpDir, err := resolveGeekbenchBinary()
	if err != nil || geekbenchBin == "" {
		logError("cannot find geekbench binary", fmt.Errorf("not in PATH and not embedded"))
		return ""
	}
	if tmpDir != "" {
		defer os.RemoveAll(tmpDir)
	}

	// Detect version.
	// e.g. "Geekbench 5.4.5 Tryout Build 503938 (corktown-master-build 6006e737ba)"
	// NOTE: Some geekbench builds return a non-zero exit code (e.g. 255) even
	// for --version on certain VPS/headless platforms (license checks, missing
	// display libraries, etc.). Capture the output regardless and continue;
	// the actual --upload call will surface a real failure if the binary is
	// truly unusable on this system.
	versionOut, versionErr := exec.Command(geekbenchBin, "--version").CombinedOutput()
	version := strings.TrimSpace(string(versionOut))
	if versionErr != nil {
		logError("geekbench version check warning", versionErr)
		if version == "" {
			// No output at all – infer from adjacent helper binaries embedded
			// alongside the launcher to pick the right display label.
			dir := filepath.Dir(geekbenchBin)
			switch {
			case fileExists(filepath.Join(dir, "geekbench_x86_64")):
				version = "Geekbench 6"
			case fileExists(filepath.Join(dir, "geekbench_aarch64")) && !fileExists(filepath.Join(dir, "geekbench_armv7")):
				version = "Geekbench 6"
			default:
				version = "Geekbench"
			}
		}
	}

	// Geekbench 6 cannot run on CentOS 7 without GLIBC_2.27.
	if strings.Contains(version, "Geekbench 6") {
		if file, openErr := os.Open("/etc/os-release"); openErr == nil {
			defer file.Close()
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				if strings.Contains(scanner.Text(), "CentOS Linux 7") {
					if language == "zh" {
						return "需要预先下载 GLIBC_2.27 才能使用 geekbench 6"
					}
					return "You need to pre-download GLIBC_2.27 to use geekbench 6."
				}
			}
		}
	}

	// Run the benchmark and capture the results URL.
	runOut, err := exec.Command(geekbenchBin, "--upload").CombinedOutput()
	if err != nil {
		logError("run geekbench command error", err)
		return ""
	}

	var link string
	for _, line := range strings.Split(string(runOut), "\n") {
		if strings.Contains(line, "https://browser.geekbench.com") && strings.Contains(line, "cpu") {
			link = strings.TrimSpace(line)
			break
		}
	}
	if link == "" {
		return ""
	}

	// Always output version and link.
	result := strings.TrimSpace(strings.ReplaceAll(version, "\n", "")) + "\n"

	// Try to fetch scores; show them when available but never suppress the link.
	singleScore, multiScore := fetchGeekbenchScores(link)
	if singleScore != "" {
		result += "Single-Core Score: " + singleScore + "\n"
	}
	if multiScore != "" {
		result += "Multi-Core Score: " + multiScore + "\n"
	}
	result += "Link: " + link + "\n"

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
				tempL := strings.Fields(l)
				if len(tempL) >= 2 {
					tempText := tempL[len(tempL)-2]
					if language == "en" {
						result += "CPU AES256 encrypt: "
					} else {
						result += "CPU AES256 加密: "
					}
					result += tempText + "MB/s" + "\n"
				}
			}
		}
	}
	cmd2 := exec.Command("winsat", "cpu", "-compression")
	output2, err2 := cmd2.Output()
	if err2 != nil {
		logError("winsat cpu compression error: ", err2)
		return result
	} else {
		tempList := strings.Split(string(output2), "\n")
		for _, l := range tempList {
			if strings.Contains(l, "CPU LZW") {
				tempL := strings.Fields(l)
				if len(tempL) >= 2 {
					tempText := tempL[len(tempL)-2]
					if language == "en" {
						result += "CPU LZW Compression: "
					} else {
						result += "CPU LZW 压缩: "
					}
					result += tempText + "MB/s" + "\n"
				}
			}
		}
	}
	return result
}
