package stack

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Status returns the container status for each service in the stack.
func Status(dir string) ([]ContainerStatus, error) {
	out, err := composeOutput(dir, "ps", "--format", "table {{.Name}}\t{{.State}}\t{{.Status}}")
	if err != nil {
		return nil, err
	}

	var rows []ContainerStatus
	scanner := bufio.NewScanner(strings.NewReader(out))
	first := true
	for scanner.Scan() {
		line := scanner.Text()
		if first {
			first = false
			continue // skip header
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		row := ContainerStatus{}
		if len(parts) > 0 {
			row.Name = strings.TrimSpace(parts[0])
		}
		if len(parts) > 1 {
			row.State = strings.TrimSpace(parts[1])
		}
		if len(parts) > 2 {
			row.Status = strings.TrimSpace(parts[2])
		}
		rows = append(rows, row)
	}
	return rows, scanner.Err()
}

// Logs streams recent logs from the stack (last 100 lines, no follow).
func Logs(dir string, lines int) (string, error) {
	if lines <= 0 {
		lines = 100
	}
	return composeOutput(dir, "logs", "--no-color", "--tail", fmt.Sprintf("%d", lines))
}

// Restart runs `docker compose restart`.
func Restart(dir string, log func(string)) error {
	log("Restarting stack...")
	return composeRun(dir, log, "restart")
}

// Rotate generates a new random password, writes it to secrets/, then calls
// the stack's rotate Lua hook (which updates the DB user), and restarts.
func Rotate(dir string, log func(string)) error {
	log("Generating new password...")
	newPass, err := generateSecret(24)
	if err != nil {
		return fmt.Errorf("generate secret: %w", err)
	}

	// Find all .txt files under secrets/ and rotate the first non-root one.
	// Stacks with a rotate hook take full control via Lua.
	secretsDir := filepath.Join(dir, "secrets")
	entries, err := os.ReadDir(secretsDir)
	if err != nil {
		return fmt.Errorf("read secrets dir: %w", err)
	}

	rotated := false
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".txt" {
			continue
		}
		// Skip root_password — that requires extra steps.
		if e.Name() == "root_password.txt" {
			continue
		}
		p := filepath.Join(secretsDir, e.Name())
		log(fmt.Sprintf("Writing new secret → %s", p))
		if werr := os.WriteFile(p, []byte(newPass), 0o600); werr != nil { // #nosec G306
			return fmt.Errorf("write secret: %w", werr)
		}
		rotated = true
		break
	}
	if !rotated {
		return fmt.Errorf("no rotatable secret found in %s", secretsDir)
	}

	// Run the stack's rotate Lua hook (applies the change inside the DB).
	luaPath := filepath.Join(dir, "stack.lua")
	if fileExists(luaPath) {
		log("Running rotate hook (stack.lua)...")
		if herr := RunHook(luaPath, "rotate", dir, log); herr != nil {
			return fmt.Errorf("rotate hook: %w", herr)
		}
	} else {
		log("No stack.lua rotate hook found — restarting to pick up new secret.")
	}

	log("Restarting stack to apply new credentials...")
	return composeRun(dir, log, "up", "-d", "--force-recreate")
}

// Down stops and removes containers (but NOT volumes or data).
func Down(dir string, log func(string)) error {
	log("Stopping stack (docker compose down)...")
	// Intentionally never pass --volumes.
	return composeRun(dir, log, "down")
}

// Update pulls new images and restarts the stack.
func Update(dir string, log func(string)) error {
	log("Pulling latest images...")
	if err := composeRun(dir, log, "pull"); err != nil {
		return fmt.Errorf("compose pull: %w", err)
	}
	log("Restarting with updated images...")
	if err := composeRun(dir, log, "up", "-d"); err != nil {
		return fmt.Errorf("compose up after pull: %w", err)
	}
	return nil
}

// Backup copies data/ and secrets/ into backups/<timestamp>/.
func Backup(dir string, log func(string)) error {
	ts := time.Now().Format("2006-01-02T15-04-05")
	dst := filepath.Join(dir, "backups", ts)
	log(fmt.Sprintf("Creating backup in %s", dst))

	if err := os.MkdirAll(dst, 0o755); err != nil {
		return fmt.Errorf("mkdir backup: %w", err)
	}

	for _, sub := range []string{"data", "secrets"} {
		src := filepath.Join(dir, sub)
		if _, err := os.Stat(src); os.IsNotExist(err) {
			continue
		}
		dstSub := filepath.Join(dst, sub)
		log(fmt.Sprintf("Copying %s → %s", src, dstSub))
		if err := copyDir(src, dstSub); err != nil {
			return fmt.Errorf("copy %s: %w", sub, err)
		}
	}

	// Run backup Lua hook if present.
	luaPath := filepath.Join(dir, "stack.lua")
	if fileExists(luaPath) {
		log("Running backup hook (stack.lua)...")
		if err := RunHook(luaPath, "backup", dir, log); err != nil {
			log(fmt.Sprintf("Warning: backup hook error: %v", err))
		}
	}

	log("Backup completed.")
	return nil
}

// EditConfig rewrites .env with new values and optionally restarts.
func EditConfig(def StackDef, dir string, values map[string]string, restart bool, log func(string)) error {
	envPath := filepath.Join(dir, ".env")

	// Read existing env to preserve any manual changes not covered by fields.
	existing, _ := ReadEnv(envPath)

	// Merge: user-provided values override existing, field defaults fill gaps.
	for _, field := range def.Config {
		if v, ok := values[field.ID]; ok && v != "" {
			existing[envKey(field.ID)] = v
			// If it's a secret, also update the secrets file.
			if field.Type == FieldTypeSecret {
				secretPath := filepath.Join(dir, "secrets", field.ID+".txt")
				_ = os.WriteFile(secretPath, []byte(v), 0o600) // #nosec G306
			}
		}
	}

	log(fmt.Sprintf("Writing updated %s", envPath))
	if err := WriteEnv(envPath, existing, def.Config); err != nil {
		return fmt.Errorf("write .env: %w", err)
	}

	if restart {
		return Restart(dir, log)
	}
	return nil
}

// ---- internal docker compose helpers ----

func composeValidate(dir string) error {
	cmd := exec.Command("docker", "compose", "config", "--quiet") // #nosec G204
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w\n%s", err, string(out))
	}
	return nil
}

func composeUp(dir string, log func(string)) error {
	return composeRun(dir, log, "up", "-d")
}

func composeRun(dir string, log func(string), args ...string) error {
	full := append([]string{"compose"}, args...)
	cmd := exec.Command("docker", full...) // #nosec G204
	cmd.Dir = dir
	cmd.Env = os.Environ()

	out, err := cmd.CombinedOutput()
	if log != nil && len(out) > 0 {
		for _, line := range strings.Split(string(out), "\n") {
			if strings.TrimSpace(line) != "" {
				log(line)
			}
		}
	}
	return err
}

func composeOutput(dir string, args ...string) (string, error) {
	full := append([]string{"compose"}, args...)
	cmd := exec.Command("docker", full...) // #nosec G204
	cmd.Dir = dir
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// copyDir recursively copies src directory to dst.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		return copyFile(path, target)
	})
}
