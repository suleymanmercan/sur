package checker

import (
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
