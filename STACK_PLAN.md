# sur Stack Catalog Plan

Goal: add a simple `sudo sur stack` flow for Docker Compose based development and monitoring tools without turning the CLI into a large command surface.

## Core Decisions

- Stack templates live in `catalog/stacks/` inside this repo.
- The binary does **not** embed the catalog. On first use (or on `Fetch/update catalog`), the app downloads templates from the GitHub raw URL.
- Install live stack instances under `/opt/sur/stacks/<stack-id>/`.
- One main user entrypoint: `sudo sur stack`.
- TUI collects and edits config values; users do not need to hunt for `.env` and Compose files manually.
- Manual editing remains possible: installed files stay visible under `/opt/sur/stacks/<stack-id>/`.
- Users can add **custom stacks** by placing a valid stack directory under `/etc/sur/stacks/<stack-id>/`.

## Catalog Distribution

Templates are fetched from GitHub raw at runtime — not embedded in the binary.

```text
Download source:
  https://raw.githubusercontent.com/suleymanmercan/sur/main/catalog/stacks/<stack-id>/<file>

Index file:
  https://raw.githubusercontent.com/suleymanmercan/sur/main/catalog/stacks/index.yaml
```

- `index.yaml` lists all available stack IDs and their display names.
- Individual template files (`stack.yaml`, `compose.yml`, `stack.lua`) are fetched on demand during install.
- Downloaded templates are cached under `/var/cache/sur/catalog/` with a configurable TTL (default: 24 h).
- `Fetch/update catalog` in the TUI re-downloads the index and refreshes the cache.

## Repository Template Layout

```text
catalog/
  stacks/
    index.yaml
    postgres/
      stack.yaml
      compose.yml
      stack.lua
    redis/
      stack.yaml
      compose.yml
      stack.lua
    mariadb/
      stack.yaml
      compose.yml
      stack.lua
    mongo/
      stack.yaml
      compose.yml
      stack.lua
    uptime-kuma/
      stack.yaml
      compose.yml
      stack.lua
```

Use one `stack.lua` per stack with optional lifecycle functions:

```lua
function install(ctx) end
function update(ctx) end
function backup(ctx) end
function status(ctx) end
```

## User Custom Stacks

Users may place their own stacks under:

```text
/etc/sur/stacks/<stack-id>/
  stack.yaml
  compose.yml
  stack.lua        (optional)
```

Discovery order when listing installable stacks:

1. `/var/cache/sur/catalog/` (official, fetched from GitHub)
2. `/etc/sur/stacks/` (user-defined, local machine)

Custom stacks are displayed with a `[custom]` label in the TUI. They are never overwritten by catalog fetch operations.

## Installed Runtime Layout

```text
/opt/sur/stacks/postgres/
  stack.yaml
  compose.yml
  stack.lua
  .env
  secrets/
    postgres_password.txt
  data/
  backups/
```

This directory is the source of truth after installation. Config values are read from `.env` by Docker Compose. Users may manually edit `compose.yml` or `.env`; the normal path is TUI config editing.

## stack.yaml Shape

```yaml
id: postgres
name: PostgreSQL
description: Local PostgreSQL database
risk_level: medium

config:
  - id: bind_host
    label: Bind host
    type: select
    default: 127.0.0.1
    options:
      - 127.0.0.1
      - 0.0.0.0
      - internal

  - id: port
    label: Port
    type: number
    default: 5432

  - id: db_name
    label: Database name
    type: text
    default: app

  - id: username
    label: Username
    type: text
    default: app

  - id: password
    label: Password
    type: secret
    generate: true
```

Config field types:

- `text`
- `number`
- `select`
- `bool`
- `secret`

### .env Generation

Each config field maps to an environment variable in the generated `.env`:

```text
BIND_HOST=127.0.0.1
PORT=5432
DB_NAME=app
USERNAME=app
PASSWORD=<generated>
```

`compose.yml` references these via `${VAR_NAME}`. On config edit, sur rewrites `.env` in place and optionally restarts the stack.

### Secret Behavior

- Generate when empty and `generate: true`.
- Write to `secrets/<id>.txt` with file mode `0600`.
- Reference in `.env` as the generated value.
- Never print secret values in logs, status output, or TUI summary.

## TUI Flow

Main command:

```bash
sudo sur stack
```

Top-level TUI:

```text
Install stack
Installed stacks
Fetch/update catalog
Quit
```

Install flow:

