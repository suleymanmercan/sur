// Package checker implements the built-in security checks that run
// without external tooling. Each check returns one or more Findings.
package checker

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/suleymanmercan/sur/internal/common"
)

// CheckFunc runs a single built-in check.
type CheckFunc func(ctx context.Context) []common.Finding

// All returns every built-in check registered in sur.
func All() []CheckFunc {
	return []CheckFunc{
		CheckSSH,
		CheckFirewall,
		CheckFail2ban,
		CheckUnattendedUpgrades,
		CheckListeningPorts,
		CheckSudoNoPasswd,
	}
}

// Run executes every check and returns the merged report.
func Run(ctx context.Context) common.Report {
	var findings []common.Finding
	for _, c := range All() {
		findings = append(findings, c(ctx)...)
	}
	return BuildReport(findings)
}

// BuildReport computes score from a slice of findings.
// Score formula: start at 100, subtract weighted penalties for non-pass entries.
func BuildReport(findings []common.Finding) common.Report {
	score := 100
	issues := 0
	for _, f := range findings {
		switch f.Status {
		case common.StatusFail:
			issues++
			switch f.Severity {
			case common.SeverityHigh:
				score -= 15
			case common.SeverityMed:
				score -= 8
			default:
				score -= 4
			}
		case common.StatusWarn:
			issues++
			score -= 3
		}
	}
	if score < 0 {
		score = 0
	}
	return common.Report{Findings: findings, Score: score, MaxScore: 100, Issues: issues}
}

// ---------- helpers ----------

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func readFile(p string) (string, error) {
	b, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// sshdConfigValue parses key from the main sshd_config content string.
// It returns the last matching value (later entries override earlier ones).
func sshdConfigValue(content, key string) string {
	scanner := bufio.NewScanner(strings.NewReader(content))
	val := ""
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if strings.EqualFold(fields[0], key) {
			val = fields[1]
		}
	}
	return val
}

// sshdConfigValueFromFiles reads the main sshd_config and all drop-in
// files under /etc/ssh/sshd_config.d/*.conf (Ubuntu 22.04+, Debian 12+).
// The last matching value wins — same precedence as sshd itself applies.
func sshdConfigValueFromFiles(mainContent, key string) string {
	val := sshdConfigValue(mainContent, key)

	// Walk drop-in directory; ignore errors (directory may not exist).
	matches, _ := filepath.Glob("/etc/ssh/sshd_config.d/*.conf")
	for _, p := range matches {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		if v := sshdConfigValue(string(b), key); v != "" {
			val = v
		}
	}
	return val
}

// sshdEffectiveValues runs `sshd -T` to get the fully-merged runtime
// configuration. Returns an empty (non-nil) map when sshd -T fails
// (e.g. non-root caller on some distros) so callers can fall back to
// file parsing without triggering a nil-map panic.
func sshdEffectiveValues(ctx context.Context) map[string]string {
	out, code, err := runCmd(ctx, "sshd", "-T")
	if err != nil || code != 0 {
		// sshd -T may require root; fall back to file-based parsing silently.
		return map[string]string{}
	}
	values := map[string]string{}
	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		values[strings.ToLower(fields[0])] = fields[1]
	}
	return values
}

// sshdValue resolves a config key using the following priority:
// 1. sshd -T effective output (most accurate, requires root)
// 2. main sshd_config + sshd_config.d/*.conf drop-ins
func sshdValue(effective map[string]string, content, key string) string {
	if v := effective[strings.ToLower(key)]; v != "" {
		return v
	}
	return sshdConfigValueFromFiles(content, key)
}

func runCmd(ctx context.Context, name string, args ...string) (string, int, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	code := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			code = exitErr.ExitCode()
			return string(out), code, nil
		}
		return string(out), -1, err
	}
	return string(out), code, nil
}

// ---------- checks ----------

