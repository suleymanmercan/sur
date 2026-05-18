// Package osdetect parses /etc/os-release and identifies the running distro
// so other components can pick the correct package manager.
package osdetect

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Family represents a coarse-grained Linux distribution family.
type Family string

const (
	FamilyDebian  Family = "debian"
	FamilyRHEL    Family = "rhel"
	FamilyFedora  Family = "fedora"
	FamilySUSE    Family = "suse"
	FamilyArch    Family = "arch"
	FamilyUnknown Family = "unknown"
)

// OSInfo describes the running operating system.
type OSInfo struct {
	ID         string // e.g. "ubuntu", "debian", "rocky", "almalinux", "fedora"
	IDLike     string // e.g. "debian", "rhel fedora"
	Name       string // pretty name
	VersionID  string // e.g. "22.04", "9.3"
	Family     Family
	PkgManager string // apt, dnf, yum, zypper, pacman, unknown
}

// Supported returns true when the engine has good support for the family.
func (o OSInfo) Supported() bool {
	switch o.Family {
	case FamilyDebian, FamilyRHEL, FamilyFedora:
		return true
	}
	return false
}

// Detect reads /etc/os-release and returns parsed OSInfo.
// On non-Linux or missing /etc/os-release it returns an OSInfo
// flagged as unknown rather than an error.
func Detect() (*OSInfo, error) {
	return detectFromFile("/etc/os-release")
}

func detectFromFile(path string) (*OSInfo, error) {
	info := &OSInfo{Family: FamilyUnknown, PkgManager: "unknown"}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return info, nil
		}
		return info, fmt.Errorf("read %s: %w", path, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		val = strings.Trim(val, `"`)
		switch key {
		case "ID":
			info.ID = strings.ToLower(val)
		case "ID_LIKE":
			info.IDLike = strings.ToLower(val)
		case "NAME", "PRETTY_NAME":
			if info.Name == "" || key == "PRETTY_NAME" {
				info.Name = val
			}
		case "VERSION_ID":
			info.VersionID = val
		}
	}
	if err := scanner.Err(); err != nil {
		return info, err
	}

	info.Family, info.PkgManager = classify(info.ID, info.IDLike)
	return info, nil
}

func classify(id, idLike string) (Family, string) {
	tokens := append([]string{id}, strings.Fields(idLike)...)
	for _, t := range tokens {
		switch t {
		case "ubuntu", "debian", "raspbian", "linuxmint", "pop":
			return FamilyDebian, "apt"
		case "fedora":
			return FamilyFedora, "dnf"
		case "rhel", "centos", "rocky", "almalinux", "ol", "amzn":
			return FamilyRHEL, "dnf"
		case "opensuse", "opensuse-leap", "opensuse-tumbleweed", "sles", "suse":
			return FamilySUSE, "zypper"
		case "arch", "manjaro", "endeavouros":
			return FamilyArch, "pacman"
		}
	}
	return FamilyUnknown, "unknown"
}
