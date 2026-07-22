package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/oneclickvirt/cputest/cpu"
	"github.com/oneclickvirt/cputest/model"
	. "github.com/oneclickvirt/defaultset"
)

type cliOptions struct {
	help, version, jsonOutput, log   bool
	language, testMethod, threadMode string
	duration                         time.Duration
	threads                          int
	languageSet, methodSet           bool
	threadModeSet                    bool
	durationSet, threadsSet          bool
}

func parseCLI(args []string) (cliOptions, error) {
	opts := cliOptions{}
	fs := newFlagSet(&opts, io.Discard)
	if err := fs.Parse(args); err != nil {
		return opts, err
	}
	if fs.NArg() != 0 {
		return opts, fmt.Errorf("unexpected positional arguments: %s", strings.Join(fs.Args(), " "))
	}
	fs.Visit(func(current *flag.Flag) {
		switch current.Name {
		case "l":
			opts.languageSet = true
		case "m":
			opts.methodSet = true
		case "t":
			opts.threadModeSet = true
		case "duration":
			opts.durationSet = true
		case "threads":
			opts.threadsSet = true
		}
	})
	opts.language = strings.ToLower(strings.TrimSpace(opts.language))
	opts.testMethod = strings.ToLower(strings.TrimSpace(opts.testMethod))
	opts.threadMode = strings.ToLower(strings.TrimSpace(opts.threadMode))
	if opts.help || opts.version {
		return opts, nil
	}
	if opts.language != "" && opts.language != "en" && opts.language != "zh" {
		return opts, fmt.Errorf("language must be en or zh")
	}
	if opts.testMethod != "" && opts.testMethod != "sysbench" && opts.testMethod != "geekbench" && !(runtime.GOOS == "windows" && opts.testMethod == "winsat") {
		return opts, fmt.Errorf("unsupported CPU test method")
	}
	if opts.threadMode != "" && opts.threadMode != "single" && opts.threadMode != "multi" {
		return opts, fmt.Errorf("thread mode must be single or multi")
	}
	if opts.jsonOutput {
		if opts.languageSet || opts.methodSet {
			return opts, fmt.Errorf("-l and -m are not used with structured output")
		}
		if opts.durationSet && (opts.duration <= 0 || opts.duration > 20*time.Second) {
			return opts, fmt.Errorf("structured duration must be greater than zero and at most 20s")
		}
		if opts.threadsSet && opts.threads <= 0 {
			return opts, fmt.Errorf("structured threads must be positive")
		}
		if opts.threadsSet && opts.threadModeSet {
			return opts, fmt.Errorf("-threads and -t cannot be combined")
		}
	} else if opts.durationSet || opts.threadsSet {
		return opts, fmt.Errorf("-duration and -threads require structured output")
	}
	return opts, nil
}

func newFlagSet(opts *cliOptions, output io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet("cputest", flag.ContinueOnError)
	fs.SetOutput(output)
	fs.BoolVar(&opts.help, "h", false, "Show help information")
	fs.BoolVar(&opts.version, "v", false, "show version")
	fs.StringVar(&opts.language, "l", "", "Language parameter (en or zh)")
	fs.StringVar(&opts.testMethod, "m", "", "Specific Test Method (sysbench or geekbench)")
	fs.StringVar(&opts.threadMode, "t", "", "Specific Test Thread Mode (single or multi)")
	fs.BoolVar(&opts.log, "log", false, "Enable logging")
	fs.BoolVar(&opts.jsonOutput, "json", false, "Print the Go structured CPU result as JSON")
	fs.BoolVar(&opts.jsonOutput, "structured", false, "Print the Go structured CPU result as JSON")
	fs.DurationVar(&opts.duration, "duration", 0, "Structured benchmark duration (for example 5s)")
	fs.IntVar(&opts.threads, "threads", 0, "Structured benchmark worker count")
	return fs
}

func printCLIHelp(program string) {
	fmt.Printf("Usage: %s [options]\n", program)
	newFlagSet(&cliOptions{}, os.Stdout).PrintDefaults()
}

func selectCLIAction(opts cliOptions) string {
	if opts.help {
		return "help"
	}
	if opts.version {
		return "version"
	}
	if opts.jsonOutput {
		return "structured"
	}
	return "legacy"
}

func main() {
	opts, err := parseCLI(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, sanitizeErrorText(err.Error()))
		os.Exit(2)
	}
	model.EnableLoger = opts.log
	action := selectCLIAction(opts)
	if action == "help" || action == "version" {
		printLegacyHeader()
		if action == "help" {
			printCLIHelp(os.Args[0])
			return
		}
		fmt.Println(model.CpuTestVersion)
		return
	}
	if action == "structured" {
		threads := opts.threads
		if threads == 0 {
			threads = 1
			if strings.EqualFold(strings.TrimSpace(opts.threadMode), "multi") {
				threads = runtime.NumCPU()
			}
		}
		duration := opts.duration
		if duration <= 0 {
			duration = 5 * time.Second
		}
		ctx := context.Background()
		result := cpu.RunStructured(ctx, cpu.StructuredConfig{Threads: threads, Duration: duration})
		encoded, marshalErr := json.Marshal(result)
		if marshalErr != nil {
			fmt.Fprintln(os.Stderr, marshalErr)
			return
		}
		fmt.Println(string(encoded))
		if result.Status != "ok" {
			os.Exit(1)
		}
		return
	}
	printLegacyHeader()
	language, testMethod, testThreadMode := opts.language, opts.testMethod, opts.threadMode
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
	fmt.Println(" --------------------------------------------------")
	fmt.Print(indentLegacyOutput(res))
	fmt.Println(" --------------------------------------------------")
}

func printLegacyHeader() {
	go func() {
		http.Get("https://hits.spiritlhl.net/cputest.svg?action=hit&title=Hits&title_bg=%23555555&count_bg=%230eecf8&edge_flat=false")
	}()
	fmt.Println(" "+Green("项目地址:"), Yellow("https://github.com/oneclickvirt/cputest"))
}
