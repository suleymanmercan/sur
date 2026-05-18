package osdetect

import (
	"os"
	"path/filepath"
	"testing"
)

func writeRelease(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "os-release")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestDetect_Ubuntu(t *testing.T) {
	p := writeRelease(t, `NAME="Ubuntu"
ID=ubuntu
ID_LIKE=debian
VERSION_ID="22.04"
PRETTY_NAME="Ubuntu 22.04.3 LTS"
`)
	info, err := detectFromFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if info.Family != FamilyDebian || info.PkgManager != "apt" {
		t.Fatalf("unexpected: %+v", info)
	}
	if info.VersionID != "22.04" {
		t.Fatalf("VersionID = %q", info.VersionID)
	}
	if !info.Supported() {
		t.Fatal("expected supported")
	}
}

func TestDetect_Rocky(t *testing.T) {
	p := writeRelease(t, `NAME="Rocky Linux"
ID="rocky"
ID_LIKE="rhel centos fedora"
VERSION_ID="9.3"
`)
	info, _ := detectFromFile(p)
	if info.Family != FamilyRHEL || info.PkgManager != "dnf" {
		t.Fatalf("unexpected: %+v", info)
	}
}

func TestDetect_Missing(t *testing.T) {
	info, err := detectFromFile("/no/such/file")
	if err != nil {
		t.Fatal(err)
	}
	if info.Family != FamilyUnknown {
		t.Fatalf("expected unknown family, got %s", info.Family)
	}
}
