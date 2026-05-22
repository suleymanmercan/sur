# sur

`sur` is a local-first Linux/VPS hardening and setup assistant.

It audits a server, shows risky defaults, lets you choose fixes in a TUI, applies selected tasks directly on the host, records sessions in SQLite, and supports rollback where possible. It also manages Docker Compose based development stacks through the `sur stack` command.

Documentation: https://suleymanmercan.github.io/sur/

## What It Does

- Audits the host with `sur check` — SSH, firewall, fail2ban, auto-updates, sudoers, listening ports, and installed stack health.
- Opens an interactive hardening picker with `sur harden`.
- Opens an interactive server setup picker with `sur install`.
- Manages Docker Compose stacks (databases, services, monitoring) with `sur stack`.
- Filters tasks before showing them — unsupported OS and already-satisfied tasks are hidden.
- Runs selected task commands directly on the host.
- Stores sessions, task status, rollback data, and history in SQLite.
- Installs as a single static Go binary.

## Install

Recommended install:

```bash
curl -fsSL https://raw.githubusercontent.com/suleymanmercan/sur/main/install.sh | sudo bash
```

Update an existing install:

```bash
curl -fsSL https://raw.githubusercontent.com/suleymanmercan/sur/main/install.sh | sudo bash -s -- --update
```

The update flow downloads the latest release archive for the detected OS/architecture, verifies it against `checksums.txt`, and replaces `/usr/local/bin/sur`. It does not delete `/var/lib/sur` state.

Uninstall only the binary:

```bash
curl -fsSL https://raw.githubusercontent.com/suleymanmercan/sur/main/install.sh | sudo bash -s -- --uninstall
```

Remove the binary and local state:

```bash
curl -fsSL https://raw.githubusercontent.com/suleymanmercan/sur/main/install.sh | sudo bash -s -- --uninstall --purge
```

Build from source:

```bash
git clone https://github.com/suleymanmercan/sur.git
cd sur
make build
sudo install -m 0755 sur /usr/local/bin/sur
```

## Quick Start

Run a security audit:

```bash
sur check
```

Run a deeper audit with Lynis:

```bash
sur check --deep
```

Open the interactive hardening TUI:

```bash
sudo sur harden
```

Open the interactive install/setup TUI:

```bash
sudo sur install
```

Open the interactive stack manager:

```bash
sudo sur stack
```

View previous sessions:

```bash
sur history
```

Roll back a previous session:

```bash
sudo sur rollback <session-id>
```

## Command Reference

| Command                            | Purpose                                                                |
| ---------------------------------- | ---------------------------------------------------------------------- |
| `sur check`                        | Audit the host and print a security report.                            |
| `sur check --deep`                 | Include Lynis findings when Lynis is available.                        |
| `sur check --deep --install-lynis` | Install Lynis first if it is missing.                                  |
| `sur harden`                       | Select and apply hardening tasks interactively.                        |
| `sur harden --dry-run`             | Show selected hardening actions without changing the host.             |
| `sur harden --yes`                 | Apply all applicable hardening tasks without the TUI.                  |
| `sur harden --only <ids>`          | Apply only comma-separated hardening task IDs.                         |
| `sur harden --resume`              | Resume the latest unfinished session.                                  |
| `sur harden --tasks <dir>`         | Load hardening tasks from a custom directory.                          |
| `sur harden --state <path>`        | Use a custom SQLite state database.                                    |
| `sur install`                      | Select and apply server setup tasks interactively.                     |
| `sur install --dry-run`            | Show selected install actions without changing the host.               |
| `sur install --yes`                | Apply all applicable install tasks without the TUI.                    |
| `sur install --only <ids>`         | Apply only comma-separated install task IDs.                           |
| `sur install --tasks <dir>`        | Load install tasks from a custom directory.                            |
| `sur install --state <path>`       | Use a custom SQLite state database.                                    |
| `sur stack`                        | Open the interactive Docker Compose stack manager.                     |
| `sur rollback <session-id>`        | Roll back a session in reverse task order where rollback is supported. |
| `sur history`                      | List previous sessions.                                                |
| `--json`                           | Emit machine-readable JSON for supported commands.                     |

