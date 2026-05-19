---
# https://vitepress.dev/reference/default-theme-home-page
layout: home

hero:
  name: "sur"
  text: "Local-first VPS hardening"
  tagline: "Audit et, uygulanabilir fixleri seç, dry-run gör, rollback destekli task'ları kayıt altında çalıştır."
  actions:
    - theme: brand
      text: "Nasıl Kurulur?"
      link: "/kurulum"
    - theme: alt
      text: "Komutlar"
      link: "/komutlar"

features:
  - title: "Check"
    details: "SSH, firewall, fail2ban, automatic updates, listening ports ve sudoers gibi temel VPS risklerini raporlar."
  - title: "Harden"
    details: "OS ve pre-check sonucuna göre sadece gerekli task'ları TUI içinde gösterir."
  - title: "Install"
    details: "Swap, Docker, Caddy ve temel server paketleri gibi fresh-server setup task'larını seçilebilir hale getirir."
  - title: "History"
    details: "Session, task sonucu ve rollback datasını SQLite içinde kayıt altında tutar."
---

## Beta Durumu

`sur`, kontrollü VPS kullanımı için güçlü bir beta seviyesindedir. Public production release için gerçek distro smoke testleri ve daha iyi apply/result ekranı hâlâ yapılmalıdır.
