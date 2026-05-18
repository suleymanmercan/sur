# sur — Linux Hardening CLI · PRD

## Original Problem Statement
`sur` adında, Go ile yazılmış, yerel çalışan interaktif bir Linux/VPS hardening CLI aracı.
Hedef kitle: yeni bir sunucu açan geliştiriciler ve küçük/orta ölçekli DevOps ekipleri.
"Sunucuyu güvenli yapayım ama neyin değiştiğini görmek, onaylamak ve hata olursa geri almak istiyorum."

## User Personas
- **Solo dev / indie hacker** açtığı yeni VPS'yi 5 dakikada güvenli hale getirmek isteyen
- **Küçük DevOps ekibi** her sunucuda aynı baseline'ı tutarlı şekilde uygulamak isteyen
- **SRE / sysadmin** rollback edebileceği auditable bir hardening pipeline isteyen

## Core Requirements
- Single static binary (Go + modernc.org/sqlite, CGO-free)
- Built-in checks: SSH (root login, password auth, port), firewall, fail2ban, unattended-upgrades, listening ports, sudo NOPASSWD
- Lynis wrapper (optional, `--deep`)
- Bubble Tea TUI for `harden`
- Per-task lifecycle: pre_check → backup → exec → post_check → rollback
- SQLite-backed session/task state at `/var/lib/sur/sur.db`
- `--dry-run`, `--yes`, `--json` flags
- OS detection (Debian/Ubuntu, RHEL/Rocky/Alma, Fedora)
- YAML task definitions in `tasks/`

## Architecture
```
sur/
├── cmd/                    cobra commands (root, check, harden, rollback, history, install)
├── internal/
│   ├── osdetect/           /etc/os-release parser + family/pkg-manager mapping
│   ├── checker/            6 built-in checks → common.Report scoring
│   ├── lynis/              shell-out + report parser
│   ├── engine/             YAML loader + task lifecycle runner
│   ├── store/              SQLite migrations + CRUD (sessions, task_executions)
│   ├── tui/                Bubble Tea checkbox picker
│   └── common/             shared types (Finding, Report, Severity, Status)
├── tasks/                  5 sample task YAMLs
├── web/                    static landing/docs page
└── Makefile
```

## What's Implemented (2026-01)
- ✅ Project scaffold (go.mod, Makefile, README)
- ✅ `osdetect` with unit tests (Ubuntu/Rocky/missing-file paths)
- ✅ `checker` 6 built-in checks + scoring + unit tests
- ✅ `store` SQLite CRUD (modernc.org/sqlite) + unit tests
- ✅ `engine` YAML loader + lifecycle runner + rollback + unit tests (success/failure/dry-run)
- ✅ `lynis` wrapper + report parser + unit test
- ✅ `tui` Bubble Tea picker (up/down, space toggle, a/n, enter, q)
- ✅ `cmd/check` colorized table + score + `--json` + `--deep`
- ✅ `cmd/harden` `--dry-run`, `--yes`, `--all`, `--only`, `--resume`, `--tasks`, `--state`
- ✅ `cmd/rollback` and `cmd/history`
- ✅ 5 task YAMLs: disable_root_ssh, ssh_password_auth_off, install_fail2ban, enable_ufw, unattended_upgrades
- ✅ Static landing/docs page (`web/index.html`)
- ✅ End-to-end CLI test: apply → file created → rollback → file removed

## Test Coverage
- `go test ./...` — all packages pass (checker, engine, lynis, osdetect, store)
- Live `sur check` validated against Debian 12 host (real findings produced)
- Live `sur harden --all` + `sur rollback` round-trip validated end-to-end

## Backlog / P1
- More task YAMLs (CrowdSec, SSH key-only, kernel sysctl baseline, audit daemon)
- `--resume` proper re-entry that skips already-applied tasks based on stored state
- Integration test that exercises `--deep` with a real lynis install
- Signed releases (goreleaser) + checksum manifest
- Bash/zsh completion (cobra already supports `completion` subcommand — ship it)

## Backlog / P2 (explicitly deferred from MVP)
- SSH remote management (run sur against another host)
- Terraform / cloud-init integration
- OpenSCAP / CIS Benchmark mapping
- Web UI / multi-host fleet dashboard
- Per-host profile management

## Key Files
- `/app/sur/cmd/check.go` — audit command, JSON + colored table output
- `/app/sur/cmd/harden.go` — TUI / `--yes` / dry-run logic + flags
- `/app/sur/internal/engine/engine.go` — lifecycle runner with rollback semantics
- `/app/sur/internal/store/store.go` — SQLite schema + CRUD
- `/app/sur/tasks/*.yaml` — drop-in task definitions
