# sur Stack Catalog Plan

Goal: add a simple `sudo sur stack` flow for Docker Compose based development and monitoring tools without turning the CLI into a large command surface.

## Core Decision

- Keep stack templates in this repo first.
- Install live stack instances under `/opt/sur/stacks/<stack-id>/`.
- Use one main user entrypoint: `sudo sur stack`.
- Let the TUI collect and edit config values so users do not need to hunt for `.env` and Compose files manually.
- Keep manual editing possible: installed files remain visible under `/opt/sur/stacks/<stack-id>/`.

## Repository Template Layout

```text
catalog/
  stacks/
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

Do not split `install.lua`, `update.lua`, and `backup.lua` at first. Use one `stack.lua` with optional functions:

```lua
function install(ctx) end
function update(ctx) end
function backup(ctx) end
function status(ctx) end
```

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

This is the source of truth after installation. Users may manually edit `compose.yml` or `.env`, but the normal path should be TUI config editing.

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

Initial config field types:

- `text`
- `number`
- `select`
- `bool`
- `secret`

Secret behavior:

- Generate when empty and `generate: true`.
- Write to `secrets/<id>.txt`.
- Use file mode `0600`.
- Never print secret values in logs, status, or TUI summary.

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
  PostgreSQL
    Bind host: 127.0.0.1
    Port: 5432
    Database name: app
    Username: app
    Password: generated
    Install
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

Keep advanced direct subcommands optional/internal. The normal user path should remain `sudo sur stack`.

## Safety Rules

- Default bind host must be `127.0.0.1`, not public.
- Public bind `0.0.0.0` must be explicit in TUI.
- Avoid image tag `latest`; pin major versions like `postgres:16`, `redis:7`.
- Never run `docker compose down -v` from normal flows.
- Do not delete `data/`, `secrets/`, or `backups/` unless a future destructive action has an explicit confirm flow.
- Run `docker compose config` before `up`.
- For updates, default to manual and warn before DB major upgrades.
- Store stack state in the existing `/var/lib/sur/sur.db` path when needed.

## Phase Plan

### Phase 1: Stack Skeleton

- Add `cmd/stack.go` with `sudo sur stack`.
- Add `internal/stack` package.
- Discover repo-local templates from `catalog/stacks`.
- Discover installed stacks from `/opt/sur/stacks`.
- Show a minimal TUI with installable and installed lists.

### Phase 2: Template Install

- Add `catalog/stacks/postgres` and `catalog/stacks/redis`.
- Copy selected template into `/opt/sur/stacks/<id>`.
- Generate `.env`, `secrets/`, `data/`, and `backups/`.
- Validate with `docker compose config`.
- Run `docker compose up -d` after confirmation.

### Phase 3: Config TUI

- Parse `stack.yaml` config fields.
- Render TUI inputs for `text`, `number`, `select`, `bool`, and `secret`.
- Write config to `.env` and secrets to `secrets/*.txt`.
- Support editing installed stack config and optionally restarting.

### Phase 4: Lifecycle Actions

- Add status, logs, restart, down, backup, update actions inside the TUI.
- Use `stack.lua` hooks where present.
- Keep dangerous operations hidden or unavailable in the first version.

### Phase 5: check Integration

- Make `sudo sur check` report installed stack health only.
- Do not install or mutate stacks from `check`.
- Example output:

```text
Stacks:
  postgres: running
  redis: stopped
  uptime-kuma: not installed
```

### Phase 6: External Catalog Later

Only after the local catalog flow works:

```text
/etc/sur/catalogs/local/stacks/<stack-id>/
```

Potential future TUI item:

```text
Fetch/update catalog
```

Do not start with a separate GitHub repo. Split to `sur-stacks` later only if the catalog becomes large or community-maintained.

## First Implementation Scope

Start with:

- `sudo sur stack`
- repo-local `catalog/stacks`
- `/opt/sur/stacks`
- `postgres`
- `redis`
- install, status, logs, edit config, restart, down

Defer:

- remote catalog fetching
- destructive uninstall/purge
- Watchtower/Portainer/n8n/Grafana
- advanced networking automation
- DB restore flow
