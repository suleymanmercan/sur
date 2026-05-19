# sur Production Roadmap

This roadmap tracks the path from the current beta-quality CLI to a production-ready public release.

## Current State

`sur` is now a solid beta:

- Single static Go binary.
- `check`, `harden`, `install`, `rollback`, and `history` command surface.
- Embedded YAML task definitions.
- Interactive TUI task picker.
- SQLite session/history tracking.
- Install/update script with checksum verification.
- OS and pre-check filtering before showing runnable tasks.

It is suitable for controlled use on your own VPS servers, but it still needs release automation, real distro testing, stronger task-result UX, and tighter safety guarantees before a public production release.

## Phase 1: Release Pipeline & CI

Goal: make install/update real, repeatable, and trustworthy, while ensuring baseline code quality.

Tasks:

- Add GitHub Actions build workflow.
- Add CI workflow with `golangci-lint`, `go test`, and `gosec` (Go Security Checker).
- Build Linux AMD64 and ARM64 binaries.
- Generate `.sha256` files for every release asset.
- Publish release artifacts automatically from tags.
- Verify `install.sh` works against an actual GitHub release.
- Document the release command/process.

Exit criteria:

- CI pipeline validates code quality and security on every PR.
- A tagged release produces:
  - `sur-linux-amd64`
  - `sur-linux-amd64.sha256`
  - `sur-linux-arm64`
  - `sur-linux-arm64.sha256`
- `curl .../install.sh | sudo bash` installs the latest release.
- `curl .../install.sh | sudo bash -s -- --update` replaces the binary without deleting state.

## Phase 2: Real OS Smoke Tests

Goal: stop guessing whether task commands work across distros.

Tasks:

- Test on Debian 12.
- Test on Ubuntu 22.04.
- Test on Ubuntu 24.04.
- Test on Rocky Linux or AlmaLinux.
- Record exact command output for:
  - `sur check`
  - `sur harden --dry-run`
  - selected `sur harden` tasks
  - `sur install --dry-run`
  - selected `sur install` tasks
- Fix package-manager or service-name differences found during testing.

Exit criteria:

- Supported distro list reflects tested reality.
- Debian/Ubuntu hardening tasks work end to end.
- RHEL-family task support is either proven or explicitly limited.
- Known distro gaps are documented.

## Phase 3: Check-to-Task Mapping

Goal: make `sur check` and `sur harden` feel connected instead of separate systems.

Tasks:

- Add an auto-fix mapping for findings.
- Show which findings can be fixed automatically.
- Show which findings require manual review.
- Avoid telling users to run `sur harden` for findings that have no automatic task.
- Add task IDs to relevant remediation output.
- Keep warnings such as `ports.listening` as manual-review items.

Exit criteria:

- `sur check` can clearly say:
  - auto-fix available
  - manual review required
  - informational only
- `sur harden` only presents tasks that are applicable and actually needed.
- No already-passing task appears in the TUI.

## Phase 4: Better TUI Apply Flow

Goal: avoid the current "select, apply, exit" feeling.

Tasks:

- Add an apply progress screen.
- Show currently running task.
- Show success, skipped, failed, and rolled-back tasks in the TUI.
- Keep final results visible instead of immediately dropping the user back to shell.
- Add final actions:
  - run `sur check` again
  - show session ID
  - exit
- Make skipped tasks understandable in the UI.

Exit criteria:

- User can see what happened without reading raw shell logs.
- Failed tasks show the failing step and exit code.
- Session ID is easy to copy/use for rollback.

## Phase 5: Harden Critical Task Implementations & Idempotency

Goal: reduce fragile shell-string behavior in risky areas and ensure tasks are robust.

Tasks:

- Move SSH config edits into Go helpers.
- Preserve file permissions when editing config files.
- Validate SSH config before restart.
- Detect whether the SSH service is `ssh` or `sshd`.
- Improve UFW task safety around active SSH sessions.
- Make package installation helpers reusable.
- Add unit tests for config editing helpers.
- Add strict YAML task schema validation (fail gracefully on bad task definitions).
- Audit and ensure all core tasks are perfectly idempotent.

Exit criteria:

- SSH hardening does not rely on long `sed` commands.
- Critical config edits are test-covered.
- Service restart behavior is distro-aware.
- Failed validation prevents applying unsafe changes.
- Invalid YAML tasks produce clear errors on startup instead of panics.
- Running `sur harden --yes` multiple times produces no errors and leaves the system in the same expected state.

## Phase 6: Rollback, State Quality & Logging

Goal: make rollback expectations honest, reliable, and easy to debug.

Tasks:

- Separate rollback-capable tasks from non-rollbackable tasks in the UI.
- Store more task metadata in SQLite.
- Store command output or failure snippets for debugging.
- Add rollback tests for real backup/restore paths.
- Make install tasks clearly marked as non-rollbackable when package removal is not safe.
- Improve `sur history` output with task counts and status summary.
- Implement centralized file logging (e.g., `/var/log/sur/sur.log`) with debug/info/error levels.
- Add `--debug` / `--verbose` CLI flags for detailed output.

Exit criteria:

- Rollback behavior is predictable.
- Non-rollbackable tasks are obvious before apply.
- History gives enough context to understand past runs.
- Users can provide a `.log` file for bug reports.

## Phase 7: Documentation, Community & Beta Release

Goal: make the project usable and contributable by someone who did not read the source code.

Tasks:

- Update README command examples.
- Add a safety guide for remote VPS use.
- Add a task authoring guide.
- Add distro support notes.
- Add release/update/uninstall guide.
- Add troubleshooting section.
- Add a soft version check to notify users of new releases.
- Add open-source community files (`CONTRIBUTING.md`, Issue/PR templates).

Exit criteria:

- A new user can install, check, dry-run, harden, update, and uninstall from docs alone.
- Known risks are documented.
- Beta release notes explain what is supported and what is not.
- Users are gracefully notified of updates.
- Community contribution guidelines are clear.

## Phase 8: Production Release Gate

Goal: decide whether the project is ready for public production use.

Checklist:

- Release pipeline is working.
- Checksums are published and verified.
- Debian and Ubuntu smoke tests pass.
- At least one RHEL-family distro is tested or marked limited.
- TUI apply flow is understandable.
- Critical SSH tasks are test-covered.
- Install/update/uninstall docs are accurate.
- Rollback limitations are explicit.
- No task claims support for a distro where it was not tested.

Production release criteria:

- No known task can lock the user out without an explicit warning.
- No supported distro path is untested.
- Install/update flow works from a clean server.
- `sur check` output does not promise automatic fixes for manual-only findings.