## Stack Manager

`sur stack` provides an interactive TUI to install and manage Docker Compose based development and monitoring stacks.

```bash
sudo sur stack
```

### How It Works

The stack catalog is fetched at runtime from GitHub:

```
https://raw.githubusercontent.com/suleymanmercan/sur/main/catalog/stacks/
```

Templates are cached under `/var/cache/sur/catalog/` with a 24-hour TTL. The "Fetch / update catalog" menu item forces a refresh.

Installed stacks live under `/opt/sur/stacks/<stack-id>/`.

### Official Stacks

| Stack      | Image        | Description                          |
| ---------- | ------------ | ------------------------------------ |
| PostgreSQL | `postgres:16` | Local PostgreSQL database            |
| Redis      | `redis:7`     | In-memory key-value store with AOF   |

### User Custom Stacks

Place a valid stack directory under `/etc/sur/stacks/<stack-id>/`:

```text
/etc/sur/stacks/my-app/
  stack.yaml
  compose.yml
  stack.lua     (optional)
```

Custom stacks appear in the TUI with a `[custom]` label and are never overwritten by catalog updates.

### Stack Lifecycle Actions

From the TUI, each installed stack supports:

| Action        | Description                                          |
| ------------- | ---------------------------------------------------- |
| Status        | Show container states.                               |
| Logs          | Display the last 80 lines of compose logs.           |
| Edit config   | Update `.env` values via the TUI form.               |
| Restart       | `docker compose restart`.                            |
| Backup        | Copy `data/` and `secrets/` to `backups/<timestamp>` |
| Update        | Pull latest images and restart.                      |
| Stop (down)   | `docker compose down` (data is preserved).           |

### Safety Guarantees

- Default bind host is always `127.0.0.1` — public binding requires explicit selection.
- Image tags are pinned (`postgres:16`, `redis:7`) — `latest` is never used.
- `docker compose down -v` is never run from normal flows.
- `data/`, `secrets/`, and `backups/` directories are never deleted automatically.
- Compose config is validated with `docker compose config` before every `up`.

### stack.yaml Format

```yaml
id: mystack
name: My Stack
description: A custom stack
risk_level: low

config:
  - id: port
    label: Port
    type: number
    default: "8080"

  - id: password
    label: Password
    type: secret
    generate: true
```

Config field types: `text`, `number`, `select`, `bool`, `secret`.

Secret fields are written to `secrets/<id>.txt` with mode `0600` and are never printed in logs or the TUI.

### stack.lua Hooks

Each stack can optionally provide lifecycle hooks in `stack.lua`:

```lua
function install(ctx)
  ctx.log("Custom install step.")
end

function update(ctx)  end
function backup(ctx)  end
function status(ctx)  end
```

`ctx.dir` contains the installed stack directory. `ctx.log(msg)` writes a log line.

## Built-In Security Checks (`sur check`)

| Check ID             | Category    | What It Checks                              |
| -------------------- | ----------- | ------------------------------------------- |
| `ssh.root_login`     | SSH         | `PermitRootLogin` is disabled               |
| `ssh.password_auth`  | SSH         | `PasswordAuthentication` is off             |
| `ssh.port`           | SSH         | SSH is not on default port 22               |
| `firewall.ufw`       | Firewall    | UFW is active                               |
| `firewall.firewalld` | Firewall    | firewalld is active                         |
| `fail2ban.active`    | Brute-force | fail2ban is installed and running           |
| `updates.auto`       | Updates     | Automatic security updates are configured   |
| `ports.listening`    | Network     | Listening socket count                      |
| `sudo.nopasswd`      | Sudo        | No `NOPASSWD` entries in sudoers            |
| `stacks.*`           | Stacks      | Running state of installed sur stacks (INFO)|

`sur check` is read-only and never modifies the system.

## Built-In Hardening Tasks

