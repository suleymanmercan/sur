# Task Sistemi

`sur`, task'ları YAML veya Lua dosyalarından okur ve binary içine embed eder.

- **Hardening task'ları:** `tasks/`
- **Install/setup task'ları:** `install_tasks/`

`sur` Ansible çalıştırmaz. Seçilen task'lardaki komutlar local host üzerinde doğrudan `sh -c` ile çalıştırılır ve çıktılar **canlı olarak** TUI ekranına akar.

---

## Çalışma Sırası

```text
task yükle (embedded → sistem dizini → yerel dizin → --tasks)
OS uyumluluğunu kontrol et  (distros listesi)
pre_check çalıştır          → zaten yapıldıysa TUI'da gösterme
backup_files dosyalarını yedekle
exec adımlarını çalıştır    → çıktı satır satır TUI'a akar
post_check ile doğrula
session sonucunu SQLite'a yaz
hata olursa rollback (mümkünse)
```

---

## Alan Referansı

| Alan | Tip | Açıklama |
|---|---|---|
| `id` | string | Zorunlu. Benzersiz kimlik — `--only` ve history'de kullanılır. Değiştirme. |
| `name` | string | TUI listesinde gösterilen başlık. |
| `description` | string | TUI'da seçili satırın altında görünen kısa açıklama. |
| `risk_level` | string | `"low"`, `"medium"` veya `"high"`. TUI'da renklendirilir. |
| `rollback_possible` | bool | Rollback desteği var mı? Yoksa TUI `⚠ no rollback` gösterir. |
| `distros` | list | Task'ın gösterileceği dağıtımlar. Boş → tüm sistemler. |
| `backup_files` | list | exec öncesi yedeklenecek dosya yolları. İlk var olan yedeklenir. |
| `pre_check.command` | string | Çalıştırılan shell komutu. |
| `pre_check.expect_exit` | int | Beklenen exit kodu. Eşleşirse task çalıştırılabilir. |
| `exec[]` | list | Asıl değişikliği yapan adımlar. Her biri `sh -c` ile çalışır. |
| `post_check` | object | exec sonrası başarı doğrulama. |
| `rollback[]` | list | Geri alma komutları. `{backup_path}` token'ı otomatik değiştirilir. |

---

## Neden `pre_check` Ters Görünüyor?

`pre_check`, "sistem iyi mi?" değil **"bu task'a hâlâ ihtiyaç var mı?"** sorusunu sorar.

**Örnek:** `disable_root_ssh` task'ı yalnızca root login hâlâ açıksa TUI'da görünmelidir. Root login zaten kapalıysa `pre_check` komutu beklenen kodu döndürmez → task "zaten yapılmış" sayılır → TUI'da gösterilmez.

Bu sayede `sur harden --yes` iki kez çalıştırılsa bile aynı işi tekrar yapmaz.

---

## Canlı Çıktı Akışı

Task'lar çalışırken terminal ekranına şöyle bir görünüm yansır:

```
  sur — hardening              Task 2 / 5  ████████░░░░░  40%

  ✓  disable_root_ssh                              0.3s
  ▶  install_fail2ban          (çalışıyor)
     $ apt-get install -y fail2ban
     Reading package lists...
     Get:1 http://archive.ubuntu.com/ubuntu ...
     Unpacking fail2ban ...
     Setting up fail2ban ...
  ○  enable_ufw
  ○  sysctl_hardening
```

Her exec adımı (`$ komut`) ve o adımın stdout/stderr çıktısı satır satır gerçek zamanlı olarak akışlandırılır. CI/pipe ortamında veya `--json` modunda TUI gösterilmez; çıktılar stderr'e yazılır.

---

## Hibrit Task Yükleme

`sur`, task'ları şu öncelik sırasına göre yükler ve birleştirir:

1. **Embedded:** Binary içindeki gömülü task'lar
2. **Sistem dizini:** `/etc/sur/tasks/` veya `/etc/sur/install_tasks/`
3. **Yerel dizin:** Çalıştırılan dizindeki `./tasks/` veya `./install_tasks/`

Aynı `id`'ye sahip task varsa sonraki kaynak öncekini override eder. Binary'yi yeniden derlemeden gömülü task'ı özelleştirebilirsin.

> [!TIP]
> Kurumsal ortamlarda `/etc/sur/tasks/` altına şirket özel task'larını koyarak tüm sunucularda merkezi bir kural seti yönetebilirsin.

### Harici Dizin: `--tasks`

```bash
sudo sur harden --tasks /path/to/custom_tasks
```

`--tasks` ile belirtilen dizin, varsayılan kaynaklara **ek olarak** yüklenir ve aynı ID'ye sahip task'ları override eder.

---

## Task Türleri

Sur hem YAML hem Lua task'larını destekler. Her iki tür de aynı lifecycle'ı paylaşır.

| | YAML | Lua |
|---|---|---|
| **En iyi olduğu yer** | Shell komutları dizisi | Dosya okuma/yazma, koşullu mantık |
| **Sözdizimi** | Bildirimsel | Programatik |
| **Dry-run** | Otomatik desteklenir | `run()` ve `write_file()` otomatik simüle edilir |
| **Hata mesajı** | Komut çıktısı + exit kodu | Fonksiyon dönüş değeri |

Detaylı rehberler için:
- [YAML Task Yazma Rehberi](/yaml-rehberi)
- [Lua Task Yazma Rehberi](/lua-rehberi)

---

## State ve Rollback

`sur` session, task durumu ve rollback verisini SQLite'ta saklar:

```text
/var/lib/sur/sur.db
```

Özel bir yol kullanmak için:

```bash
sudo sur harden --state ./sur.db
# veya
SUR_DB=./sur.db sudo -E sur harden
```

Rollback task bağımlıdır:
- Config dosyası değiştiren task'lar genelde tam geri alınabilir
- Paket kurulumu ve firewall değişiklikleri manuel toparlama gerektirebilir — `⚠ no rollback` işaretine dikkat et
