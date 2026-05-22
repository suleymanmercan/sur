package stack

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
)

// Install copies the stack template into InstallDir, generates .env and secrets,
// validates the compose file, then starts the stack.
//
// values maps field ID → user-provided string value.
// For secret fields with generate=true and no user value, a random password is generated.
func Install(def StackDef, values map[string]string, log func(string)) error {
	dir := InstalledDirFor(def.ID)

	log(fmt.Sprintf("Creating directory %s", dir))
	// Directory permission plan:
	//   stack root, data, backups → 0755 (user can cd/ls without sudo)
	//   secrets                   → 0700 (root-only; contains passwords)
	subPerms := map[string]os.FileMode{
		"":        0o755,
		"data":    0o755,
		"backups": 0o755,
		"secrets": 0o700,
	}
	for sub, perm := range subPerms {
		p := filepath.Join(dir, sub)
		if err := os.MkdirAll(p, perm); err != nil {
			return fmt.Errorf("mkdir %s: %w", p, err)
		}
	}
	// Chown the stack dir (except secrets) to the real invoking user so they
	// can manage their own stacks without sudo.
	chownToRealUser(dir, log)

	// --- copy template files ---
	filesToCopy := []string{"stack.yaml", "compose.yml"}
	for _, f := range filesToCopy {
		var srcPath string
		if def.Source == "custom" {
			srcPath = filepath.Join(CustomDir, def.ID, f)
		} else {
			var err error
			if f == "stack.yaml" {
				// already have the def; write it from cache
				srcPath = filepath.Join(CacheDir, def.ID, f)
				if !fileExists(srcPath) {
					// re-fetch
					_, err = FetchStackDef(def.ID)
					if err != nil {
						return fmt.Errorf("re-fetch stack.yaml: %w", err)
					}
				}
			} else {
				srcPath, err = FetchTemplateFile(def.ID, f)
				if err != nil {
					return fmt.Errorf("fetch template %s: %w", f, err)
				}
			}
		}
		dst := filepath.Join(dir, f)
		log(fmt.Sprintf("Copying %s → %s", srcPath, dst))
		if err := copyFile(srcPath, dst); err != nil {
			return fmt.Errorf("copy %s: %w", f, err)
		}
	}

	// Try to copy stack.lua if present (optional).
	luaSrc := ""
	if def.Source == "custom" {
		luaSrc = filepath.Join(CustomDir, def.ID, "stack.lua")
	} else {
		p, err := FetchTemplateFile(def.ID, "stack.lua")
		if err == nil {
			luaSrc = p
		}
	}
	if luaSrc != "" && fileExists(luaSrc) {
		dst := filepath.Join(dir, "stack.lua")
		_ = copyFile(luaSrc, dst)
	}

	// --- resolve config values and generate secrets ---
	resolvedValues := make(map[string]string, len(def.Config))
	for _, field := range def.Config {
		val := values[field.ID]
		if field.Type == FieldTypeSecret {
			if val == "" {
				if field.Generate {
					var err error
					val, err = generateSecret(24)
					if err != nil {
						return fmt.Errorf("generate secret for %s: %w", field.ID, err)
					}
					log(fmt.Sprintf("Generated secret for field '%s'", field.Label))
				}
			}
			// Write to secrets/<field_id>.txt with mode 0600.
			secretPath := filepath.Join(dir, "secrets", field.ID+".txt")
			if err := os.WriteFile(secretPath, []byte(val), 0o600); err != nil { // #nosec G306
				return fmt.Errorf("write secret %s: %w", secretPath, err)
			}
			log(fmt.Sprintf("Secret written to %s", secretPath))
		}
		resolvedValues[field.ID] = val
	}

	// --- write .env ---
	envMap := EnvFromConfig(def.Config, resolvedValues)
	envPath := filepath.Join(dir, ".env")
	log(fmt.Sprintf("Writing %s", envPath))
	if err := WriteEnv(envPath, envMap, def.Config); err != nil {
		return fmt.Errorf("write .env: %w", err)
	}

	// --- run install Lua hook if present ---
	luaDst := filepath.Join(dir, "stack.lua")
	if fileExists(luaDst) {
		log("Running install hook (stack.lua)...")
		if err := RunHook(luaDst, "install", dir, log); err != nil {
			log(fmt.Sprintf("Warning: install hook error: %v", err))
		}
	}

	// --- validate compose ---
	log("Validating compose file...")
	if err := composeValidate(dir); err != nil {
		return fmt.Errorf("compose validation failed: %w", err)
	}

	// --- start the stack ---
	log("Starting stack (docker compose up -d)...")
	if err := composeUp(dir, log); err != nil {
		return fmt.Errorf("docker compose up: %w", err)
	}

	log(fmt.Sprintf("Stack '%s' installed and started successfully.", def.Name))
	return nil
}

// chownToRealUser changes ownership of the stack directory tree (except the
// secrets sub-dir) to the user who invoked sudo, so they can browse and edit
// their own stacks without needing sudo for every ls/cat/vim.
func chownToRealUser(dir string, log func(string)) {
	username := os.Getenv("SUDO_USER")
	if username == "" {
		// Not running under sudo; nothing to fix.
		return
	}
	u, err := user.Lookup(username)
	if err != nil {
		log(fmt.Sprintf("Warning: could not look up user %q: %v", username, err))
		return
	}
	// Chown the stack root, data, and backups dirs to the real user.
	// secrets/ intentionally stays root-owned (0700).
	for _, sub := range []string{"", "data", "backups"} {
		p := filepath.Join(dir, sub)
		cmd := exec.Command("chown", u.Uid+":"+u.Gid, p) // #nosec G204
		if out, cerr := cmd.CombinedOutput(); cerr != nil {
			log(fmt.Sprintf("Warning: chown %s: %v %s", p, cerr, string(out)))
		}
	}
	log(fmt.Sprintf("Ownership of %s set to %s", dir, username))
}

// generateSecret returns a URL-safe random string of approx byteLen*4/3 characters.
func generateSecret(byteLen int) (string, error) {
	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src) // #nosec G304 — src is always a known catalog cache or custom stack path
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o640) // #nosec G306
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
