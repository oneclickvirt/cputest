package cpu

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// geekbenchBinsFS embeds the geekbench_bins directory.
// In CI, the directory is populated with geekbench binaries before compilation.
// In local development builds the directory is empty and the code falls back
// to looking for a system-installed geekbench in PATH.
//
//go:embed all:geekbench_bins
var geekbenchBinsFS embed.FS

// extractEmbeddedGeekbench extracts the embedded geekbench files to a new temp
// directory and returns the path to the geekbench binary and the temp directory
// that the caller must remove when done. Returns ("", "", err) when no geekbench
// binary has been embedded (e.g. local / unsupported-platform builds).
func extractEmbeddedGeekbench() (binPath, tmpDir string, err error) {
	entries, err := geekbenchBinsFS.ReadDir("geekbench_bins")
	if err != nil {
		return "", "", err
	}

	hasGeekbench := false
	for _, e := range entries {
		if e.Name() == "geekbench" {
			hasGeekbench = true
			break
		}
	}
	if !hasGeekbench {
		return "", "", fmt.Errorf("no embedded geekbench binary")
	}

	tmpDir, err = os.MkdirTemp("", "cputest-geekbench-*")
	if err != nil {
		return "", "", err
	}

	for _, e := range entries {
		// skip hidden placeholder / ignore files
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		data, readErr := geekbenchBinsFS.ReadFile("geekbench_bins/" + e.Name())
		if readErr != nil {
			os.RemoveAll(tmpDir)
			return "", "", readErr
		}
		dest := filepath.Join(tmpDir, e.Name())
		if writeErr := os.WriteFile(dest, data, 0755); writeErr != nil {
			os.RemoveAll(tmpDir)
			return "", "", writeErr
		}
	}

	return filepath.Join(tmpDir, "geekbench"), tmpDir, nil
}
