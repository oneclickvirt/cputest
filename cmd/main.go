package main

import (
	"flag"
	"fmt"
	"net/http"
	"runtime"
	"strings"

	"github.com/oneclickvirt/cputest/cpu"
	. "github.com/oneclickvirt/defaultset"
)

func main() {
	go func() {
		http.Get("https://hits.seeyoufarm.com/api/count/incr/badge.svg?url=https%3A%2F%2Fgithub.com%2Foneclickvirt%2Fcputest&count_bg=%2323E01C&title_bg=%23555555&icon=sonarcloud.svg&icon_color=%23E7E7E7&title=hits&edge_flat=false")
	}()
	fmt.Println(Green("项目地址:"), Yellow("https://github.com/oneclickvirt/cputest"))
	var showVersion bool
	flag.BoolVar(&showVersion, "v", false, "show version")
	languagePtr := flag.String("l", "", "Language parameter (en or zh)")
	testMethodPtr := flag.String("m", "", "Specific Test Method (sysbench or geekbench)")
	testThreadsPtr := flag.String("t", "", "Specific Test Threads (single or multi)")
	flag.Parse()
	if showVersion {
		fmt.Println(cpu.CpuTestVersion)
		return
	}
	var language, res, testMethod, testThread string
	if *languagePtr == "" {
		language = "zh"
	} else {
		language = strings.ToLower(*languagePtr)
	}
	if *testMethodPtr == "" || *testMethodPtr == "sysbench" {
		testMethod = "sysbench"
	} else if *testMethodPtr == "geekbench" {
		testMethod = "geekbench"
	}
	if *testThreadsPtr == "" || *testThreadsPtr == "single" {
		testThread = "single"
	} else {
		testThread = strings.TrimSpace(strings.ToLower(*testThreadsPtr))
	}
	if runtime.GOOS == "windows" {
		res = cpu.WinsatTest(language, testThread)
	} else {
		if testMethod == "sysbench" {
			res = cpu.SysBenchTest(language, testThread)
			if res == "" {
				res = "sysbench test failed, switch to use dd test.\n"
				res += cpu.GeekBenchTest(language, testThread)
			}
		} else if testMethod == "geekbench" {
			res = cpu.GeekBenchTest(language, testThread)
		}
	}
	fmt.Println("--------------------------------------------------")
	fmt.Printf(res)
	fmt.Println("--------------------------------------------------")
}
