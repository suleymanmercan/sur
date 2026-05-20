package checker

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/suleymanmercan/sur/internal/common"
)

func TestBuildReport_Scoring(t *testing.T) {
	r := BuildReport([]common.Finding{
		{Status: common.StatusPass},
		{Status: common.StatusFail, Severity: common.SeverityHigh},
		{Status: common.StatusFail, Severity: common.SeverityMed},
		{Status: common.StatusWarn},
	})
	if r.Issues != 3 {
		t.Fatalf("issues = %d, want 3", r.Issues)
	}
	want := 100 - 15 - 8 - 3
	if r.Score != want {
		t.Fatalf("score = %d, want %d", r.Score, want)
	}
}

func TestBuildReport_FloorAtZero(t *testing.T) {
	var f []common.Finding
	for i := 0; i < 20; i++ {
		f = append(f, common.Finding{Status: common.StatusFail, Severity: common.SeverityHigh})
	}
	r := BuildReport(f)
	if r.Score != 0 {
		t.Fatalf("score should clamp to 0, got %d", r.Score)
	}
}

func TestSshdConfigValue(t *testing.T) {
	cfg := `
# comment
Port 2222
PermitRootLogin no
PasswordAuthentication yes
`
	if v := sshdConfigValue(cfg, "Port"); v != "2222" {
		t.Fatalf("Port = %q", v)
	}
	if v := sshdConfigValue(cfg, "PermitRootLogin"); v != "no" {
		t.Fatalf("PermitRootLogin = %q", v)
	}
	if v := sshdConfigValue(cfg, "Missing"); v != "" {
		t.Fatalf("Missing = %q", v)
	}
}

// TestSshdConfigValue_LastWins verifies that when the same key appears
// multiple times the last occurrence wins (matching sshd include order).
func TestSshdConfigValue_LastWins(t *testing.T) {
	cfg := "PasswordAuthentication yes\nPasswordAuthentication no\n"
	if v := sshdConfigValue(cfg, "PasswordAuthentication"); v != "no" {
		t.Fatalf("expected last value 'no', got %q", v)
	}
}

// TestSshdConfigValueFromFiles_FallbackToMain verifies that when no
// sshd_config.d drop-in files exist, the main config value is returned.
func TestSshdConfigValueFromFiles_FallbackToMain(t *testing.T) {
	origGlob := sshdDropinGlob
	t.Cleanup(func() { sshdDropinGlob = origGlob })

	tempDir := t.TempDir()
	sshdDropinGlob = filepath.Join(tempDir, "*.conf")

	mainCfg := "PasswordAuthentication yes\n"
	val := sshdConfigValueFromFiles(mainCfg, "PasswordAuthentication")
	if val != "yes" {
		t.Fatalf("expected 'yes' from main config, got %q", val)
	}
}

// TestSshdConfigValueFromFiles_DropinFile writes an actual drop-in file
// and verifies it overrides the main config value.
func TestSshdConfigValueFromFiles_DropinFile(t *testing.T) {
	origGlob := sshdDropinGlob
	t.Cleanup(func() { sshdDropinGlob = origGlob })

	tempDir := t.TempDir()
	sshdDropinGlob = filepath.Join(tempDir, "*.conf")

	tmpFile := filepath.Join(tempDir, "99-sur-test.conf")
	if err := os.WriteFile(tmpFile, []byte("PermitRootLogin no\n"), 0o644); err != nil {
		t.Fatalf("cannot write to temp file: %v", err)
	}

	mainCfg := "PermitRootLogin yes\n"
	val := sshdConfigValueFromFiles(mainCfg, "PermitRootLogin")
	if val != "no" {
		t.Fatalf("drop-in should override to 'no', got %q", val)
	}
}
