package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/suleymanmercan/sur/internal/osdetect"
)

var (
	installDryRun    bool
	installYesFlag   bool
	installResume    bool
	installAllowAll  bool
	installOnlyIDs   []string
	installTaskDir   string
	installStateFile string
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Interactively pick and install server essentials",
	Long: `Pick optional server setup tasks such as swap, Docker, Caddy,
and common CLI packages. sur runs selected tasks directly on the host and
records the session in the same SQLite state store used by hardening tasks.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if os.Geteuid() != 0 && !installDryRun {
			return fmt.Errorf("sur install requires root privileges (run with sudo) — or pass --dry-run")
		}

		tasks, err := loadTaskSet(embeddedTaskFS, "install_tasks", installTaskDir)
		if err != nil {
			return err
		}
		sessionID, results, err := runTaskSet(cmd.Context(), tasks, taskRunOptions{
			DryRun:   installDryRun,
			Yes:      installYesFlag,
			Resume:   installResume,
			All:      installAllowAll,
			OnlyIDs:  installOnlyIDs,
			State:    installStateFile,
			Timeout:  45 * time.Minute,
			TUITitle: "sur — choose install tasks",
		})
		if err != nil || results == nil {
			return err
		}
		if len(results) == 0 {
			return nil
		}

		if jsonOutput {
			return emitJSON(map[string]any{
				"session_id": sessionID,
				"results":    results,
			})
		}
		printResults(sessionID, results)
		return nil
	},
}

// installLynisPackage tries to install lynis via the detected package manager.
func installLynisPackage(ctx context.Context) error {
	info, _ := osdetect.Detect()
	var cmd string
	switch info.PkgManager {
	case "apt":
		cmd = "DEBIAN_FRONTEND=noninteractive apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y lynis"
	case "dnf":
		cmd = "dnf install -y epel-release || true; dnf install -y lynis"
	case "yum":
		cmd = "yum install -y epel-release || true; yum install -y lynis"
	case "zypper":
		cmd = "zypper -n install lynis"
	default:
		return fmt.Errorf("unknown package manager — install lynis manually")
	}
	if os.Geteuid() != 0 {
		return fmt.Errorf("need root to install lynis (re-run with sudo)")
	}
	return runShell(ctx, cmd)
}

func runShell(ctx context.Context, command string) error {
	c := execCommandContext(ctx, "sh", "-c", command)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func init() {
	installCmd.Flags().BoolVar(&installDryRun, "dry-run", false, "show planned actions without touching the system")
	installCmd.Flags().BoolVar(&installYesFlag, "yes", false, "skip TUI and apply every install task")
	installCmd.Flags().BoolVar(&installResume, "resume", false, "resume the last unfinished session")
	installCmd.Flags().BoolVar(&installAllowAll, "all", false, "apply every install task without prompting")
	installCmd.Flags().StringSliceVar(&installOnlyIDs, "only", nil, "comma-separated install task IDs to run")
	installCmd.Flags().StringVar(&installTaskDir, "tasks", "", "directory containing install task YAML files")
	installCmd.Flags().StringVar(&installStateFile, "state", "", "override SQLite path (default: /var/lib/sur/sur.db or $SUR_DB)")
	rootCmd.AddCommand(installCmd)
}
