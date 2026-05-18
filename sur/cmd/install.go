package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/suleymanmercan/sur/internal/osdetect"
)

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
