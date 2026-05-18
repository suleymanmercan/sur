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
		if f.Status == common.StatusFail && f.Remediation != "" {
			fmt.Println(dim.Render("         → " + f.Remediation))
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
		fmt.Println(dim.Render("Run `sur harden` to fix interactively."))
	}
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
