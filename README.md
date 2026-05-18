# sur

Interactive Linux/VPS hardening CLI: audit a host, select fixes in a TUI, apply them with backups, and roll back failed or unwanted changes.

`sur` is a single-binary Go tool for developers provisioning fresh Linux servers and small DevOps teams that want controlled hardening without opaque automation.

> Harden the server, show exactly what changed, require approval, and allow rollback if something breaks.

## Features

- `sur check`
  - Runs the built-in security audit.
  - Checks SSH configuration, firewall status, fail2ban, unattended upgrades, listening ports, and sudoers configuration.
  - Can run a deeper Lynis scan with `--deep`.
- `sur harden`
  - Opens an interactive Bubble Tea TUI.
  - Lets you select hardening tasks with checkboxes.
  - Runs each task through backup, execution, post-check validation, and rollback-on-failure steps.
- SQLite state tracking
  - Uses `modernc.org/sqlite`, so the binary stays CGO-free.
  - Stores sessions, task executions, rollback metadata, and audit history.
  - Defaults to `/var/lib/sur/sur.db`, or `$SUR_DB` when set.
- `sur rollback <session-id>`
  - Reverts a previous hardening session in reverse task order.
- `sur history`
  - Lists previous hardening sessions.
- `--dry-run`
  - Previews changes without modifying the system.
- `--json`
  - Emits machine-readable output for automation.
- Static Go binary
  - No CGO requirement.
  - Suitable for minimal VPS environments.

## Installation

### Install Script

```bash
curl -fsSL https://raw.githubusercontent.com/suleymanmercan/sur/main/install.sh | sudo bash
```

### Linux AMD64

```bash
curl -fsSL https://github.com/suleymanmercan/sur/releases/latest/download/sur-linux-amd64 -o /tmp/sur
sudo mv /tmp/sur /usr/local/bin/sur
sudo chmod +x /usr/local/bin/sur
```

### Linux ARM64

Use this build for ARM64 servers, Raspberry Pi, Oracle Ampere, and AWS Graviton:

```bash
curl -fsSL https://github.com/suleymanmercan/sur/releases/latest/download/sur-linux-arm64 -o /tmp/sur
sudo mv /tmp/sur /usr/local/bin/sur
sudo chmod +x /usr/local/bin/sur
```

### Build From Source

```bash
git clone https://github.com/suleymanmercan/sur.git
cd sur
go build -o sur .
sudo mv sur /usr/local/bin/sur
```

### Requirements

- Go 1.22 or newer when building from source
- Linux system
- `sudo` or root access for hardening operations

## Quick Start

Run a security audit:

```bash
sur check
```

Run a deeper audit with Lynis:

```bash
sur check --deep
```

Install Lynis automatically when it is missing:

```bash
sudo sur check --deep --install-lynis
```

Preview hardening changes:

```bash
sudo sur harden --dry-run
```

Open interactive hardening mode:

```bash
sudo sur harden
```

Run all hardening tasks without the TUI:

```bash
sudo sur harden --yes
```

Run selected tasks only:

```bash
sudo sur harden --only disable_root_ssh,ssh_password_auth_off
```

Use a custom task directory or state database:

```bash
sudo sur harden --tasks ./tasks --state ./sur.db
```

View history:

```bash
sur history
```

Roll back a previous session:

```bash
sudo sur rollback <session-id>
```

Emit JSON:

```bash
sur check --json
sudo sur harden --dry-run --json
```

## Commands

| Command | Purpose |
| --- | --- |
| `sur check` | Audit the host and print a security report. |
| `sur check --deep` | Include Lynis findings when Lynis is available. |
| `sur harden` | Select and apply hardening tasks interactively. |
| `sur harden --dry-run` | Show the planned hardening run without changing the host. |
| `sur harden --yes` | Apply every task without opening the TUI. |
| `sur harden --only <ids>` | Apply only comma-separated task IDs. |
| `sur harden --resume` | Resume the latest unfinished session. |
| `sur rollback <session-id>` | Roll back a session in reverse execution order. |
| `sur history` | List previous sessions. |

## Task System

Hardening operations are defined as YAML task files under `tasks/`.

Example:

```yaml
id: disable_root_ssh
name: "Disable SSH root login"
description: "Sets PermitRootLogin no in /etc/ssh/sshd_config and restarts sshd."
rollback_possible: true
backup_files:
  - /etc/ssh/sshd_config
risk_level: low
distros: [ubuntu, debian, rocky, alma, fedora]

pre_check:
  command: "grep -Eiq '^[#[:space:]]*PermitRootLogin[[:space:]]+yes' /etc/ssh/sshd_config"
  expect_exit: 0

exec:
  - command: "sed -ri 's/^#?\\s*PermitRootLogin\\s+.*/PermitRootLogin no/' /etc/ssh/sshd_config"
  - command: "systemctl restart ssh 2>/dev/null || systemctl restart sshd"

post_check:
  command: "grep -Eq '^PermitRootLogin[[:space:]]+no' /etc/ssh/sshd_config"
  expect_exit: 0

rollback:
  - command: "cp {backup_path} /etc/ssh/sshd_config"
  - command: "systemctl restart ssh 2>/dev/null || systemctl restart sshd"
```

Each task follows this lifecycle:

```text
pre-check
backup
execute
post-check
success or rollback
```

## Built-In Tasks

| Task ID | Description | Risk |
| --- | --- | --- |
| `disable_root_ssh` | Sets `PermitRootLogin no` in SSH config. | low |
| `ssh_password_auth_off` | Sets `PasswordAuthentication no` for key-based SSH auth. | medium |
| `enable_ufw` | Allows SSH and enables UFW. | high |
| `install_fail2ban` | Installs and enables fail2ban. | low |
| `unattended_upgrades` | Enables automatic security updates on Debian/Ubuntu. | low |

## Supported Distributions

| Distribution | Family | Package Manager |
| --- | --- | --- |
| Ubuntu | Debian | `apt` |
| Debian | Debian | `apt` |
| Rocky Linux | RHEL | `dnf` |
| AlmaLinux | RHEL | `dnf` |
| CentOS | RHEL | `dnf`/`yum` |
| Fedora | Fedora | `dnf` |
| openSUSE | SUSE | `zypper` |

## Architecture

```text
sur/
├── cmd/                # Cobra CLI commands
│   ├── root
│   ├── check
│   ├── harden
│   ├── rollback
│   └── history
├── internal/
│   ├── osdetect/       # /etc/os-release parser
│   ├── checker/        # Security checks
│   ├── lynis/          # Lynis integration and parsing
│   ├── engine/         # Task lifecycle engine
│   ├── store/          # SQLite persistence
│   ├── tui/            # Bubble Tea interface
│   └── common/         # Shared models/types
├── tasks/              # YAML hardening tasks
├── docs/               # Static landing/docs page
└── Makefile
```

## Development

Build the binary:

```bash
make build
```

Run tests:

```bash
make test
```

Run `go vet`:

```bash
make lint
```

Install locally:

```bash
sudo make install
```

Clean build output:

```bash
make clean
```

## Safety Notes

- Run `sur harden --dry-run` before applying changes on an important host.
- Keep a working SSH session open when changing SSH settings remotely.
- Review high-risk tasks such as firewall activation before confirming them.
- Rollback is task-dependent; tasks with `rollback_possible: false` may require manual recovery.

## Non-MVP Scope

Currently out of scope:

- SSH remote fleet management
- Multi-host orchestration
- Web dashboard
- Agent-based remediation
- Kubernetes hardening
- Full CIS benchmark automation
