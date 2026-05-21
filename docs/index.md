---
# https://vitepress.dev/reference/default-theme-home-page
layout: home

hero:
  name: "sur"
  text: "Local-first VPS hardening ve server setup"
  tagline: "Tek binary ile sunucuyu kontrol et, gerekli güvenlik ve kurulum task'larını TUI içinde seç, canlı ilerlemeyi izle."
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
  - title: "Kontrol Et"
    details: "SSH, firewall, fail2ban, açık portlar ve sudoers gibi temel VPS risklerini hızlıca raporlar."
  - title: "Seçerek Uygula"
    details: "Güvenlik task'larını TUI içinde seçtirir; uygulanacak adımlar canlı ekranda görünür."
  - title: "Sunucu Hazırla"
    details: "Swap, Docker, Caddy ve temel paketler gibi fresh-server kurulumlarını aynı seçilebilir akışa taşır."
  - title: "Geriye Bak"
    details: "Session geçmişini ve desteklenen rollback bilgilerini SQLite state içinde saklar."
---
## Ne İçin?

`sur`, yeni açılan veya elde tutulan Linux VPS'lerde ilk güvenlik kontrolünü ve tekrar edilebilir server setup adımlarını hızlandırmak için tasarlandı. Merkezi agent, web paneli veya uzak orchestrator gerektirmez; komut hedef host üzerinde çalışır ve karar kullanıcıda kalır.

<div class="sur-kpi">
  <div><strong>Tek binary</strong><span>Go ile dağıtılır; runtime bağımlılığı istemez.</span></div>
  <div><strong>TUI seçim</strong><span>Task ID ezberletmeden uygulanacak işleri seçtirir.</span></div>
  <div><strong>Canlı çıktı</strong><span>Uzun süren paket ve servis işlemlerini ekrandan izletir.</span></div>
  <div><strong>History</strong><span>Ne çalıştı, ne atlandı, ne geri alınabilir kayıt altında tutar.</span></div>
</div>

## Temel Akış

<div class="sur-flow">
  <div><strong>1. Kontrol</strong><span><code>sur check</code> ile SSH, firewall, fail2ban ve port durumunu gör.</span></div>
  <div><strong>2. Güvenlik</strong><span><code>sudo sur harden</code> ile uygulanacak güvenlik task'larını seç.</span></div>
  <div><strong>3. Kurulum</strong><span><code>sudo sur install</code> ile temel server setup task'larını seç.</span></div>
  <div><strong>4. Geçmiş</strong><span><code>sur history</code> ile yapılan işleri kontrol et.</span></div>
</div>

## Hızlı Örnek

```bash
curl -fsSL https://raw.githubusercontent.com/suleymanmercan/sur/main/install.sh | sudo bash
sur check
sudo sur harden
sudo sur install
```

> [!IMPORTANT]
> Remote VPS üzerinde SSH, firewall veya port değiştirirken ayrı bir SSH oturumunu açık bırak. `sur` güvenli varsayılanlar sunar; production bağlamını yine operatör değerlendirir.

## Nereden Devam Etmeli?

| İhtiyaç | Sayfa |
| --- | --- |
| Kurulum, güncelleme, uninstall | [Kurulum](/kurulum) |
| Günlük komut akışı | [Komutlar](/komutlar) |
| SSH/firewall değişikliklerinde güvenli akış | [Güvenlik Notları](/guvenlik) |
| YAML veya Lua ile task yazma | [Task Sistemi](/task-sistemi) |
