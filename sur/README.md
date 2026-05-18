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

## Quick start

```bash
make build
sudo ./sur check
sudo ./sur harden --dry-run        # preview
sudo ./sur harden                  # interactive TUI
sudo ./sur harden --yes            # CI / headless mode
sudo ./sur rollback <session-id>
./sur history
```

## Project layout

```
sur/
├── cmd/                # cobra commands (root, check, harden, rollback, history)
├── internal/
│   ├── osdetect/       # /etc/os-release parser
│   ├── checker/        # built-in security checks
│   ├── lynis/          # lynis wrapper + report parser
│   ├── engine/         # YAML task loader + lifecycle motoru
│   ├── store/          # SQLite persistence (modernc.org/sqlite)
│   ├── tui/            # Bubble Tea picker
│   └── common/         # shared types (Finding, Report, Severity, ...)
├── tasks/              # *.yaml hardening tasks (drop-in)
├── web/                # static landing/docs page
└── Makefile
```

## Adding a task

Drop a YAML file into `tasks/` (or `/etc/sur/tasks/`):

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

`{backup_path}` is substituted with the file `sur` backed up before applying.

## Testing

```bash
make test
```

All packages have unit tests; the engine has end-to-end tests covering
success, failure-with-rollback and dry-run paths.

## What's NOT in the MVP

- SSH remote management
- Terraform / cloud provider integration
- OpenSCAP / CIS benchmark
- Web UI / multi-host fleet management
