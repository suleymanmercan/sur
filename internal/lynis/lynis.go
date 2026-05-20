// Package lynis is a thin wrapper around the lynis auditor.
// sur shells out to lynis, parses its report file and emits Findings.
package lynis

import (
	"bufio"
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"

	"github.com/suleymanmercan/sur/internal/common"
)

// ErrNotInstalled is returned when the lynis binary is not on PATH.
var ErrNotInstalled = errors.New("lynis is not installed")

// IsInstalled reports whether lynis is on PATH.
func IsInstalled() bool {
	_, err := exec.LookPath("lynis")
	return err == nil
}

// Run executes `lynis audit system --quiet` and returns parsed findings.
// reportPath is where lynis writes its machine-readable report.
func Run(ctx context.Context, reportPath string) ([]common.Finding, error) {
	if !IsInstalled() {
		return nil, ErrNotInstalled
	}
	if reportPath == "" {
		reportPath = "/tmp/sur-lynis.dat"
	}
	_ = os.Remove(reportPath)

	cmd := exec.CommandContext(ctx, "lynis", "audit", "system", // #nosec G204 -- "lynis" is a fixed well-known binary, not user-controlled
		"--quiet", "--no-colors", "--report-file", reportPath)
	// lynis returns non-zero for warnings — we don't treat that as fatal.
	_ = cmd.Run()

	return ParseReport(reportPath)
}

// ParseReport parses the key=value report file lynis writes.
// Documented keys: https://cisofy.com/documentation/lynis/get-started/#reading-the-report-file
func ParseReport(path string) ([]common.Finding, error) {
	f, err := os.Open(path) // #nosec G304 -- path is the lynis report file written by lynis itself to a controlled location
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var findings []common.Finding
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)

		switch key {
		case "warning[]":
			findings = append(findings, parseLynisIssue(val, common.StatusFail, common.SeverityMed))
		case "suggestion[]":
			findings = append(findings, parseLynisIssue(val, common.StatusWarn, common.SeverityLow))
		}
	}
	return findings, scanner.Err()
}

// parseLynisIssue extracts the test-id and description from a pipe-separated record.
// Example: "AUTH-9229|Password file consistency check|description text||"
func parseLynisIssue(raw string, status common.Status, sev common.Severity) common.Finding {
	parts := strings.Split(raw, "|")
	id, title, detail := "", raw, ""
	if len(parts) >= 1 {
		id = parts[0]
	}
	if len(parts) >= 2 {
		title = strings.TrimSpace(parts[1])
	}
	if len(parts) >= 3 {
		detail = strings.TrimSpace(parts[2])
	}
	return common.Finding{
		ID: "lynis." + id, Category: "Lynis",
		Title: title, Status: status, Severity: sev,
		Detail: detail, Source: "lynis",
	}
}
