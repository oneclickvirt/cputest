package cpu

import (
	"archive/tar"
	"compress/gzip"
	"embed"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
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

// downloadAndExtractGeekbench downloads Geekbench 6 from the official CDN and
// extracts the required files to a fresh temp directory.  It is used as a
// runtime fallback when the embedded binary fails to initialise (e.g. the
// binary embedded at build time is too old for the current platform).
// The caller is responsible for removing tmpDir when done.
func downloadAndExtractGeekbench() (binPath, tmpDir string, err error) {
	const (
		base = "https://cdn.geekbench.com"
		ver  = "6.7.1"
	)

	type variant struct {
		tarURL, launcherPath, helperPath, plarPath string
	}

	variants := map[string]variant{
		"linux-amd64": {
			tarURL:       fmt.Sprintf("%s/Geekbench-%s-Linux.tar.gz", base, ver),
			launcherPath: fmt.Sprintf("Geekbench-%s-Linux/geekbench6", ver),
			helperPath:   fmt.Sprintf("Geekbench-%s-Linux/geekbench_x86_64", ver),
			plarPath:     fmt.Sprintf("Geekbench-%s-Linux/geekbench.plar", ver),
		},
		"linux-arm64": {
			tarURL:       fmt.Sprintf("%s/Geekbench-%s-LinuxARMPreview.tar.gz", base, ver),
			launcherPath: fmt.Sprintf("Geekbench-%s-LinuxARMPreview/geekbench6", ver),
			helperPath:   fmt.Sprintf("Geekbench-%s-LinuxARMPreview/geekbench_aarch64", ver),
			plarPath:     fmt.Sprintf("Geekbench-%s-LinuxARMPreview/geekbench.plar", ver),
		},
		"linux-arm": {
			tarURL:       fmt.Sprintf("%s/Geekbench-%s-LinuxARMPreview.tar.gz", base, "5.5.1"),
			launcherPath: fmt.Sprintf("Geekbench-%s-LinuxARMPreview/geekbench5", "5.5.1"),
			helperPath:   fmt.Sprintf("Geekbench-%s-LinuxARMPreview/geekbench_armv7", "5.5.1"),
			plarPath:     fmt.Sprintf("Geekbench-%s-LinuxARMPreview/geekbench.plar", "5.5.1"),
		},
	}

	key := runtime.GOOS + "-" + runtime.GOARCH
	v, ok := variants[key]
	if !ok {
		return "", "", fmt.Errorf("no geekbench download available for %s", key)
	}

	tmpDir, err = os.MkdirTemp("", "cputest-geekbench-*")
	if err != nil {
		return "", "", err
	}

	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Get(v.tarURL)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", "", fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		os.RemoveAll(tmpDir)
		return "", "", fmt.Errorf("download HTTP %d", resp.StatusCode)
	}

	gr, err := gzip.NewReader(resp.Body)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", "", fmt.Errorf("gzip open: %w", err)
	}
	defer gr.Close()

	destMap := map[string]string{
		v.launcherPath: filepath.Join(tmpDir, "geekbench"),
		v.helperPath:   filepath.Join(tmpDir, filepath.Base(v.helperPath)),
		v.plarPath:     filepath.Join(tmpDir, "geekbench.plar"),
	}
	extracted := 0
	tr := tar.NewReader(gr)
	for {
		hdr, terr := tr.Next()
		if terr == io.EOF {
			break
		}
		if terr != nil {
			os.RemoveAll(tmpDir)
			return "", "", fmt.Errorf("tar read: %w", terr)
		}
		dest, ok := destMap[hdr.Name]
		if !ok {
			continue
		}
		data, rerr := io.ReadAll(tr)
		if rerr != nil {
			os.RemoveAll(tmpDir)
			return "", "", fmt.Errorf("read entry: %w", rerr)
		}
		if werr := os.WriteFile(dest, data, 0755); werr != nil {
			os.RemoveAll(tmpDir)
			return "", "", fmt.Errorf("write %s: %w", dest, werr)
		}
		extracted++
		if extracted == len(destMap) {
			break
		}
	}
	if extracted < len(destMap) {
		os.RemoveAll(tmpDir)
		return "", "", fmt.Errorf("incomplete download: got %d/%d files", extracted, len(destMap))
	}

	return filepath.Join(tmpDir, "geekbench"), tmpDir, nil
}
