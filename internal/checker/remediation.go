package checker

import (
	"github.com/suleymanmercan/sur/internal/common"
	"github.com/suleymanmercan/sur/internal/osdetect"
)

type autoFixHint struct {
	Default  []string
	ByFamily map[osdetect.Family][]string
}

var autoFixHints = map[string]autoFixHint{
	"ssh.root_login": {
		Default: []string{"disable_root_ssh"},
	},
	"ssh.password_auth": {
		Default: []string{"ssh_password_auth_off"},
	},
	"ssh.port": {
		Default: []string{"change_ssh_port"},
	},
	"firewall.ufw": {
		Default: []string{"enable_ufw"},
	},
	"firewall.none": {
		ByFamily: map[osdetect.Family][]string{
			osdetect.FamilyDebian: []string{"enable_ufw"},
			osdetect.FamilyRHEL:   []string{"enable_firewalld"},
			osdetect.FamilyFedora: []string{"enable_firewalld"},
		},
	},
	"fail2ban.installed": {
		Default: []string{"install_fail2ban"},
	},
	"fail2ban.active": {
		Default: []string{"install_fail2ban"},
	},
	"updates.auto": {
		ByFamily: map[osdetect.Family][]string{
			osdetect.FamilyDebian: []string{"unattended_upgrades"},
			osdetect.FamilyRHEL:   []string{"dnf_automatic"},
			osdetect.FamilyFedora: []string{"dnf_automatic"},
		},
	},
}

// ApplyRemediationHints annotates findings with task-backed auto-fix metadata.
func ApplyRemediationHints(report common.Report, osInfo *osdetect.OSInfo) common.Report {
	report.Findings = AnnotateRemediationHints(report.Findings, osInfo)
	return report
}

// AnnotateRemediationHints returns a copied finding slice with remediation mode metadata.
func AnnotateRemediationHints(findings []common.Finding, osInfo *osdetect.OSInfo) []common.Finding {
	out := make([]common.Finding, len(findings))
	copy(out, findings)

	for i := range out {
		out[i].RemediationMode = ""
		out[i].AutoFixTaskIDs = nil

		switch out[i].Status {
		case common.StatusFail, common.StatusWarn:
			taskIDs := autoFixTaskIDs(out[i].ID, osInfo)
			if len(taskIDs) > 0 {
				out[i].RemediationMode = common.RemediationAutoFix
				out[i].AutoFixTaskIDs = taskIDs
			} else {
				out[i].RemediationMode = common.RemediationManualReview
			}
		case common.StatusSkip:
			out[i].RemediationMode = common.RemediationInformational
		}
	}

	return out
}

func autoFixTaskIDs(findingID string, osInfo *osdetect.OSInfo) []string {
	hint, ok := autoFixHints[findingID]
	if !ok {
		return nil
	}
	if osInfo != nil {
		if taskIDs := hint.ByFamily[osInfo.Family]; len(taskIDs) > 0 {
			return append([]string(nil), taskIDs...)
		}
	}
	if len(hint.Default) == 0 {
		return nil
	}
	return append([]string(nil), hint.Default...)
}