| Task ID                 | What it does                                                        | Risk   | Rollback |
| ----------------------- | ------------------------------------------------------------------- | ------ | -------- |
| `disable_root_ssh`      | Sets `PermitRootLogin no` and restarts SSH after validation.        | low    | yes      |
| `ssh_password_auth_off` | Sets `PasswordAuthentication no` and restarts SSH after validation. | medium | yes      |
| `enable_ufw`            | Installs UFW when needed, allows SSH, and enables the firewall.     | high   | no       |
| `install_fail2ban`      | Installs fail2ban and enables the service.                          | low    | partial  |
| `unattended_upgrades`   | Enables automatic security updates on Debian/Ubuntu.                | low    | yes      |

## Built-In Install Tasks

| Task ID          | What it does                                                                           | Risk   | Rollback |
| ---------------- | -------------------------------------------------------------------------------------- | ------ | -------- |
| `server_basics`  | Installs common CLI packages such as curl, git, unzip, jq, htop, and CA certificates. | low    | no       |
| `configure_swap` | Creates `/swapfile`, enables it, and persists it in `/etc/fstab`.                      | medium | yes      |
| `install_docker` | Installs Docker Engine and the Compose plugin from Docker's package repository.        | medium | no       |
| `install_caddy`  | Installs and enables the Caddy web server.                                             | low    | no       |

Swap size defaults to `2G`. Override with:

```bash
sudo SUR_SWAP_SIZE=4G sur install --only configure_swap
```

## Task System

Hardening tasks live under `tasks/`. Install/setup tasks live under `install_tasks/`. Both directories are embedded into the Go binary with `go:embed`.

Each task lifecycle:

```text
load task
filter by OS
run pre_check
hide task if already satisfied
backup configured files
execute steps
run post_check
record result
rollback on failure when supported
```

Important task fields:

| Field               | Meaning                                                              |
| ------------------- | -------------------------------------------------------------------- |
| `id`                | Stable task identifier used by `--only` and session history.         |
| `name`              | Human-readable TUI label.                                            |
| `description`       | Short explanation shown in the picker.                               |
| `distros`           | OS IDs/families where the task is applicable.                        |
| `pre_check`         | Command that returns the expected exit code when the task is needed. |
| `exec`              | Commands to apply the task.                                          |
| `post_check`        | Command that verifies the task succeeded.                            |
| `backup_files`      | Files saved before mutation.                                         |
| `rollback`          | Commands used to reverse the task when possible.                     |
| `rollback_possible` | Whether rollback should be offered/trusted for that task.            |

## State and Rollback

By default, `sur` stores state in:

```text
/var/lib/sur/sur.db
```

Override it with:

```bash
sudo sur harden --state ./sur.db
```

Rollback is task-dependent. Config-file tasks can usually be rolled back because `sur` stores backup data. Package installs and firewall changes may require manual recovery and are marked accordingly.

## Safety Notes

- Always run `sur check` before applying changes.
- Always run `sur harden --dry-run` on important servers.
- Keep an active SSH session open before changing SSH or firewall settings.
- Review high-risk tasks such as `enable_ufw` before applying them remotely.
- Do not treat the security score as a compliance certificate.
- Manual-review findings should not be blindly auto-fixed.

## Supported Systems

| Distribution | Family | Package manager |
| ------------ | ------ | --------------- |
| Debian       | Debian | `apt`           |
| Ubuntu       | Debian | `apt`           |
| Rocky Linux  | RHEL   | `dnf`           |
| AlmaLinux    | RHEL   | `dnf`           |
| Fedora       | Fedora | `dnf`           |
| openSUSE     | SUSE   | `zypper`        |

Debian and Ubuntu are the primary development targets. RHEL/Fedora/openSUSE support exists for system detection and selected tasks.

## Development

Build:

```bash
make build
```

Test:

```bash
make test
```

Lint:

```bash
make lint
```

Install locally from source:

```bash
sudo make install
```

Uninstall local binary:

```bash
sudo make uninstall
```

Remove local binary and state:

```bash
sudo make purge
```

## Documentation

The documentation site lives under `docs/`:

```bash
cd docs
npm install
npm run docs:dev
```
