# sur

> Interactive Linux/VPS hardening CLI — audit, pick fixes in a TUI, apply with backup + rollback.

`sur` is a single-binary Go tool aimed at developers spinning up a fresh server
and small DevOps teams who want to say:

> "Harden my box — but show me what changed, let me approve it, and let me
> roll it back if something breaks."

## Highlights

- **`sur check`** — built-in audit (SSH, firewall, fail2ban, unattended-upgrades,
  listening ports, sudoers). Optional Lynis integration via `--deep`.
- **`sur harden`** — Bubble Tea TUI: pick tasks with checkboxes, apply with
  per-task backup → exec → post-check → rollback lifecycle.
- **State in SQLite** (`modernc.org/sqlite`, pure-Go, static binary) at
  `/var/lib/sur/sur.db`. Every session and task execution is recorded.
- **`sur rollback <session-id>`** — undo a previous run, in reverse order.
- **`sur history`** — list past sessions.
- **`--json`** on every command for CI pipelines.
- **`--dry-run`** to preview changes without touching the system.

---

## Install

### One-liner (Linux amd64)

```bash
curl -fsSL https://github.com/suleymanmercan/sur/releases/latest/download/sur-linux-amd64 -o /tmp/sur \
  && sudo mv /tmp/sur /usr/local/bin/sur \
  && sudo chmod +x /usr/local/bin/sur
```

### ARM64 (Raspberry Pi, Oracle Ampere, AWS Graviton)

```bash
curl -fsSL https://github.com/suleymanmercan/sur/releases/latest/download/sur-linux-arm64 -o /tmp/sur \
  && sudo mv /tmp/sur /usr/local/bin/sur \
  && sudo chmod +x /usr/local/bin/sur
```

### Build from source

```bash
git clone https://github.com/suleymanmercan/sur.git
cd sur
go build -o sur .
sudo mv sur /usr/local/bin/sur
```

> Requires Go 1.22+. No CGO, no external dependencies.

---

## Quick start

```bash
# Audit the system (no root needed)
sur check

# Preview what would change
sudo sur harden --dry-run

# Interactive TUI — pick and apply
sudo sur harden

# CI / headless mode
sudo sur harden --yes

# Undo a previous session
sudo sur rollback <session-id>

# List past sessions
sur history
```

---

## Project layout

```
sur/
├── cmd/                # cobra commands (root, check, harden, rollback, history)
├── internal/
│   ├── osdetect/       # /etc/os-release parser
│   ├── checker/        # built-in security checks
│   ├── lynis/          # lynis wrapper + report parser
│   ├── engine/         # YAML task loader + lifecycle engine
│   ├── store/          # SQLite persistence (modernc.org/sqlite)
│   ├── tui/            # Bubble Tea picker
│   └── common/         # shared types (Finding, Report, Severity, ...)
├── tasks/              # *.yaml hardening tasks (drop-in, embedded into binary)
└── Makefile
```

---

## Adding a task

Drop a YAML file into `tasks/`:

```yaml
id: disable_root_ssh
name: "Disable SSH root login"
rollback_possible: true
backup_files: [/etc/ssh/sshd_config]
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

`{backup_path}` is substituted at rollback time with the file `sur` backed up before applying.

Task YAML files are embedded into the binary at build time — no extra files needed on the target machine.

---

## Supported distros

| Distro | Family | Package manager |
|--------|--------|----------------|
| Ubuntu / Debian | debian | apt |
| Rocky Linux / AlmaLinux / CentOS | rhel | dnf |
| Fedora | fedora | dnf |
| openSUSE | suse | zypper |

---

## Testing

```bash
make test
```

---

## What's NOT in the MVP

- SSH remote management
- Terraform / cloud provider integration
- OpenSCAP / CIS Benchmark
- Web UI / multi-host fleet management