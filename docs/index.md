---
# https://vitepress.dev/reference/default-theme-home-page
layout: home

hero:
  name: "sur"
  text: "Linux hardening, with consent"
  tagline: "Audit, pick fixes in a TUI, apply with per-task backup + rollback."
  actions:
    - theme: brand
      text: "Nasıl Kurulur?"
      link: "/kurulum"
    - theme: alt
      text: "GitHub'da Gör"
      link: "https://github.com/suleymanmercan/sur"

features:
  - title: "Consent-first (Onay Odaklı)"
    details: "Sisteme dokunmadan önce her change gösterilir, check edilir ve onay ister."
  - title: "Reversible (Geri Alınabilir)"
    details: "Her task config backup alır ve rollback kaydı oluşturur. Tek komut ile bütün oturum geri alınabilir."
  - title: "Auditable (Denetlenebilir)"
    details: "SQLite (/var/lib/sur/sur.db) içinde her session, task ve backup blob kayıt altına alınır."
  - title: "Static Binary (Tek Dosya)"
    details: "Pure-Go SQLite driver sayesinde CGO veya hiçbir ekstra kurulum gerektirmez. Tek dosyadır."
---
