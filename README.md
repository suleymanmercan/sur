# sur

`sur` is a local-first Linux/VPS hardening and setup assistant.

It audits a server, shows risky defaults, lets you choose supported fixes in a TUI, applies selected tasks locally, records the session in SQLite, and supports rollback where a task can safely be reversed.

The project is currently **beta-quality**. It is useful for controlled VPS hardening, but it is not yet a formal CIS/STIG compliance tool or an enterprise fleet-management platform.

documentation live on: https://suleymanmercan.github.io/sur/

## What It Does

- Runs local security checks with `sur check`.
- Finds common VPS risks:
  - SSH root login
  - SSH password authentication
  - default SSH port
  - inactive or missing firewall
  - missing or inactive fail2ban
  - missing automatic security updates
  - listening sockets
  - sudoers `NOPASSWD` entries
- Opens an interactive hardening picker with `sur harden`.
- Opens an interactive server setup picker with `sur install`.
- Filters tasks before showing them:
  - unsupported OS tasks are hidden
  - already-satisfied tasks are hidden
  - only applicable tasks are shown
- Runs selected task commands directly on the host.
- Stores sessions, task status, rollback data, and history in SQLite.
- Installs as a single static Go binary.

## What It Is Not

`sur` is not Ansible, OpenSCAP, Wazuh, Lynis, Nessus, or a CIS benchmark implementation.

It does not manage remote fleets, run a web dashboard, or guarantee full compliance. The goal is a practical local workflow for developers and small teams preparing Linux servers.

## Install

Recommended install:

```bash
curl -fsSL https://raw.githubusercontent.com/suleymanmercan/sur/main/install.sh | sudo bash
```

Update an existing install:

```bash
curl -fsSL https://raw.githubusercontent.com/suleymanmercan/sur/main/install.sh | sudo bash -s -- --update
```

The update flow downloads the latest release asset for the detected Linux architecture, verifies its `.sha256` checksum, and replaces `/usr/local/bin/sur`. It does not delete `/var/lib/sur` state.

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
go build -trimpath -ldflags "-s -w" -o sur .
sudo install -m 0755 sur /usr/local/bin/sur
```

## Quick Start

Run a basic audit:

```bash
sur check
```

Run a deeper audit with Lynis:

```bash
sur check --deep
```

Install Lynis automatically when missing:

```bash
sudo sur check --deep --install-lynis
```

Preview hardening actions:

```bash
sudo sur harden --dry-run
```

Open the interactive hardening TUI:

```bash
sudo sur harden
```

Run selected hardening tasks only:

```bash
sudo sur harden --only enable_ufw,install_fail2ban
```

Open the interactive install/setup TUI:

```bash
sudo sur install
```

Run selected install tasks only:

```bash
sudo sur install --only configure_swap,install_docker,install_caddy
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
| `sur harden --all`                 | Apply all applicable hardening tasks without prompting.                |
| `sur harden --only <ids>`          | Apply only comma-separated hardening task IDs.                         |
| `sur harden --resume`              | Resume the latest unfinished session.                                  |
| `sur harden --tasks <dir>`         | Load hardening tasks from a custom directory.                          |
| `sur harden --state <path>`        | Use a custom SQLite state database.                                    |
| `sur install`                      | Select and apply server setup tasks interactively.                     |
| `sur install --dry-run`            | Show selected install actions without changing the host.               |
| `sur install --yes`                | Apply all applicable install tasks without the TUI.                    |
| `sur install --all`                | Apply all applicable install tasks without prompting.                  |
| `sur install --only <ids>`         | Apply only comma-separated install task IDs.                           |
| `sur install --tasks <dir>`        | Load install tasks from a custom directory.                            |
| `sur install --state <path>`       | Use a custom SQLite state database.                                    |
| `sur rollback <session-id>`        | Roll back a session in reverse task order where rollback is supported. |
| `sur history`                      | List previous sessions.                                                |
| `--json`                           | Emit machine-readable JSON for supported commands.                     |

## Built-In Hardening Tasks

| Task ID                 | What it does                                                        | Risk   | Rollback |
| ----------------------- | ------------------------------------------------------------------- | ------ | -------- |
| `disable_root_ssh`      | Sets `PermitRootLogin no` and restarts SSH after validation.        | low    | yes      |
| `ssh_password_auth_off` | Sets `PasswordAuthentication no` and restarts SSH after validation. | medium | yes      |
| `enable_ufw`            | Installs UFW when needed, allows SSH, and enables the firewall.     | high   | no       |
| `install_fail2ban`      | Installs fail2ban and enables the service.                          | low    | partial  |
| `unattended_upgrades`   | Enables automatic security updates on Debian/Ubuntu.                | low    | yes      |

## Built-In Install Tasks

| Task ID          | What it does                                                                          | Risk   | Rollback |
| ---------------- | ------------------------------------------------------------------------------------- | ------ | -------- |
| `server_basics`  | Installs common CLI packages such as curl, git, unzip, jq, htop, and CA certificates. | low    | no       |
| `configure_swap` | Creates `/swapfile`, enables it, and persists it in `/etc/fstab`.                     | medium | yes      |
| `install_docker` | Installs Docker Engine and the Compose plugin from Docker's package repository.       | medium | no       |
| `install_caddy`  | Installs and enables the Caddy web server.                                            | low    | no       |

Swap size defaults to `2G`. Override it with:

```bash
sudo SUR_SWAP_SIZE=4G sur install --only configure_swap
```

or:

```bash
sudo SUR_SWAP_MB=4096 sur install --only configure_swap
```

## Task System

Hardening tasks live under `tasks/`.

Install/setup tasks live under `install_tasks/`.

Both directories are embedded into the Go binary with `go:embed`. `sur` does not call Ansible. It loads YAML task definitions and runs selected task steps directly on the local host with `sh -c`.

Each task follows this lifecycle:

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

or:

```bash
SUR_DB=./sur.db sudo -E sur harden
```

Rollback is task-dependent. Config-file tasks can usually be rolled back because `sur` stores backup data. Package installs and firewall changes may require manual recovery and are marked accordingly.

## Safety Notes

- Always run `sur check` before applying changes.
- Always run `sur harden --dry-run` on important servers.
- Keep an active SSH session open before changing SSH or firewall settings.
- Review high-risk tasks such as `enable_ufw` before applying them remotely.
- Do not treat the score as a compliance certificate.
- Manual-review findings, such as many listening ports, should not be blindly auto-fixed.

## Supported Systems

The code detects these Linux families:

| Distribution | Family | Package manager |
| ------------ | ------ | --------------- |
| Debian       | Debian | `apt`           |
| Ubuntu       | Debian | `apt`           |
| Rocky Linux  | RHEL   | `dnf`           |
| AlmaLinux    | RHEL   | `dnf`           |
| Fedora       | Fedora | `dnf`           |
| openSUSE     | SUSE   | `zypper`        |

Current production-readiness note: Debian/Ubuntu paths are the primary target. RHEL/Fedora/openSUSE support exists in detection and selected task commands, but every distro path still needs real VM smoke testing before a public production claim.

## Development

Build:

```bash
make build
```

Test:

```bash
make test
```

Run `go vet`:

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

The lightweight documentation site lives under `docs/`.

```bash
cd docs
npm install
npm run docs:dev
```

## Project Status

`sur` is a strong beta. The next work should focus on:

- release pipeline
- real OS smoke tests
- check-to-task mapping
- better TUI apply/result screen
- safer Go implementations for critical SSH/firewall tasks
- clearer rollback/history reporting
