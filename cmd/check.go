package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/suleymanmercan/sur/internal/checker"
	"github.com/suleymanmercan/sur/internal/common"
	"github.com/suleymanmercan/sur/internal/lynis"
	"github.com/suleymanmercan/sur/internal/osdetect"
)

var (
	deepCheck    bool
	installLynis bool
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Audit the host and produce a security report",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Minute)
		defer cancel()

		osInfo, _ := osdetect.Detect()
		report := checker.Run(ctx)

		if deepCheck {
			lynisFindings := runLynis(ctx)
			report.Findings = append(report.Findings, lynisFindings...)
			report = checker.BuildReport(report.Findings)
		}
		report = checker.ApplyRemediationHints(report, osInfo)

		if jsonOutput {
			return emitJSON(map[string]any{
				"os":     osInfo,
				"report": report,
			})
		}
		renderReport(osInfo, report)
		return nil
	},
}

func init() {
	checkCmd.Flags().BoolVar(&deepCheck, "deep", false, "also run lynis audit (slower)")
	checkCmd.Flags().BoolVar(&installLynis, "install-lynis", false, "auto-install lynis when missing")
	rootCmd.AddCommand(checkCmd)
}

func runLynis(ctx context.Context) []common.Finding {
	if !lynis.IsInstalled() {
		if !installLynis {
			fmt.Fprintln(os.Stderr, "lynis not installed; pass --install-lynis to install it automatically, or skip --deep.")
			return nil
		}
		if err := installLynisPackage(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "lynis install failed: %v\n", err)
			return nil
		}
	}
	findings, err := lynis.Run(ctx, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "lynis run error: %v\n", err)
		return nil
	}
	return findings
}

// renderReport prints a colorized table + score block to stdout.
func renderReport(osInfo *osdetect.OSInfo, r common.Report) {
	header := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7CE38B"))
	dim := lipgloss.NewStyle().Faint(true)

	fmt.Println(header.Render("sur check"))
	if osInfo != nil && osInfo.Name != "" {
		fmt.Println(dim.Render(fmt.Sprintf("host: %s · pkg: %s", osInfo.Name, osInfo.PkgManager)))
	}
	fmt.Println()

	// table
	idW, titleW := 22, 50
	fmt.Printf("%-8s %-*s %s\n", "STATUS", idW, "ID", "TITLE")
	fmt.Println(strings.Repeat("─", 8+1+idW+1+titleW))
	for _, f := range r.Findings {
		icon := statusIcon(f.Status)
		title := truncate(f.Title, titleW)
		fmt.Printf("%-8s %-*s %s\n", icon, idW, truncate(f.ID, idW), title)
		if f.Status != common.StatusPass && f.Remediation != "" {
			fmt.Println(dim.Render("         → " + f.Remediation))
		}
		if hint := remediationHint(f); hint != "" {
			fmt.Println(dim.Render("         " + hint))
		}
	}
	fmt.Println()

	scoreColor := lipgloss.Color("#7CE38B")
	switch {
	case r.Score < 40:
		scoreColor = lipgloss.Color("#F25F5C")
	case r.Score < 75:
		scoreColor = lipgloss.Color("#FFBF00")
	}
	style := lipgloss.NewStyle().Bold(true).Foreground(scoreColor)
	fmt.Println(style.Render(fmt.Sprintf("Score: %d/%d   Issues: %d", r.Score, r.MaxScore, r.Issues)))
	if r.Issues > 0 {
		autoFixes, manual := remediationCounts(r.Findings)
		switch {
		case autoFixes > 0 && manual > 0:
			fmt.Println(dim.Render(fmt.Sprintf("%d finding(s) have task-backed auto-fixes; %d require manual review.", autoFixes, manual)))
			fmt.Println(dim.Render("Use `sur harden --only <task_id>` for targeted automatic fixes."))
		case autoFixes > 0:
			fmt.Println(dim.Render(fmt.Sprintf("%d finding(s) have task-backed auto-fixes.", autoFixes)))
			fmt.Println(dim.Render("Use `sur harden --only <task_id>` for targeted automatic fixes."))
		default:
			fmt.Println(dim.Render("Manual review required; no automatic hardening task is mapped for these findings."))
		}
	}
}

func remediationHint(f common.Finding) string {
	switch f.RemediationMode {
	case common.RemediationAutoFix:
		if len(f.AutoFixTaskIDs) == 0 {
			return ""
		}
		return fmt.Sprintf("auto-fix: sur harden --only %s", strings.Join(f.AutoFixTaskIDs, ","))
	case common.RemediationManualReview:
		if f.Status == common.StatusFail || f.Status == common.StatusWarn {
			return "manual review required"
		}
	case common.RemediationInformational:
		if f.Status == common.StatusSkip {
			return "informational only"
		}
	}
	return ""
}

func remediationCounts(findings []common.Finding) (autoFixes int, manual int) {
	for _, f := range findings {
		if f.Status != common.StatusFail && f.Status != common.StatusWarn {
			continue
		}
		switch f.RemediationMode {
		case common.RemediationAutoFix:
			autoFixes++
		case common.RemediationManualReview:
			manual++
		}
	}
	return autoFixes, manual
}

func statusIcon(s common.Status) string {
	switch s {
	case common.StatusPass:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#7CE38B")).Render("PASS")
	case common.StatusWarn:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FFBF00")).Render("WARN")
	case common.StatusFail:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#F25F5C")).Render("FAIL")
	case common.StatusSkip:
		return lipgloss.NewStyle().Faint(true).Render("SKIP")
	}
	return string(s)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "…"
}

func emitJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