// CheckSSH inspects /etc/ssh/sshd_config for unsafe defaults.
func CheckSSH(ctx context.Context) []common.Finding {
	path := "/etc/ssh/sshd_config"
	if !fileExists(path) {
		return []common.Finding{{
			ID: "ssh.config", Category: "SSH", Title: "SSH server",
			Status: common.StatusSkip, Severity: common.SeverityInfo,
			Detail: "sshd_config not found — OpenSSH likely not installed",
			Source: "builtin",
		}}
	}
	content, err := readFile(path)
	if err != nil {
		return []common.Finding{{
			ID: "ssh.config", Category: "SSH", Title: "SSH server",
			Status: common.StatusWarn, Severity: common.SeverityLow,
			Detail: fmt.Sprintf("cannot read %s: %v", path, err),
			Source: "builtin",
		}}
	}

	var out []common.Finding
	effective := sshdEffectiveValues(ctx)
	rootLogin := strings.ToLower(sshdValue(effective, content, "PermitRootLogin"))
	if rootLogin == "" || rootLogin == "yes" {
		out = append(out, common.Finding{
			ID: "ssh.root_login", Category: "SSH",
			Title:       "SSH root login is allowed",
			Status:      common.StatusFail,
			Severity:    common.SeverityHigh,
			Detail:      "PermitRootLogin is set to '" + rootLogin + "'",
			Remediation: "Set 'PermitRootLogin no' in /etc/ssh/sshd_config",
			Source:      "builtin",
		})
	} else {
		out = append(out, common.Finding{
			ID: "ssh.root_login", Category: "SSH",
			Title: "SSH root login disabled", Status: common.StatusPass,
			Severity: common.SeverityInfo, Source: "builtin",
		})
	}

	passAuth := strings.ToLower(sshdValue(effective, content, "PasswordAuthentication"))
	if passAuth != "no" {
		out = append(out, common.Finding{
			ID: "ssh.password_auth", Category: "SSH",
			Title:       "SSH password authentication is enabled",
			Status:      common.StatusFail,
			Severity:    common.SeverityMed,
			Detail:      "PasswordAuthentication is '" + passAuth + "'",
			Remediation: "Use SSH keys and set 'PasswordAuthentication no'",
			Source:      "builtin",
		})
	} else {
		out = append(out, common.Finding{
			ID: "ssh.password_auth", Category: "SSH",
			Title: "SSH password auth disabled", Status: common.StatusPass,
			Severity: common.SeverityInfo, Source: "builtin",
		})
	}

	port := sshdValue(effective, content, "Port")
	if port == "" || port == "22" {
		out = append(out, common.Finding{
			ID: "ssh.port", Category: "SSH",
			Title:       "SSH listens on default port 22",
			Status:      common.StatusWarn,
			Severity:    common.SeverityLow,
			Detail:      "Default port attracts automated scanners",
			Remediation: "Move SSH to a non-standard port",
			Source:      "builtin",
		})
	} else {
		out = append(out, common.Finding{
			ID: "ssh.port", Category: "SSH",
			Title: "SSH on non-default port (" + port + ")", Status: common.StatusPass,
			Severity: common.SeverityInfo, Source: "builtin",
		})
	}
	return out
}

// CheckFirewall verifies that ufw or firewalld is enabled.
func CheckFirewall(ctx context.Context) []common.Finding {
	if _, err := exec.LookPath("ufw"); err == nil {
		out, _, _ := runCmd(ctx, "ufw", "status")
		if strings.Contains(strings.ToLower(out), "status: active") {
			return []common.Finding{{
				ID: "firewall.ufw", Category: "Firewall",
				Title: "UFW firewall active", Status: common.StatusPass,
				Severity: common.SeverityInfo, Source: "builtin",
			}}
		}
		return []common.Finding{{
			ID: "firewall.ufw", Category: "Firewall",
			Title:       "UFW installed but inactive",
			Status:      common.StatusFail,
			Severity:    common.SeverityHigh,
			Remediation: "Run 'sudo ufw enable' after allowing SSH",
			Source:      "builtin",
		}}
	}
	if _, err := exec.LookPath("firewall-cmd"); err == nil {
		out, _, _ := runCmd(ctx, "firewall-cmd", "--state")
		if strings.TrimSpace(out) == "running" {
			return []common.Finding{{
				ID: "firewall.firewalld", Category: "Firewall",
				Title: "firewalld active", Status: common.StatusPass,
				Severity: common.SeverityInfo, Source: "builtin",
			}}
		}
	}
	return []common.Finding{{
		ID: "firewall.none", Category: "Firewall",
		Title:       "No active host firewall detected",
		Status:      common.StatusFail,
		Severity:    common.SeverityHigh,
		Remediation: "Install and enable ufw or firewalld",
		Source:      "builtin",
	}}
}

