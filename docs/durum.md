# Proje Durumu ve Vizyonu

`sur`, modern bulut ortamlarında çalışan VPS ve sunucular için tasarlanmış, local-first, bağımsız çalışabilen bir Linux hardening ve kurulum asistanıdır.

Geliştiriciler ve DevOps ekipleri için karmaşık güvenlik süreçlerini hafifletmek, tekrarlanabilir sunucu şablonları oluşturmak ve güvenlik durumunu kolayca denetlemek amacıyla tasarlanmıştır.

## Bugünkü Durum

| Alan | Durum |
| --- | --- |
| Core CLI | Kullanılabilir: `check`, `harden`, `install`, `history`, `rollback` |
| Release | GitHub Release + GoReleaser archive + `checksums.txt` akışı var |
| Dokümantasyon | VitePress site ile yayınlanıyor |
| Task sistemi | Embedded YAML/Lua + `/etc/sur/*` + local override destekli |
| Testler | Unit test, vet, golangci-lint, gosec ve govulncheck CI yüzeyi var |
| Üretim denemesi | Çoklu dağıtım VM smoke test matrisi hâlâ yol haritasında |

## Temel Yetkinlikler

- **Tekil Binary Mimari:** Harici kütüphane veya dil çalışma ortamı gerektirmeden hızlıca kurulur ve çalıştırılır.
- **Yerel Öncelikli (Local-First):** Merkezi bir sunucuya ihtiyaç duymadan, tüm işlemlerinizi doğrudan hedef host üzerinde yürütür.
- **Etkileşimli TUI (Terminal User Interface):** Güvenlik adımlarını ve kurulacak bileşenleri kolayca seçebileceğiniz görsel arayüz sunar.
- **İşletim Sistemi ve Pre-check Akıllı Filtreleme:** Sistem durumunu ve dağıtım türünü algılayarak yalnızca ilgili ve gerekli task'ları gösterir.
- **SQLite Oturum ve Geçmiş Yönetimi:** Tüm oturumları, uygulanan task durumlarını ve rollback verilerini yerel bir SQLite veritabanında kaydeder.
- **Çok Yönlü Komut Seti:** `check`, `harden`, `install`, `rollback` ve `history` komutları ile uçtan uca yönetim sağlar.
- **Hibrit Task Yönetimi (Truly Hybrid Loading):** Gömülü (embedded) task'lar ile yerel/sistem dizinlerindeki (`/etc/sur/tasks`) özel kuralları pürüzsüzce birleştirir ve override desteği sunar.
- **Lua Script Desteği:** Statik YAML dosyalarının yetersiz kaldığı karmaşık akışlar için güçlü ve dinamik Lua betikleri yazma imkanı tanır.

## Doğru Konumlandırma

`sur` bir compliance platformu, SIEM veya fleet-management ürünü değildir. Daha dar ve pratik bir işi hedefler: geliştiricinin yeni bir VPS'i hızlıca kontrol etmesi, temel riskleri görmesi, güvenli varsayılanları seçerek uygulaması ve yaptığı değişiklikleri yerel history içinde takip etmesi.

| Değil | Evet |
| --- | --- |
| Merkezi agent sistemi | Local-first CLI |
| Sertifikasyon aracı | Pratik audit ve hardening yardımcısı |
| Her portu otomatik kapatan araç | Operatöre risk sinyali veren araç |
| Her dağıtımda tam garanti | Debian/Ubuntu odaklı, diğer ailelerde genişleyen destek |

## Geliştirme Yol Haritası ve Yakın Plan Hedefler

Projenin kararlılığını ve yetenek setini artırmak adına aşağıdaki başlıklar aktif bir şekilde geliştirilmektedir:

- **Çoklu Dağıtım Test Matrisi (VM Smoke Tests):** Debian, Ubuntu, RHEL, Fedora ve openSUSE üzerinde otomatik sanal makine test altyapısının kurulması.
- **Gelişmiş TUI Oturum Arayüzü:** Tamamlandı — Task'lar çalışırken canlı ilerleme ekranı, satır satır komut çıktısı akışı ve progress bar eklendi.
- **Check → Auto-Fix Eşleşmesi:** `sur check` bulgularının, sistemdeki uygun düzeltme (hardening) task'ları ile doğrudan eşleştirilerek kullanıcılara önerilmesi.
- **GitHub Release Entegrasyonu:** Release pipeline süreçlerinin otomatik testler eşliğinde uçtan uca doğrulanması.

## Tasarım Felsefesi ve Konumlandırma

`sur`, bulut üzerinde VPS ve sunucu ayağa kaldıran **geliştiriciler için pratik, hızlı ve güvenilir bir güvenlik asistanı** olarak konumlandırılmıştır. 

Büyük ölçekli kurumsal SIEM, fleet-management veya merkezi uyumluluk platformlarının getirdiği hantallık ve bağımlılıklardan kaçınarak; hafif, geliştirici dostu ve hızlı bir yerel iş akışı sunmaya odaklanır.
