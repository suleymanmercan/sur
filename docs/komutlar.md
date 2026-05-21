# Komutlar

`sur`, local host üzerinde çalışan tek binary bir CLI'dır. Root gerektiren işlemler için `sudo` ile çalıştırılır.

## Komut Haritası

| Komut | Ne zaman kullanılır? | Root gerekir mi? |
| --- | --- | --- |
| `sur check` | Mevcut güvenlik durumunu görmek için | Hayır |
| `sur check --deep` | Lynis ile daha derin audit almak için | Lynis kurulumunda evet |
| `sur harden` | Güvenlik task'larını seçip uygulamak için | Evet |
| `sur install` | Fresh server setup task'larını seçip uygulamak için | Evet |
| `sur history` | Eski session kayıtlarını görmek için | State yoluna bağlı |
| `sur rollback <id>` | Desteklenen task değişikliklerini geri almak için | Evet |

> [!TIP]
> Remote VPS üzerinde ilk kullanım için en güvenli sıra: `sur check`, `sudo sur harden --dry-run`, sonra `sudo sur harden --only <task-id>`.

## `sur check` — Güvenlik Denetimi

Sunucudaki temel güvenlik durumunu kontrol eder ve renkli bir rapor üretir.

```bash
sur check
```

**Denetlenenler:**
- SSH root login durumu
- SSH password authentication
- SSH port (varsayılan 22 açık mı?)
- Firewall (ufw / firewalld) durumu
- fail2ban servisi
- Otomatik güvenlik güncellemeleri
- Açık dinleyen soketler
- sudoers `NOPASSWD` girişleri

**Lynis ile derin denetim:**

```bash
sur check --deep
```

Lynis yoksa otomatik kurup çalıştırmak için:

```bash
sudo sur check --deep --install-lynis
```

| Bayrak | Açıklama |
|---|---|
| `--deep` | Lynis audit dahil et |
| `--install-lynis` | Lynis yoksa otomatik kur |
| `--json` | Makine okunabilir JSON çıktı |

### JSON çıktı

```bash
sur check --json | jq .report.score
```

`sur check --json` çıktısı stdout üzerinde temiz JSON döner.

## `sur harden` — Güvenlik Sıkılaştırma

Önce dry-run ile neyin değişeceğini gör:

```bash
sudo sur harden --dry-run
```

İnteraktif TUI ile task seç:

```bash
sudo sur harden
```

Yalnızca belirli task'ları çalıştır:

```bash
sudo sur harden --only enable_ufw,install_fail2ban
```

TUI açmadan tüm uygulanabilir task'ları çalıştır:

```bash
sudo sur harden --yes
```

Özel task dizini yükle:

```bash
sudo sur harden --tasks /etc/sur/custom_tasks
```

| Bayrak | Açıklama |
|---|---|
| `--dry-run` | Değişiklik yapmadan hangi adımların çalışacağını göster |
| `--yes` | TUI açmadan tüm uygulanabilir task'ları çalıştır |
| `--all` | `--yes` ile aynı |
| `--only <id,id,...>` | Yalnızca belirtilen task ID'lerini çalıştır |
| `--resume` | Son yarım kalan session'ı devam ettir |
| `--tasks <dizin>` | Özel task dizini yükle, varsayılanlarla birleştir |
| `--state <dosya>` | Özel SQLite state dosyası kullan |
| `--json` | JSON çıktı (TUI devre dışı) |

> [!IMPORTANT]
> `--yes` ve `--all`, uygulanabilir tüm task'ları seçer. İlan edilen önerilen kullanım bu değildir; kritik sunucuda önce `--dry-run`, sonra `--only` veya TUI seçimi daha kontrollüdür.

## `sur install` — Sunucu Kurulumu

Fresh server setup task'ları:

```bash
sudo sur install
```

Belirli task'ları seçerek kur:

```bash
sudo sur install --only configure_swap,install_docker,install_caddy
```

Swap boyutunu ortam değişkeniyle ayarla:

```bash
sudo SUR_SWAP_SIZE=4G sur install --only configure_swap
# veya
sudo SUR_SWAP_MB=4096 sur install --only configure_swap
```

`sur install` de aynı bayrakları destekler: `--dry-run`, `--yes`, `--only`, `--tasks`, `--state`, `--json`.

## State Dosyası

Varsayılan state yolu:

```text
/var/lib/sur/sur.db
```

Geçici deneme veya CI için özel state dosyası kullanabilirsin:

```bash
sudo sur harden --dry-run --state ./sur.db
SUR_DB=./sur.db sudo -E sur harden --dry-run
```

## `sur history` — Geçmiş Session'lar

Önceki tüm oturumları listeler:

```bash
sur history
```

---

## `sur rollback` — Geri Alma

```bash
sudo sur rollback <session-id>
```

Session ID'yi `sur history` ile öğrenebilirsin. Rollback her task için garanti değildir — config dosyası değiştiren task'lar genelde geri alınabilir; paket kurulumu ve firewall değişiklikleri manuel toparlama gerektirebilir.

## Canlı Çıktı Akışı

`sur harden` veya `sur install` çalışırken task picker kapandıktan sonra canlı bir ilerleme ekranı açılır:

```
  sur — hardening              Task 2 / 5  ████████░░░░░  40%

  ✓  disable_root_ssh                              0.3s
  ▶  install_fail2ban          (çalışıyor)
     $ apt-get install -y fail2ban
     Reading package lists...
     Get:1 http://archive.ubuntu.com/ubuntu ...
     Setting up fail2ban ...
  ○  enable_ufw
  ○  sysctl_hardening
```

| İkon | Anlam |
|---|---|
| `▶` turuncu | Şu an çalışıyor |
| `✓` yeşil | Başarılı |
| `✗` kırmızı | Başarısız |
| `↺` sarı | Rollback yapıldı |
| `·` gri | Atlandı (zaten yapılmış veya OS uyumsuz) |
| `○` gri | Henüz başlamadı |

> [!TIP]
> Her exec adımı (`$ komut`) ve o adımın stdout/stderr çıktısı satır satır gerçek zamanlı olarak akar. Uzun süren işlemlerde (`apt-get`, paket indirme vb.) ne olduğu ekrandan takip edilebilir.

---

## CI / Pipe Modu

`sur` bir terminal tespit edemezse ya da `--json` verilirse TUI gösterilmez. JSON stdout'a, task logları stderr'e yazılır; bu sayede pipe edilen JSON bozulmaz:

```bash
# CI: tüm task'ları interaktifsiz çalıştır, JSON al
sudo sur harden --dry-run --json | jq .results

# Pipe: TUI çalışmaz, log stderr'e akar
echo | sudo sur harden --only disable_root_ssh
```

---

## Global Bayraklar

`--json` bayrağı destekleyen komutlarda makine okunabilir JSON çıktısı alınır:

```bash
sur check --json
sur harden --dry-run --json
sudo sur install --dry-run --json
```
