package stack

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	// rawBaseURL is the GitHub raw URL for catalog files.
	rawBaseURL = "https://raw.githubusercontent.com/suleymanmercan/sur/main/catalog/stacks"

	// CacheDir is where fetched catalog files are stored.
	CacheDir = "/var/cache/sur/catalog"

	// CacheTTL is how long a cached file is considered fresh.
	CacheTTL = 24 * time.Hour

	// InstallDir is where live stack instances live.
	InstallDir = "/opt/sur/stacks"

	// CustomDir is where users place their own stack templates.
	CustomDir = "/etc/sur/stacks"
)

// FetchIndex downloads (or returns a cached copy of) the stack index.
// On failure it falls back to any stale cached version.
func FetchIndex() ([]StackMeta, error) {
	cached := filepath.Join(CacheDir, "index.yaml")

	// Try to use a fresh cache first.
	if isFresh(cached) {
		return parseIndex(cached)
	}

	// Download fresh copy.
	data, err := httpGet(rawBaseURL + "/index.yaml")
	if err != nil {
		// Fall back to stale cache if available.
		if fileExists(cached) {
			return parseIndex(cached)
		}
		return nil, fmt.Errorf("fetch catalog index: %w", err)
	}

	if err := writeCached(cached, data); err != nil {
		return nil, fmt.Errorf("write catalog cache: %w", err)
	}
	return parseIndexBytes(data)
}

// FetchStackDef downloads stack.yaml for stackID and returns the parsed def.
func FetchStackDef(stackID string) (StackDef, error) {
	cached := filepath.Join(CacheDir, stackID, "stack.yaml")

	var data []byte
	if isFresh(cached) {
		b, err := os.ReadFile(cached) // #nosec G304 — path built from CacheDir + validated stackID
		if err == nil {
			data = b
		}
	}

	if data == nil {
		var err error
		data, err = httpGet(rawBaseURL + "/" + stackID + "/stack.yaml")
		if err != nil {
			if fileExists(cached) {
				data, _ = os.ReadFile(cached) // #nosec G304
			} else {
				return StackDef{}, fmt.Errorf("fetch stack.yaml for %s: %w", stackID, err)
			}
		}
		_ = writeCached(cached, data)
	}

	var def StackDef
	if err := yaml.Unmarshal(data, &def); err != nil {
		return StackDef{}, fmt.Errorf("parse stack.yaml for %s: %w", stackID, err)
	}
	def.Source = "official"
	return def, nil
}

// FetchTemplateFile downloads a single template file (compose.yml or stack.lua)
// into the cache directory and returns its local path.
func FetchTemplateFile(stackID, filename string) (string, error) {
	cached := filepath.Join(CacheDir, stackID, filename)

	if isFresh(cached) {
		return cached, nil
	}

	data, err := httpGet(rawBaseURL + "/" + stackID + "/" + filename)
	if err != nil {
		if fileExists(cached) {
			return cached, nil // stale fallback
		}
		return "", fmt.Errorf("fetch %s for %s: %w", filename, stackID, err)
	}

	if err := writeCached(cached, data); err != nil {
		return "", err
	}
	return cached, nil
}

// RefreshCache forces re-download of the index and all currently-cached stacks.
func RefreshCache() error {
	// Remove cached index to force re-download.
	_ = os.Remove(filepath.Join(CacheDir, "index.yaml"))

	idx, err := FetchIndex()
	if err != nil {
		return err
	}
	for _, m := range idx {
		// Remove cached stack files.
		dir := filepath.Join(CacheDir, m.ID)
		_ = os.RemoveAll(dir)
	}
	// Re-fetch index (repopulates cache).
	_, err = FetchIndex()
	return err
}

// ListCustom returns stack definitions found in CustomDir (/etc/sur/stacks/).
func ListCustom() ([]StackDef, error) {
	entries, err := os.ReadDir(CustomDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var defs []StackDef
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		p := filepath.Join(CustomDir, e.Name(), "stack.yaml")
		b, err := os.ReadFile(p) // #nosec G304 — CustomDir is a well-known system path
		if err != nil {
			continue
		}
		var def StackDef
		if err := yaml.Unmarshal(b, &def); err != nil {
			continue
		}
		def.Source = "custom"
		defs = append(defs, def)
	}
	return defs, nil
}

// ListInstalled returns all stacks that have been installed to InstallDir.
func ListInstalled() ([]InstalledStack, error) {
	entries, err := os.ReadDir(InstallDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var stacks []InstalledStack
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(InstallDir, e.Name())
		p := filepath.Join(dir, "stack.yaml")
		b, err := os.ReadFile(p) // #nosec G304 — InstallDir is a well-known system path
		if err != nil {
			continue
		}
		var def StackDef
		if err := yaml.Unmarshal(b, &def); err != nil {
			continue
		}
		running := isRunning(dir)
		stacks = append(stacks, InstalledStack{Def: def, Dir: dir, Running: running})
	}
	return stacks, nil
}

// InstalledDirFor returns the runtime directory for a given stack ID.
func InstalledDirFor(id string) string {
	return filepath.Join(InstallDir, id)
}

// IsInstalled returns true when the stack directory exists.
func IsInstalled(id string) bool {
	return fileExists(filepath.Join(InstallDir, id, "stack.yaml"))
}

// ---- internal helpers ----

func isFresh(p string) bool {
	info, err := os.Stat(p)
	if err != nil {
		return false
	}
	return time.Since(info.ModTime()) < CacheTTL
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func httpGet(url string) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url) // #nosec G107 — URL is constructed from a fixed rawBaseURL constant
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d fetching %s", resp.StatusCode, url)
	}
	return io.ReadAll(resp.Body)
}

func writeCached(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o640) // #nosec G703 G306  — catalog cache files are not sensitive path is validated upstream, not user-controlled
}

func parseIndex(path string) ([]StackMeta, error) {
	b, err := os.ReadFile(path) // #nosec G304 — CacheDir is a well-known system path
	if err != nil {
		return nil, err
	}
	return parseIndexBytes(b)
}

func parseIndexBytes(data []byte) ([]StackMeta, error) {
	var idx Index
	if err := yaml.Unmarshal(data, &idx); err != nil {
		return nil, err
	}
	return idx.Stacks, nil
}

// isRunning returns true if at least one container in the stack is running.
func isRunning(dir string) bool {
	out, err := composeOutput(dir, "ps", "--services", "--filter", "status=running")
	if err != nil {
		return false
	}
	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) != "" {
			return true
		}
	}
	return false
}
