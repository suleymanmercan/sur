# YAML Task Yazma Rehberi

`sur` task'larının en yaygın biçimi YAML'dır. Bir `.yaml` dosyası oluşturup `tasks/` (hardening) veya `install_tasks/` (setup) dizinine atıldığında binary'e embed edilmeden önce veya `--tasks` ile runtime'da yüklenerek doğrudan kullanılabilir.

## Tam Alan Referansı

```yaml
id: ""                  # zorunlu — benzersiz kimlik; --only ve history'de kullanılır
name: ""                # TUI listesinde gösterilecek başlık
description: ""         # TUI'da seçili satırın altında görünen kısa açıklama
risk_level: "low"       # "low" | "medium" | "high"
rollback_possible: false
distros: []             # boş → tüm dağıtımlarda göster
backup_files: []        # exec öncesi kaydedilecek dosya yolları (yalnızca ilk var olan yedeklenir)

pre_check:
  command: ""           # exit kodunu döndüren shell komutu
  expect_exit: 0        # pre_check bu kodu döndürürse task TUI'da gösterilir

exec:
  - command: ""
    expect_exit: 0      # varsayılan 0; farklıysa belirt

post_check:
  command: ""           # 0 dışındaki dönüş başarısızlık sayılır
  expect_exit: 0

rollback:
  - command: ""         # {backup_path} token'ı otomatik değiştirilir
```

---

## Alan Detayları

### `id`

Task'ın kalıcı kimliğidir. Bir kez belirlendikten sonra değiştirme — `sur history` kayıtları ve `--only` bayrakları buna bağlıdır.

```yaml
id: disable_root_ssh
```

Önerilen format: `küçük_harf_snake_case`, açıklayıcı.

### `distros`

Boş bırakılırsa task tüm sistemlerde görünür. Değer girilirse `sur` OS'u algılar ve eşleşmeyenleri filtreler.

```yaml
distros: [ubuntu, debian]           # Sadece Debian ailesi
distros: [rocky, alma, fedora]      # Sadece RHEL ailesi
distros: []                         # Tüm dağıtımlar
```

Tanınan değerler: `ubuntu`, `debian`, `rocky`, `almalinux`, `fedora`, `opensuse`.

### `pre_check`

Bu alan "sistem sağlıklı mı?" kontrolü **değildir**. "Bu task'a hâlâ ihtiyaç var mı?" sorusunu yanıtlar.

`command` beklenen exit kodunu döndürürse task TUI'da görünür ve çalıştırılır. Beklenen kodu döndürmezse "zaten yapılmış" sayılır ve atlanır.

```yaml
# SSH root login açıksa (exit 0) → task göster
pre_check:
  command: "! grep -Eiq '^PermitRootLogin no' /etc/ssh/sshd_config"
  expect_exit: 0
```

> [!TIP]
> `pre_check` başarısız olan task'lar `--yes` ve `--all` modlarında da atlanır. Bunu bir savunma katmanı olarak kullan — aynı task iki kez çalıştırılmaz.

### `exec`

Task'ın asıl işini yapan adımlar. Her adım `sh -c` ile çalıştırılır. **Yeni:** Komutun çıktısı artık satır satır canlı olarak TUI'ya akar — uzun süren işlemler için kullanıcı beklerken ne olduğunu görür.

```yaml
exec:
  - command: "apt-get install -y fail2ban"
  - command: "systemctl enable --now fail2ban"
```

Bir adım beklenen exit kodunu döndürmezse o adımda durulur ve rollback tetiklenir.

### `post_check`

`exec` bittikten sonra başarıyı doğrular. `expect_exit` dışında bir kod gelirse task `FAILED` olarak işaretlenir.

```yaml
post_check:
  command: "systemctl is-active fail2ban"
  expect_exit: 0
```

### `rollback`

`{backup_path}` token'ı otomatik olarak `backup_files` listesinin ilk var olan dosyasının yedek yoluyla değiştirilir. Go runner yedek dosyayı yerleştirdikten sonra buradaki komutlar çalışır.

```yaml
rollback:
  - command: "cp {backup_path} /etc/ssh/sshd_config"
  - command: "systemctl restart ssh 2>/dev/null || systemctl restart sshd"
```

