# Komutlar

`sur`, hedef host üzerinde çalışan tek binary bir CLI'dır. Günlük kullanımda akılda tutulması gereken yüzey küçüktür:

```bash
sur check
sudo sur harden
sudo sur install
sudo sur stack
sur history
```

Root gerektiren değişikliklerde `sudo` kullanılır. Normal akışta task ID ezberlemen, JSON çıktısı üretmen veya özel state dosyası seçmen gerekmez.

## Sade Akış

| Komut | Ne yapar? | Ne zaman kullanılır? |
| --- | --- | --- |
| `sur check` | Sunucunun durumunu raporlar | Önce mevcut tabloyu görmek için |
| `sudo sur harden` | Güvenlik task'larını TUI'da seçtirir | SSH, firewall, fail2ban gibi ayarları uygulamak için |
| `sudo sur install` | Temel server setup task'larını TUI'da seçtirir | Yeni VPS hazırlarken |
| `sudo sur stack` | Docker Compose ortamlarını yönetir | Postgres, Redis, monitoring kurarken |
| `sur history` | Geçmiş oturumları listeler | Ne uygulandığını görmek için |
| `sudo sur rollback <id>` | Desteklenen değişiklikleri geri alır | Bir session'ı toparlamak gerektiğinde |

> [!TIP]
> İlk VPS akışı için yeterli sıra: `sur check`, sonra `sudo sur harden`, sonra gerekirse `sudo sur install`.

## Güvenlik Denetimi

```bash
sur check
```

`sur check` sunucudaki temel güvenlik durumunu raporlar:

- SSH root login
- SSH password authentication
- SSH port durumu
- Firewall durumu
- fail2ban servisi
- Otomatik güvenlik güncellemeleri
- Açık dinleyen soketler
- sudoers `NOPASSWD` girişleri
- Kurulu (installed) stack'lerin sağlık durumu

Bu komut sistemde değişiklik yapmaz. Salt okunurdur.

## Güvenlik Task'ları

```bash
sudo sur harden
```

`sur harden`, uygulanabilir güvenlik task'larını TUI içinde gösterir. Kullanıcı seçim yapar, sonra canlı ilerleme ekranında adımlar ve çıktılar izlenir.

Emin değilsen önce preview çalıştırabilirsin:

```bash
sudo sur harden --dry-run
```

## Sunucu Kurulumu

```bash
sudo sur install
```

`sur install`, fresh server hazırlığında kullanılan setup task'larını TUI içinde seçtirir. Örnek task'lar:

- swap dosyası
- Docker Engine
- Caddy
- temel CLI paketleri
- sistem paket güncellemesi

Emin değilsen önce preview çalıştırabilirsin:

```bash
sudo sur install --dry-run
```

## Stack Yönetimi

```bash
sudo sur stack
```

`sur stack`, TUI tabanlı bir Docker Compose ortam yöneticisidir. Hazır (official) katalog üzerinden PostgreSQL, Redis gibi servisleri kurabilir veya kendi özel stack'lerinizi oluşturabilirsiniz. Parola (secret) oluşturma ve `.env` düzenleme süreçlerini otomatikleştirir. 

Daha fazla detay için [Stack Yönetimi](/stack-yonetimi) sayfasına bakabilirsiniz.

## Geçmiş ve Geri Alma

Önceki oturumları görmek için:

```bash
sur history
```

Bir oturumu geri almak için:

```bash
sudo sur rollback <session-id>
```

Rollback her task için garanti değildir. Config dosyası değiştiren task'lar genelde geri alınabilir; paket kurulumu, firewall değişikliği veya servis kurulumu bazı durumlarda manuel toparlama gerektirebilir.

## Canlı Çıktı

`sur harden` veya `sur install` çalışırken task picker kapandıktan sonra canlı ilerleme ekranı açılır:

```text
  sur - running tasks          Task 2 / 5  ████████░░░░░  40%

  ✓  disable_root_ssh                         0.3s
  ▶  install_fail2ban
     $ apt-get install -y fail2ban
     Reading package lists...
  ○  enable_ufw
```

| İkon | Anlam |
| --- | --- |
| `▶` | Şu an çalışıyor |
| `✓` | Başarılı |
| `✗` | Başarısız |
| `↺` | Rollback yapıldı |
| `·` | Atlandı |
| `○` | Henüz başlamadı |

## Gelişmiş Kullanım

Bu bölüm normal kullanım için şart değildir. Otomasyon, CI veya özel task geliştirme sırasında işe yarar.

| Bayrak | Nerede? | Ne işe yarar? |
| --- | --- | --- |
| `--dry-run` | `harden`, `install` | Değişiklik yapmadan adımları gösterir |
| `--only <id,id>` | `harden`, `install` | Sadece belirtilen task'ları çalıştırır |
| `--yes` / `--all` | `harden`, `install` | TUI açmadan uygulanabilir tüm task'ları seçer |
| `--tasks <dizin>` | `harden`, `install` | Özel task dizini yükler |
| `--state <dosya>` | `harden`, `install` | Özel SQLite state dosyası kullanır |
| `--json` | destekleyen komutlar | Makine okunabilir JSON çıktı üretir |
| `--deep` | `check` | Lynis ile daha derin audit çalıştırır |
| `--install-lynis` | `check --deep` | Lynis yoksa kurmayı dener |

Örnekler:

```bash
sudo sur harden --dry-run
sudo sur harden --only enable_ufw
sur check --json
sudo sur check --deep --install-lynis
```

> [!IMPORTANT]
> `--yes` ve `--all`, TUI seçimini atlar. Kritik sunucularda önce normal TUI akışını veya `--dry-run` çıktısını kullanmak daha kontrollüdür.

## Özel Task'lar

Kendi YAML veya Lua task'larını yazıyorsan iki yol vardır:

```bash
sudo sur harden --tasks /path/to/tasks
sudo sur install --tasks /path/to/install_tasks
```

Sistem genelinde kalıcı task'lar için:

```text
/etc/sur/tasks/
/etc/sur/install_tasks/
```

Task yazma detayları için [Task Sistemi](/task-sistemi), [YAML Rehberi](/yaml-rehberi) ve [Lua Rehberi](/lua-rehberi) sayfalarına bak.
