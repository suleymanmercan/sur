# Proje Durumu

`sur` şu an güçlü bir beta seviyesindedir.

Kendi VPS'lerinde kontrollü kullanım için uygundur. Public production release için hâlâ gerçek distro testleri, release pipeline doğrulaması ve daha iyi TUI result ekranı gerekir.

## Güçlü Taraflar

- Tek binary kurulum.
- Local-first çalışma.
- TUI ile task seçimi.
- OS ve pre-check filtreleme.
- SQLite session/history kaydı.
- Check, harden, install, rollback ve history komutları.

## Henüz Eksik Olanlar

- Debian/Ubuntu/RHEL/Fedora/openSUSE üzerinde gerçek VM smoke test matrisi.
- GitHub release pipeline'ın gerçek release üstünde doğrulanması.
- `check` finding -> auto-fix task mapping.
- Apply sırasında progress/result TUI ekranı.
- SSH/firewall gibi kritik task'larda shell string yerine daha fazla Go helper.
- Rollback sınırlarının UI içinde daha görünür olması.

## Doğru Konumlandırma

`sur`, developer-friendly VPS hardening assistant olarak konumlanmalıdır.

Enterprise compliance platform, SIEM veya fleet-management sistemi gibi pazarlanmamalıdır.
