package checker

import (
	"testing"

	"github.com/suleymanmercan/sur/internal/common"
	"github.com/suleymanmercan/sur/internal/osdetect"
)

func TestAnnotateRemediationHints_AutoFix(t *testing.T) {
	findings := []common.Finding{{
		ID:       "ssh.root_login",
		Status:   common.StatusFail,
		Severity: common.SeverityHigh,
	}}

	got := AnnotateRemediationHints(findings, &osdetect.OSInfo{Family: osdetect.FamilyDebian})

	if got[0].RemediationMode != common.RemediationAutoFix {
		t.Fatalf("mode = %q, want %q", got[0].RemediationMode, common.RemediationAutoFix)
	}
	if len(got[0].AutoFixTaskIDs) != 1 || got[0].AutoFixTaskIDs[0] != "disable_root_ssh" {
		t.Fatalf("task IDs = %#v, want disable_root_ssh", got[0].AutoFixTaskIDs)
	}
	if len(findings[0].AutoFixTaskIDs) != 0 {
		t.Fatal("AnnotateRemediationHints mutated the input slice")
	}
}

func TestAnnotateRemediationHints_FirewallUsesDistroSpecificTask(t *testing.T) {
	findings := []common.Finding{{
		ID:       "firewall.none",
		Status:   common.StatusFail,
		Severity: common.SeverityHigh,
	}}

	debian := AnnotateRemediationHints(findings, &osdetect.OSInfo{Family: osdetect.FamilyDebian})
	if got := debian[0].AutoFixTaskIDs; len(got) != 1 || got[0] != "enable_ufw" {
		t.Fatalf("debian task IDs = %#v, want enable_ufw", got)
	}

	rhel := AnnotateRemediationHints(findings, &osdetect.OSInfo{Family: osdetect.FamilyRHEL})
	if got := rhel[0].AutoFixTaskIDs; len(got) != 1 || got[0] != "enable_firewalld" {
		t.Fatalf("rhel task IDs = %#v, want enable_firewalld", got)
	}
}

func TestAnnotateRemediationHints_UnknownAutoFixFallsBackToManual(t *testing.T) {
	findings := []common.Finding{{
		ID:       "firewall.none",
		Status:   common.StatusFail,
		Severity: common.SeverityHigh,
	}}

	got := AnnotateRemediationHints(findings, &osdetect.OSInfo{Family: osdetect.FamilyUnknown})

	if got[0].RemediationMode != common.RemediationManualReview {
		t.Fatalf("mode = %q, want %q", got[0].RemediationMode, common.RemediationManualReview)
	}
	if len(got[0].AutoFixTaskIDs) != 0 {
		t.Fatalf("task IDs = %#v, want none", got[0].AutoFixTaskIDs)
	}
}

func TestAnnotateRemediationHints_ManualAndInformational(t *testing.T) {
	findings := []common.Finding{
		{ID: "ports.listening", Status: common.StatusWarn, Severity: common.SeverityLow},
		{ID: "ssh.config", Status: common.StatusSkip, Severity: common.SeverityInfo},
	}

	got := AnnotateRemediationHints(findings, &osdetect.OSInfo{Family: osdetect.FamilyDebian})

	if got[0].RemediationMode != common.RemediationManualReview {
		t.Fatalf("ports mode = %q, want %q", got[0].RemediationMode, common.RemediationManualReview)
	}
	if got[1].RemediationMode != common.RemediationInformational {
		t.Fatalf("skip mode = %q, want %q", got[1].RemediationMode, common.RemediationInformational)
	}
}
