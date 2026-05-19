# sur Market and Alternatives Analysis

Research date: 2026-05-19

This document compares `sur` with existing Linux hardening, auditing, compliance, and endpoint security tools. The goal is to understand whether `sur` has a real product angle or whether it is only a small wrapper around existing tools.

## Short Verdict

There are many serious tools in this space, but most of them are either:

- audit-only,
- compliance-heavy,
- fleet/enterprise-oriented,
- Ansible/Chef/Rudder-style configuration management,
- SIEM/XDR platforms,
- or too complex for a developer who just wants to prepare one VPS safely.

`sur` can still have a useful niche if it stays focused:

> A single-binary, local-first VPS hardening and setup assistant for developers and small teams, with check, safe remediation, install tasks, dry-run, rollback where possible, and a clear TUI.

The commercial opportunity is real but narrow. It should not try to compete head-on with Wazuh, OpenSCAP, CIS-CAT, Tenable, Rudder, or Mondoo. The better angle is developer-friendly server onboarding and repeatable VPS baseline setup.

## Alternatives

| Tool | What it does | Strength | Weakness / gap | Difference from `sur` |
| --- | --- | --- | --- | --- |
| Lynis | Local Unix/Linux security auditing, hardening guidance, compliance testing, vulnerability hints. | Mature, open source, very broad OS support, trusted by admins. | Mostly audit/guidance; remediation is not the main UX. | `sur` can use Lynis-like ideas, but focus on guided apply/remediation and fresh-server setup. |
| OpenSCAP | SCAP/NIST-standard compliance scanning and security baselines. | Serious compliance ecosystem; standard-driven; supports hardening guides and baselines. | Heavy for casual VPS users; profile/content management is not beginner-friendly. | `sur` is simpler and local-first; not a formal SCAP compliance engine. |
| Ubuntu Security Guide | Ubuntu Pro tool for CIS/DISA-STIG audit and fix workflows. | Strong for Ubuntu compliance; official Canonical path. | Ubuntu-specific and compliance-profile oriented. | `sur` can cover Debian/Ubuntu/RHEL-style VPS basics without requiring Ubuntu Pro or CIS profile knowledge. |
| CIS-CAT Pro / Lite | CIS Benchmark assessment and reporting. | Authoritative CIS ecosystem; reports and scores are the product. | Pro access depends on CIS SecureSuite membership; remediation is not the lightweight developer workflow. | `sur` should not claim CIS equivalence unless tested/mapped; it can be practical baseline hardening instead. |
| Wazuh | Open-source XDR/SIEM platform with agents, vulnerability detection, configuration assessment, FIM, compliance dashboards. | Very broad security platform; strong enterprise/fleet story. | Needs server/agent/dashboard setup; too much for one quick VPS. | `sur` is a small CLI for immediate local hardening, not monitoring/SIEM. |
| cnspec / Mondoo | Policy-as-code security/compliance scanner across OS, cloud, Kubernetes, SaaS, CI/CD. | Modern policy-as-code, broad integrations, scores/remediation guidance. | More scanner/platform than local interactive fixer; commercial platform gravity. | `sur` can be more opinionated and action-oriented for one Linux host. |
| Chef InSpec | Compliance-as-code framework for auditing systems and infrastructure. | Mature rule language and profiles; good for teams already in compliance-as-code. | Detects and reports; remediation is controlled elsewhere. | `sur` combines basic checks with direct host remediation. |
| DevSec Hardening Framework | Ansible collection for Linux, SSH, nginx, MySQL hardening; aligns with InSpec baselines. | Battle-tested Ansible ecosystem; many distro targets. | Requires Ansible mindset and playbook workflow. | `sur` targets users who do not want Ansible for one server. |
| Ansible Lockdown | Many CIS/STIG remediation roles for specific OS versions. | Deep benchmark-specific Ansible remediation. | One role per benchmark/OS; operationally heavier. | `sur` should be simpler and safer for generic VPS baseline tasks, not full CIS/STIG automation. |
| Rudder | Web-driven IT automation and compliance platform with agents, audit/enforce modes, dashboards. | Production fleet management, continuous compliance, audit logs. | Server/agent platform; aimed at fleets and organizations. | `sur` is a one-binary local tool; no central control plane. |
| osquery | Exposes OS state as SQL tables for endpoint visibility. | Excellent visibility primitive for security teams. | Query/visibility layer, not a hardening workflow. | `sur` could later borrow visibility ideas, but its job is guided setup/remediation. |
| Tenable Nessus | Vulnerability scanning and compliance auditing, including credentialed Unix/Windows audits. | Enterprise-grade scanning, reports, vulnerability/compliance coverage. | Commercial scanner, not a local TUI setup tool. | `sur` is much smaller and developer/operator focused. |
| Random hardening scripts | One-shot Bash scripts for Ubuntu/Debian hardening. | Easy to run, often covers common basics. | Usually opaque, risky, low rollback, weak state/history. | `sur` can win by being explainable, dry-runnable, stateful, and interactive. |