> [!WARNING]
> `rollback_possible: true` işaretliysen rollback komutlarını mutlaka test et. Başarısız bir rollback veriyi yarım bırakabilir.

---

## Tam Örnek: `disable_root_ssh.yaml`

```yaml
id: disable_root_ssh
name: "Disable SSH root login"
description: "Sets PermitRootLogin no in /etc/ssh/sshd_config and restarts sshd."
rollback_possible: true
backup_files:
  - /etc/ssh/sshd_config
risk_level: low
distros: [ubuntu, debian, rocky, alma, fedora]

pre_check:
  command: "! (sshd -T 2>/dev/null | grep -Eiq '^permitrootlogin[[:space:]]+no\\b' || grep -Eiq '^[[:space:]]*PermitRootLogin[[:space:]]+no\\b' /etc/ssh/sshd_config)"
  expect_exit: 0

exec:
  - command: "grep -Eiq '^[#[:space:]]*PermitRootLogin[[:space:]]+' /etc/ssh/sshd_config && sed -ri 's/^[#[:space:]]*PermitRootLogin[[:space:]]+.*/PermitRootLogin no/' /etc/ssh/sshd_config || printf '\\nPermitRootLogin no\\n' >> /etc/ssh/sshd_config"
  - command: "sshd -t"
  - command: "systemctl restart ssh 2>/dev/null || systemctl restart sshd"

post_check:
  command: "grep -Eiq '^[[:space:]]*PermitRootLogin[[:space:]]+no\\b' /etc/ssh/sshd_config"
  expect_exit: 0

rollback:
  - command: "cp {backup_path} /etc/ssh/sshd_config"
  - command: "systemctl restart ssh 2>/dev/null || systemctl restart sshd"
```

---

## Sık Yapılan Hatalar

### ❌ `pre_check`'i ters yazmak

```yaml
# YANLIŞ: "sistem iyi mi?" sorusunu soruyor
pre_check:
  command: "systemctl is-active ufw"
  expect_exit: 0
```

Bu durumda ufw aktifse task TUI'da görünür — ama zaten aktifse çalıştırmak istemiyoruz!

```yaml
# DOĞRU: "task'a ihtiyaç var mı?" sorusunu soruyor
pre_check:
  command: "systemctl is-active ufw"
  expect_exit: 1  # aktif değilse (exit 1) → task göster
```

Veya `!` ile tersle:

```yaml
pre_check:
  command: "! systemctl is-active ufw"
  expect_exit: 0  # ufw aktif değilse → exit 0 → task göster
```

### ❌ `backup_files` olmadan `rollback_possible: true`

```yaml
# YANLIŞ: rollback komutu çalışır ama geri yüklenecek dosya yok
rollback_possible: true
backup_files: []
rollback:
  - command: "cp {backup_path} /etc/ssh/sshd_config"  # backup_path boş!
```

Rollback desteği sunuyorsan mutlaka `backup_files` doldur ya da dosya restore etme adımını rollback komutuna elle yaz.

### ❌ Tek satırda çok iş

```yaml
# YANLIŞ: Hata ayıklaması zor
exec:
  - command: "apt-get update && apt-get install -y ufw && ufw allow ssh && ufw --force enable"
```

Her mantıksal adımı ayrı bırak:

```yaml
# DOĞRU: Hangi adım başarısız oldu anında görünür
exec:
  - command: "apt-get update"
  - command: "apt-get install -y ufw"
  - command: "ufw allow ssh"
  - command: "ufw --force enable"
```

---

## İpuçları

- **`sshd -t` kullan:** SSH config değiştiren her task'ta `sshd -t` ile syntax kontrolü yap. Canlı sistemde SSH bağlantısını kaybetmemenin en güvenli yolu.
- **İdempotent yaz:** Task iki kez çalıştırılsa bile sistemi bozmamalı. `pre_check` bunu büyük ölçüde garanti eder ama `exec` adımları da güvende yazılmalı (`sed -i` yerine `grep && sed || echo`).
- **`risk_level: high` koy:** Geri alınamaz işlemleri (firewall, port değişikliği) doğru risk seviyesiyle işaretle. TUI `⚠ no rollback` uyarısını gösterir.
