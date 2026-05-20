# Task Sistemi

`sur`, task'ları YAML dosyalarından okur ve binary içine embed eder.

- Hardening task'ları: `tasks/`
- Install/setup task'ları: `install_tasks/`

`sur` Ansible çalıştırmaz. Seçilen task'lardaki komutlar local host üzerinde doğrudan `sh -c` ile çalıştırılır.

## Çalışma Sırası

```text
task yükle
OS uyumluluğunu kontrol et
pre_check çalıştır
zaten yapılmışsa TUI'da gösterme
backup alınacak dosyaları sakla
exec adımlarını çalıştır
post_check ile doğrula
session sonucunu SQLite'a yaz
hata olursa rollback mümkünse geri al
```

## Önemli Alanlar

| Alan | Anlamı |
| --- | --- |
| `id` | Task kimliği. `--only` ve history için kullanılır. |
| `name` | TUI'da görünen isim. |
| `description` | Kısa açıklama. |
| `distros` | Task'ın uygulanacağı distro listesi. |
| `pre_check` | Task gerekli mi kontrolü. Beklenen exit code dönerse task çalıştırılabilir. |
| `exec` | Asıl değişikliği yapan komutlar. |
| `post_check` | Değişiklik başarılı mı kontrolü. |
| `backup_files` | Değişiklik öncesi saklanacak dosyalar. |
| `rollback` | Geri alma komutları. |
| `rollback_possible` | Task'ın rollback desteği var mı. |

## Neden pre_check ters gibi görünüyor?

`pre_check`, "sistem iyi mi?" kontrolü değildir. "Bu task'a ihtiyaç var mı?" kontrolüdür.

Örnek: `disable_root_ssh` task'ı sadece root login hâlâ açıksa görünmelidir. Root login zaten kapalıysa task TUI'da gösterilmez.

Bu yüzden `sur harden` içinde PASS olmuş işler tekrar seçilip boşuna çalıştırılmaz.

---

## Hibrit Task Yükleme (Hybrid Merging) ve Harici Dizinler

`sur`, sıfır-bağımlılıkla hızlıca çalışabilmesi için varsayılan task setini kendi binary'si içerisine gömülü (embedded) olarak taşır. Ancak aynı zamanda yerel sistemdeki task'ları da yükleyerek esnek ve genişletilebilir bir yapı sunar.

`sur` başlatıldığında, varsayılan olarak task'lar şu öncelik sırasına göre yüklenir ve birleştirilir (merge):

1. **Embedded Dizin**: Binary içindeki gömülü task'lar.
2. **Sistem Dizinleri**: `/etc/sur/tasks/` veya `/etc/sur/install_tasks/` altındaki `.yaml`, `.yml` ve `.lua` dosyaları.
3. **Yerel Geliştirme Dizini**: Çalıştırılan dizindeki `./tasks/` veya `./install_tasks/` klasörleri.

> [!TIP]
> Eğer yerel/sistem dizinlerindeki bir task'ın `id` değeri gömülü olan bir task ile eşleşirse, yerel sistemdeki task **gömülü olanın üzerine yazar (override eder)**. Bu sayede gömülü gelen varsayılan kuralları binary'yi yeniden derlemeden özelleştirebilirsiniz.

### Harici Dizin Kullanımı (`--tasks`)

Eğer task'ları varsayılan dizinler yerine tamamen harici, bağımsız bir klasörden yüklemek isterseniz `--tasks <klasör_yolu>` parametresini kullanabilirsiniz:

```bash
sudo sur harden --tasks /path/to/custom_tasks
```

> [!IMPORTANT]
> `--tasks <dizin_yolu>` parametresi kullanıldığında, `sur` bu dizindeki harici task'ları yükler ve varsayılan (gömülü, sistem, yerel) tüm task'lar ile **birleştirerek (merge ederek)** TUI ekranında gösterir. Aynı ID'ye sahip bir task harici dizinde de bulunuyorsa, harici dizindeki task varsayılan olanın üzerine yazar (override eder).

---

---

## Lua ile Dinamik Task Tanımlama

Statik YAML komut dizilerinin yetersiz kaldığı durumlar (örn. dosya manipülasyonu, regex arama/düzenleme, karmaşık mantıksal kontroller) için `sur`, **Lua** programlama dili ile task tanımlama desteği sunar.

Herhangi bir `.lua` uzantılı task dosyası, global değişkenler vasıtasıyla tanımlanan üst veriler (metadata) ve task yaşam döngüsünü (lifecycle) yöneten fonksiyonlardan oluşur.

### 1. Global Üst Veriler (Metadata)

Lua task dosyasının en başında tanımlanması gereken global değişkenler şunlardır:

* `id` (string): Task'ın benzersiz kimliği.
* `name` (string): TUI listesinde gösterilecek başlık.
* `description` (string): TUI'da gezinirken altta görünecek açıklama metni.
* `rollback_possible` (boolean): Geri alma desteği var mı.
* `risk_level` (string): Risk seviyesi (`"low"`, `"medium"`, `"high"`).
* `distros` (table): Desteklenen Linux dağıtımları listesi (örn. `{"debian", "ubuntu"}`). Boş bırakılırsa tüm sistemlerde çalışır.
* `backup_files` (table): Değişiklik yapılmadan önce otomatik yedeği alınacak dosya yolları listesi.

### 2. Yaşam Döngüsü Fonksiyonları (Lifecycle Functions)

