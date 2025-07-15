package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/oneclickvirt/cputest/cpu"
	"github.com/oneclickvirt/cputest/model"
	. "github.com/oneclickvirt/defaultset"
)

func main() {
	go func() {
		http.Get("https://hits.spiritlhl.net/cputest.svg?action=hit&title=Hits&title_bg=%23555555&count_bg=%230eecf8&edge_flat=false")
	}()
	fmt.Println(Green("项目地址:"), Yellow("https://github.com/oneclickvirt/cputest"))
	var showVersion, help bool
	var language, testMethod, testThreadMode string
	cputestFlag := flag.NewFlagSet("cputest", flag.ContinueOnError)
	cputestFlag.BoolVar(&help, "h", false, "Show help information")
	cputestFlag.BoolVar(&showVersion, "v", false, "show version")
	cputestFlag.StringVar(&language, "l", "", "Language parameter (en or zh)")
	cputestFlag.StringVar(&testMethod, "m", "", "Specific Test Method (sysbench or geekbench)")
	cputestFlag.StringVar(&testThreadMode, "t", "", "Specific Test Thread Mode (single or multi)")
	cputestFlag.BoolVar(&model.EnableLoger, "log", false, "Enable logging")
	cputestFlag.Parse(os.Args[1:])
	if help {
		fmt.Printf("Usage: %s [options]\n", os.Args[0])
		cputestFlag.PrintDefaults()
		return
	}
	if showVersion {
		fmt.Println(model.CpuTestVersion)
		return
	}
	var res string
	language = strings.ToLower(language)
	if testMethod == "" || strings.ToLower(testMethod) == "sysbench" {
		testMethod = "sysbench"
	} else if strings.ToLower(testMethod) == "geekbench" {
		testMethod = "geekbench"
	}
	if testThreadMode == "" || strings.ToLower(testThreadMode) == "single" {
		testThreadMode = "single"
	} else {
		testThreadMode = strings.TrimSpace(strings.ToLower(testThreadMode))
	}
	if runtime.GOOS == "windows" {
		if testMethod != "winsat" && testMethod != "" {
			res = "Detected host is Windows, using Winsat for testing.\n"
		}
		res += cpu.WinsatTest(language, testThreadMode)
	} else {
		switch testMethod {
		case "sysbench":
			res = cpu.SysBenchTest(language, testThreadMode)
			if res == "" {
				res = "Sysbench test failed, switching to Geekbench for testing.\n"
				res += cpu.GeekBenchTest(language, testThreadMode)
			}
		case "geekbench":
			res = cpu.GeekBenchTest(language, testThreadMode)
			if res == "" {
				res = "Geekbench test failed, switching to Sysbench for testing.\n"
				res += cpu.SysBenchTest(language, testThreadMode)
			}
		default:
			res = "Invalid test method specified.\n"
		}
	}
	fmt.Println("--------------------------------------------------")
	fmt.Print(res)
	fmt.Println("--------------------------------------------------")
}
