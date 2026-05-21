---
# https://vitepress.dev/reference/default-theme-home-page
layout: home

hero:
  name: "sur"
  text: "Local-first VPS hardening ve server setup"
  tagline: "Tek binary ile audit al, uygulanabilir task'ları seç, dry-run gör, canlı ilerlemeyi izle ve rollback destekli değişiklikleri SQLite history içinde sakla."
  actions:
    - theme: brand
      text: "Hemen Kur"
      link: "/kurulum"
    - theme: alt
      text: "Komutlara Bak"
      link: "/komutlar"
    - theme: alt
      text: "Güvenli Kullanım"
      link: "/guvenlik"

features:
  - title: "Check"
    details: "SSH, firewall, fail2ban, automatic updates, listening ports ve sudoers gibi temel VPS risklerini raporlar."
  - title: "Harden"
    details: "OS ve pre-check sonucuna göre sadece gerekli task'ları TUI içinde gösterir."
  - title: "Install"
    details: "Swap, Docker, Caddy ve temel server paketleri gibi fresh-server setup task'larını seçilebilir hale getirir."
  - title: "History"
    details: "Session, task sonucu ve rollback datasını SQLite içinde kayıt altında tutar."
  - title: "Lua & YAML Hibrit"
    details: "Hem statik YAML adımları hem de dinamik Lua scriptleri ile gelişmiş dosya ve shell operasyonları yapılabilir."
  - title: "Truly Hybrid Yükleme"
    details: "Embedded task'ları local/sistem dizinlerindeki (/etc/sur/tasks) task'larla otomatik birleştirir ve override desteği sunar."
---
## Ne İçin?

`sur`, yeni açılan veya elde tutulan Linux VPS'lerde ilk güvenlik kontrolünü ve tekrar edilebilir server setup adımlarını hızlandırmak için tasarlandı. Merkezi agent, web paneli veya uzak orchestrator gerektirmez; komut hedef host üzerinde çalışır ve karar kullanıcıda kalır.

<div class="sur-kpi">
  <div><strong>Tek binary</strong><span>Go ile dağıtılır; runtime bağımlılığı istemez.</span></div>
  <div><strong>Dry-run</strong><span>Önce ne yapılacağını gösterir, sonra uygular.</span></div>
  <div><strong>Rollback</strong><span>Desteklenen task'larda dosya yedeği ve session kaydı tutar.</span></div>
  <div><strong>YAML + Lua</strong><span>Basit shell akışları ve dinamik task mantığı aynı lifecycle'ı kullanır.</span></div>
</div>

## Temel Akış

<div class="sur-flow">
  <div><strong>1. Audit</strong><span><code>sur check</code> ile SSH, firewall, fail2ban, otomatik güncelleme ve port durumunu gör.</span></div>
  <div><strong>2. Preview</strong><span><code>sudo sur harden --dry-run</code> ile uygulanacak task adımlarını canlı izle.</span></div>
  <div><strong>3. Apply</strong><span>TUI'dan seç veya <code>--only</code> ile kontrollü ilerle.</span></div>
  <div><strong>4. Review</strong><span><code>sur history</code> ve gerektiğinde <code>sur rollback</code> ile session kayıtlarını kullan.</span></div>
</div>

## Hızlı Örnek

```bash
curl -fsSL https://raw.githubusercontent.com/suleymanmercan/sur/main/install.sh | sudo bash
sur check
sudo sur harden --dry-run
sudo sur harden
```

> [!IMPORTANT]
> Remote VPS üzerinde SSH, firewall veya port değiştirirken ayrı bir SSH oturumunu açık bırak. `sur` güvenli varsayılanlar sunar; production bağlamını yine operatör değerlendirir.

## Nereden Devam Etmeli?

| İhtiyaç | Sayfa |
| --- | --- |
| Kurulum, güncelleme, uninstall | [Kurulum](/kurulum) |
| CLI bayrakları ve JSON/CI kullanımı | [Komutlar](/komutlar) |
| SSH/firewall değişikliklerinde güvenli akış | [Güvenlik Notları](/guvenlik) |
| YAML veya Lua ile task yazma | [Task Sistemi](/task-sistemi) |