* **`pre_check()`**: Bu task'ın çalıştırılmasına gerek olup olmadığını denetler. 
  * Geri dönüş: `needs_run (boolean), exit_code (number)`
  * *Örnek*: Eğer değişiklik zaten uygulanmışsa `false, 0` dönerek task'ın atlanmasını sağlayabilirsiniz.
* **`exec()`**: Değişikliği uygulayan asıl fonksiyondur.
  * Geri dönüş: Hata durumunda hata mesajı (`string`), başarı durumunda `nil` veya hata nesnesi içermeyen bir boş dönüş.
* **`post_check()`**: Uygulanan değişikliğin başarılı olup olmadığını kontrol eder.
  * Geri dönüş: Değişiklik doğrulanmışsa `nil`, başarısızsa hata mesajı (`string`).
* **`rollback(backup_path)`**: Task başarısız olduğunda veya geri alma istendiğinde çalıştırılır.
  * Parametre: `backup_path` (otomatik yedeklenen dosyanın geçici path bilgisi).
  * Geri dönüş: Hata durumunda hata mesajı (`string`), başarı durumunda `nil`.

### 3. Kullanılabilir Yardımcı Fonksiyonlar (Lua VM API)

Lua task'ları içerisinde Go runtime tarafından sunulan şu yardımcı fonksiyonlar doğrudan çağrılabilir:

* **`run(command_string)`**: Local sistemde shell komutu çalıştırır.
  * Geri dönüş: `output (string), exit_code (number)`
* **`log(message_string)`**: `sur` arayüzüne ve log akışına bilgi mesajı yazdırır.
* **`read_file(path_string)`**: Belirtilen dosyayı okur.
  * Geri dönüş: `content (string), error_message (string|nil)`
* **`write_file(path_string, content_string)`**: Belirtilen dosyayı oluşturur/yazar.
  * Geri dönüş: Hata durumunda `error_message (string)`, başarı durumunda `nil`.
* **`file_exists(path_string)`**: Dosyanın mevcut olup olmadığını denetler.
  * Geri dönüş: `exists (boolean)`

### 4. Önemli Tasarım Kuralları ve Sınırlamalar

Lua task'ları yazarken aşağıdaki kurallara dikkat edilmelidir:

* **Tek Dosya Yedekleme Sınırı**: `backup_files` listesine birden fazla dosya yolu eklenebilse de, `sur` Go runner motoru bu listedeki dosyalardan **yalnızca ilk var olanı** yedekler. Eğer birden fazla dosyayı yedeklemeniz gerekiyorsa, bunu `exec()` ve `rollback()` fonksiyonları içinde yardımcı shell komutları vasıtasıyla kendiniz yönetmelisiniz.
* **Dry-Run (Simülasyon) Desteği**: `sur` çalıştırılırken `--dry-run` bayrağı verilmişse, Lua VM içindeki `run()` ve `write_file()` fonksiyonları sistem üzerinde hiçbir değişiklik yapmaz (salt-okunur çalışır).
  - `run()` komutu çalıştırılmış gibi simüle edilir, loglara yazılır ve çıkış kodu `0` döner.
  - `write_file()` simüle edilir, dosyaya yazma yapmaz ve `nil` hata nesnesi döner.
* **Geri Alma (Rollback)**: `rollback(backup_path)` fonksiyonu çağrılmadan hemen önce, `sur` Go motoru `backup_files` ile alınan yedeği otomatik olarak eski konumuna geri yazar. Bu fonksiyon içinde size sadece servisi reload/restart etmek veya ek temizlik yapmak kalır. Eğer task başarısız olduğunda sistemde oluşturulan yeni bir dosyanın silinmesi gerekiyorsa (yedeklenecek bir dosya yoksa), `rollback` fonksiyonu içinde `run("rm -f /dosya/yolu")` şeklinde kendiniz silmelisiniz.

---

## Örnek Lua Task Dosyası

Aşağıdaki örnekte, root SSH girişini kapatan ve bir hata durumunda rollback yapabilen dinamik bir Lua task'ının yapısı gösterilmiştir:

```lua
id = "disable_root_ssh_lua"
name = "Disable Root SSH Login (Lua)"
description = "Updates sshd_config to disable PermitRootLogin"
rollback_possible = true
backup_files = { "/etc/ssh/sshd_config" }
risk_level = "medium"
distros = { "debian", "ubuntu" }

function pre_check()
    local content, err = read_file("/etc/ssh/sshd_config")
    if err ~= nil then
        log("sshd_config okunamadi: " .. err)
        return false, 1
    end
    -- Root login aciksa task gosterilsin
    if string.find(content, "\nPermitRootLogin yes") then
        return true, 0
    end
    return false, 0
end

function exec()
    log("Disabling root SSH login...")
    local content, err = read_file("/etc/ssh/sshd_config")
    if err ~= nil then return err end

    local new_content = string.gsub(content, "PermitRootLogin yes", "PermitRootLogin no")
    local w_err = write_file("/etc/ssh/sshd_config", new_content)
    if w_err ~= nil then return w_err end

    local out, code = run("systemctl restart sshd")
    if code ~= 0 then return "SSH restart basarisiz: " .. out end
    return nil
end

function post_check()
    local content, err = read_file("/etc/ssh/sshd_config")
    if err ~= nil then return err end
    if string.find(content, "\nPermitRootLogin yes") then
        return "Root login hala aktif!"
    end
    return nil
end

function rollback(backup_path)
    -- Go runner yedek dosyayi otomatik geri yukler.
    -- Bize sadece SSH servisini yeniden baslatmak kalir:
    run("systemctl restart sshd")
    return nil
end
```