## Where `sur` Is Similar

`sur` overlaps with existing tools in these areas:

- Checks SSH root login, password auth, firewall, fail2ban, updates, ports, sudoers.
- Applies local remediation tasks.
- Produces a score-like report.
- Has hardening task definitions.
- Has install/setup helpers for common server components.

This means the idea is not unique at the category level. The category already exists.

## Where `sur` Can Be Different

The strongest differentiation is not "better compliance scanner." That market is already crowded and mature.

The stronger differentiation is:

- Single static binary.
- No Ansible/Chef/Rudder server required.
- Local-first, no SaaS account required.
- Interactive TUI that shows only applicable tasks.
- Beginner-friendly explanations.
- `check -> dry-run -> apply -> rollback/history` loop.
- Fresh VPS setup and hardening in one flow.
- Opinionated defaults for developers deploying small apps.

Good positioning:

> "A developer-friendly VPS hardening assistant."

Bad positioning:

> "An enterprise compliance platform."

## Commercial Potential

### Can this become commercially successful?

Yes, but only with a focused niche.

The strongest commercial paths are:

1. Open-source CLI with paid hosted dashboard later.
2. Paid "Pro task packs" for common deployment stacks.
3. Paid reports for agencies/freelancers managing client VPS servers.
4. A small SaaS that tracks server posture over time through `sur agent` or scheduled `sur check --json`.
5. One-command hardening profiles for common app stacks:
   - Docker host
   - Caddy reverse proxy
   - Laravel/PHP VPS
   - Node/Next.js VPS
   - Go service VPS
   - PostgreSQL single-server setup

The weak commercial path is trying to sell enterprise compliance against Wazuh, CIS-CAT, Tenable, Mondoo, Rudder, or OpenSCAP. They already own that language and buyer channel.

### Best target users

- Solo developers deploying to VPS.
- Small agencies managing many small client servers.
- Indie hackers and self-hosters.
- Small teams without a dedicated DevOps/security engineer.
- Turkish/local market developers who want a practical CLI in plain language.

### Weak target users

- Banks, healthcare, government, large enterprises.
- Teams that require formal CIS/STIG audit artifacts.
- Organizations already using Wazuh, Tenable, Rudder, Chef, Ansible, or OpenSCAP.

## Product Direction

The product should lean into practical VPS readiness:

- "Is this server safe enough to deploy a small app?"
- "What should I fix before going live?"
- "Can I install common baseline packages safely?"
- "Can I see exactly what changed?"
- "Can I undo risky config edits?"

Recommended next features:

- Check-to-task mapping.
- Better apply progress screen.
- Real VM smoke tests.
- Release pipeline.
- Safer SSH/UFW implementations.
- HTML/Markdown report export.
- Profiles:
  - `sur profile vps-basic`
  - `sur profile docker-host`
  - `sur profile web-server`
  - `sur profile selfhosted`

## Product Risks

- Root-level shell tasks can break servers.
- Firewall/SSH changes can lock users out.
- Package repository commands drift over time.
- Distro support claims become dangerous if not tested.
- "Security score" can create false confidence.
- Compliance language can create legal/support expectations.

Mitigations:

- Keep distro support conservative.
- Always dry-run first.
- Show lockout warnings for SSH/firewall tasks.
- Run real VM smoke tests before release.
- Separate "automatic fix" from "manual review."
- Avoid CIS/STIG compliance claims until there is benchmark mapping and evidence.

## Suggested Positioning

Short:

> `sur` is a local-first VPS hardening and setup assistant for developers.

Long:

> `sur` audits a fresh Linux server, explains the risky defaults, lets the user apply selected hardening/setup tasks through a TUI, records what changed, and supports rollback where safe. It is designed for small-server operators who want safer defaults without learning Ansible or running a full security platform.

## Source Notes

- Lynis: https://cisofy.com/lynis/
- OpenSCAP: https://www.open-scap.org/
- Ubuntu Security Guide: https://documentation.ubuntu.com/security/compliance/usg/
- CIS-CAT Pro: https://www.cisecurity.org/cybersecurity-tools/cis-cat-pro
- Wazuh platform: https://wazuh.com/platform/overview/
- DevSec Ansible hardening: https://github.com/dev-sec/ansible-collection-hardening
- cnspec / Mondoo: https://mondoo.com/cnspec
- Chef InSpec: https://docs.chef.io/inspec/
- Rudder: https://docs.rudder.io/reference/9.0/index.html
- osquery: https://github.com/osquery/osquery
- Tenable Nessus compliance checks: https://docs.tenable.com/nessus/compliance-checks-reference/
