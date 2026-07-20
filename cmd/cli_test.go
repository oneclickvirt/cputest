package main

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestParseCLIOptions(t *testing.T) {
	opts, err := parseCLI([]string{"--structured", "--duration", "250ms", "--threads", "3", "-t", "multi"})
	if err != nil {
		t.Fatalf("parseCLI returned error: %v", err)
	}
	if !opts.jsonOutput || opts.duration != 250*time.Millisecond || opts.threads != 3 || opts.threadMode != "multi" {
		t.Fatalf("unexpected options: %#v", opts)
	}
}

func TestHelpRetainsLegacyFlags(t *testing.T) {
	var output bytes.Buffer
	newFlagSet(&cliOptions{}, &output).PrintDefaults()
	for _, legacy := range []string{"-h", "-l string", "-m string", "-t string", "-log", "-v"} {
		if !strings.Contains(output.String(), legacy) {
			t.Fatalf("help is missing legacy flag %q: %s", legacy, output.String())
		}
	}
}

func TestParseCLIRejectsNegativeValues(t *testing.T) {
	if _, err := parseCLI([]string{"--threads", "-1"}); err == nil {
		t.Fatal("expected negative threads to be rejected")
	}
}

func TestCLIActionPrioritizesHelpAndVersion(t *testing.T) {
	if got := selectCLIAction(cliOptions{help: true, version: true, jsonOutput: true}); got != "help" {
		t.Fatalf("help action = %q", got)
	}
	if got := selectCLIAction(cliOptions{version: true, jsonOutput: true}); got != "version" {
		t.Fatalf("version action = %q", got)
	}
}