```text
Install stack
  [official] PostgreSQL
    Bind host: 127.0.0.1
    Port: 5432
    Database name: app
    Username: app
    Password: generated
    ── Install ──
  [custom] my-app
    ...
```

Installed stack flow:

```text
Installed stacks
  PostgreSQL
    Status
    Logs
    Edit config
    Restart
    Backup
    Update
    Down
```

Keep advanced direct subcommands optional/internal. The normal user path must remain `sudo sur stack`.

## sur check — Static Security Audit

`sur check` performs a **read-only, static security audit** of the running system. It does not install, mutate, or interact with stacks.

Audit categories:

- SSH configuration (PermitRootLogin, PasswordAuthentication, Port)
- Open ports vs expected services
- Firewall status (ufw / firewalld)
- Fail2ban or equivalent presence
- Unattended upgrades / automatic security updates
- World-writable files in sensitive paths
- Installed stack health (running / stopped / not installed)

Output format:

```text
[PASS] SSH root login disabled
[WARN] SSH password auth enabled — consider key-only auth
[FAIL] Firewall not active
[INFO] Installed stacks:
         postgres: running
         redis: stopped
```

Severity levels: `PASS`, `WARN`, `FAIL`, `INFO`.

- `FAIL` items are security concerns that should be addressed.
- `WARN` items are recommendations, not blockers.
- `INFO` items are neutral observations.
- `sur check` never suggests running another command automatically. The user decides next steps.

## Safety Rules

- Default bind host must be `127.0.0.1`, not public.
- Public bind `0.0.0.0` must be an explicit TUI selection.
- Avoid image tag `latest`; pin major versions like `postgres:16`, `redis:7`.
- Never run `docker compose down -v` from normal flows.
- Do not delete `data/`, `secrets/`, or `backups/` without an explicit confirm flow.
- Run `docker compose config` before `up` to catch template errors early.
- For updates, default to manual and warn before DB major-version upgrades.
- Store stack state in `/var/lib/sur/sur.db`.

## Phase Plan

### Phase 1: Stack Skeleton

- Add `cmd/stack.go` with `sudo sur stack`.
- Add `internal/stack` package.
- Fetch `index.yaml` from GitHub raw; cache under `/var/cache/sur/catalog/`.
- Discover user stacks from `/etc/sur/stacks/`.
- Discover installed stacks from `/opt/sur/stacks/`.
- Show a minimal TUI with installable (official + custom) and installed lists.

### Phase 2: Template Install

- Add `catalog/stacks/postgres`, `catalog/stacks/redis`, and `catalog/stacks/index.yaml`.
- Fetch template files on demand from GitHub raw during install.
- Copy fetched template into `/opt/sur/stacks/<id>/`.
- Generate `.env` from config values, `secrets/`, `data/`, and `backups/`.
- Validate with `docker compose config`.
- Run `docker compose up -d` after user confirmation.

### Phase 3: Config TUI

- Parse `stack.yaml` config fields.
- Render TUI inputs for `text`, `number`, `select`, `bool`, and `secret`.
- Write config to `.env`; write secrets to `secrets/*.txt`.
- Support editing installed stack config and optionally restarting the stack.

### Phase 4: Lifecycle Actions

- Add status, logs, restart, down, backup, and update actions in the TUI.
- Use `stack.lua` hooks where present.
- Keep dangerous operations (e.g., purge data) hidden in the first version.

### Phase 5: sur check Audit

- Implement static security audit checks (SSH, firewall, ports, unattended upgrades, world-writable paths).
- Add installed stack health as an `INFO` section.
- Output `PASS`, `WARN`, `FAIL`, `INFO` lines.
- No automatic remediation — read-only only.

### Phase 6: Catalog Growth

- Add `mariadb`, `mongo`, `uptime-kuma` to `catalog/stacks/`.
- Add TTL-based cache refresh UI in TUI (`Fetch/update catalog`).
- Split to a separate `sur-stacks` repo only if the catalog becomes large or community-maintained.

## First Implementation Scope

Start with:

- `sudo sur stack`
- GitHub raw catalog fetch + `/etc/sur/stacks/` for custom stacks
- `/opt/sur/stacks/` runtime
- `postgres`, `redis`
- install, status, logs, edit config, restart, down
- `sur check` static audit (SSH + firewall + stack health)

Defer:

- destructive uninstall / purge
- Watchtower, Portainer, n8n, Grafana
- advanced networking automation
- DB restore flow
- remote catalog from a separate repo