// CheckFail2ban verifies fail2ban service.
func CheckFail2ban(ctx context.Context) []common.Finding {
	if _, err := exec.LookPath("fail2ban-client"); err != nil {
		return []common.Finding{{
			ID: "fail2ban.installed", Category: "Brute-force",
			Title:       "fail2ban not installed",
			Status:      common.StatusFail,
			Severity:    common.SeverityMed,
			Remediation: "Install fail2ban and enable the sshd jail",
			Source:      "builtin",
		}}
	}
	_, code, _ := runCmd(ctx, "systemctl", "is-active", "fail2ban")
	if code == 0 {
		return []common.Finding{{
			ID: "fail2ban.active", Category: "Brute-force",
			Title: "fail2ban active", Status: common.StatusPass,
			Severity: common.SeverityInfo, Source: "builtin",
		}}
	}
	return []common.Finding{{
		ID: "fail2ban.active", Category: "Brute-force",
		Title:       "fail2ban installed but inactive",
		Status:      common.StatusWarn,
		Severity:    common.SeverityMed,
		Remediation: "systemctl enable --now fail2ban",
		Source:      "builtin",
	}}
}

// CheckUnattendedUpgrades verifies automatic security updates.
func CheckUnattendedUpgrades(ctx context.Context) []common.Finding {
	// Debian family
	if fileExists("/etc/apt/apt.conf.d/20auto-upgrades") {
		c, _ := readFile("/etc/apt/apt.conf.d/20auto-upgrades")
		if strings.Contains(c, `"1"`) {
			return []common.Finding{{
				ID: "updates.auto", Category: "Updates",
				Title: "Automatic security updates enabled", Status: common.StatusPass,
				Severity: common.SeverityInfo, Source: "builtin",
			}}
		}
	}
	// RHEL family
	if _, err := exec.LookPath("dnf-automatic"); err == nil {
		_, code, _ := runCmd(ctx, "systemctl", "is-active", "dnf-automatic.timer")
		if code == 0 {
			return []common.Finding{{
				ID: "updates.auto", Category: "Updates",
				Title: "dnf-automatic timer active", Status: common.StatusPass,
				Severity: common.SeverityInfo, Source: "builtin",
			}}
		}
	}
	return []common.Finding{{
		ID: "updates.auto", Category: "Updates",
		Title:       "Automatic security updates not configured",
		Status:      common.StatusFail,
		Severity:    common.SeverityMed,
		Remediation: "Install unattended-upgrades (Debian) or dnf-automatic (RHEL)",
		Source:      "builtin",
	}}
}

// CheckListeningPorts surfaces unexpected listening sockets.
func CheckListeningPorts(ctx context.Context) []common.Finding {
	out, _, err := runCmd(ctx, "ss", "-tulnH")
	if err != nil {
		return []common.Finding{{
			ID: "ports.listening", Category: "Network",
			Title: "Could not enumerate listening ports", Status: common.StatusSkip,
			Severity: common.SeverityInfo, Detail: err.Error(), Source: "builtin",
		}}
	}
	count := 0
	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) != "" {
			count++
		}
	}
	status := common.StatusPass
	sev := common.SeverityInfo
	if count > 8 {
		status = common.StatusWarn
		sev = common.SeverityLow
	}
	return []common.Finding{{
		ID: "ports.listening", Category: "Network",
		Title:       fmt.Sprintf("%d listening sockets", count),
		Status:      status,
		Severity:    sev,
		Remediation: "Audit listening services with 'ss -tulnp' and disable unneeded ones",
		Source:      "builtin",
	}}
}

// CheckSudoNoPasswd scans sudoers for NOPASSWD entries.
func CheckSudoNoPasswd(ctx context.Context) []common.Finding {
	paths := []string{"/etc/sudoers"}
	if entries, err := os.ReadDir("/etc/sudoers.d"); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				paths = append(paths, "/etc/sudoers.d/"+e.Name())
			}
		}
	}
	var hits []string
	for _, p := range paths {
		c, err := readFile(p)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(strings.NewReader(c))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "#") {
				continue
			}
			if strings.Contains(line, "NOPASSWD") {
				hits = append(hits, p+": "+line)
			}
		}
	}
	if len(hits) == 0 {
		return []common.Finding{{
			ID: "sudo.nopasswd", Category: "Sudo",
			Title: "No NOPASSWD entries in sudoers", Status: common.StatusPass,
			Severity: common.SeverityInfo, Source: "builtin",
		}}
	}
	return []common.Finding{{
		ID: "sudo.nopasswd", Category: "Sudo",
		Title:       fmt.Sprintf("%d NOPASSWD sudoers entr(y/ies) found", len(hits)),
		Status:      common.StatusWarn,
		Severity:    common.SeverityMed,
		Detail:      strings.Join(hits, "\n"),
		Remediation: "Remove NOPASSWD unless strictly required by automation",
		Source:      "builtin",
	}}
}
