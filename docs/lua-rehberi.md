# Lua Task Yazma Rehberi

Statik YAML komut dizilerinin yetersiz kaldığı durumlarda — dosya içeriğini okuyup regex ile düzenleme, karmaşık koşullu mantık, çoklu dosya yönetimi — `sur`, **Lua 5.1** ile tam task desteği sunar.

Bir `.lua` uzantılı dosya `tasks/` veya `install_tasks/` dizinine atıldığında otomatik olarak yüklenir.

---

## Dosya Yapısı

Lua task dosyası iki bölümden oluşur:

1. **Global değişkenler (metadata)** — dosyanın en başında tanımlanır
2. **Lifecycle fonksiyonları** — `pre_check`, `exec`, `post_check`, `rollback`

```lua
-- 1. Metadata
id          = "task_id"
name        = "Kullanıcıya görünen ad"
description = "TUI'da seçili iken altta görünen açıklama"
risk_level  = "low"          -- "low" | "medium" | "high"
rollback_possible = true
distros     = {}             -- boş → tüm sistemler
backup_files = { "/etc/örnek.conf" }

-- 2. Lifecycle fonksiyonları
function pre_check() ... end
function exec() ... end
function post_check() ... end
function rollback(backup_path) ... end
```

---

## Metadata Referansı

| Alan | Tip | Açıklama |
|---|---|---|
| `id` | string | Zorunlu. Benzersiz, kalıcı kimlik. `--only` ve history'de kullanılır. |
| `name` | string | TUI listesinde görünen başlık. |
| `description` | string | TUI'da seçili satırın altında kısa açıklama. |
| `risk_level` | string | `"low"`, `"medium"` veya `"high"`. |
| `rollback_possible` | boolean | Rollback desteği var mı? |
| `distros` | table | Desteklenen dağıtımlar. Boş → hepsi. |
| `backup_files` | table | exec'ten önce yedeklenecek dosya yolları. İlk var olan yedeklenir. |

---

## Lifecycle Fonksiyonları

### `pre_check()` → `(needs_run: boolean, exit_code: number)`

"Bu task'a ihtiyaç var mı?" sorusunu yanıtlar. **Sistem sağlıklı mı kontrolü değildir.**

- `true, 0` → task TUI'da gösterilir, çalıştırılabilir
- `false, 0` → "zaten yapılmış", task atlanır

```lua
function pre_check()
    local content, err = read_file("/etc/ssh/sshd_config")
    if err ~= nil then
        log("Dosya okunamadı: " .. err)
        return false, 1
    end
    -- Ayar yoksa task çalıştırılsın
    if string.find(content, "ClientAliveInterval 300") then
        return false, 0  -- zaten yapılmış
    end
    return true, 0  -- task gerekli
end
```

> [!TIP]
> `pre_check` başarısız olan task'lar `--yes` ve `--all` modlarında da atlanır. Bu bir güvence katmanı — idempotent davranış sağlar.

---

### `exec()` → `(error: string | nil)`

Değişikliği uygulayan asıl fonksiyon. Başarıda `nil`, hata durumunda hata mesajı string'i döner.

**Yeni:** `log()` ile yazdırılan mesajlar artık canlı TUI akışında görünür.

```lua
function exec()
    log("SSH idle timeout ayarlanıyor...")

    local content, err = read_file("/etc/ssh/sshd_config")
    if err ~= nil then return err end

    -- Mevcut satırı değiştir, yoksa ekle
    local found = false
    local lines = {}
    for line in string.gmatch(content, "[^\n]+") do
        if string.find(line, "^[#%s]*ClientAliveInterval") then
            table.insert(lines, "ClientAliveInterval 300")
            found = true
        else
            table.insert(lines, line)
        end
    end
    if not found then
        table.insert(lines, "ClientAliveInterval 300")
    end

    local w_err = write_file("/etc/ssh/sshd_config", table.concat(lines, "\n") .. "\n")
    if w_err ~= nil then return w_err end

    local out, code = run("sshd -t")
    if code ~= 0 then return "SSH config geçersiz: " .. out end

    local _, rc = run("systemctl reload ssh || systemctl reload sshd")
    if rc ~= 0 then return "SSH reload başarısız" end

    return nil
end
```

---

### `post_check()` → `(error: string | nil)`

`exec()` başarıyla bittikten sonra değişikliği doğrular. `nil` → başarı, string → hata.

```lua
function post_check()
    local out, code = run("sshd -T 2>/dev/null")
    if code ~= 0 then
        return "sshd -T çalıştırılamadı"
    end
    if not string.find(out, "clientaliveinterval 300") then
        return "ClientAliveInterval 300 olmaya ayarlanmamış"
    end
    return nil
end
```

---

### `rollback(backup_path)` → `(error: string | nil)`

Hata durumunda veya `sur rollback <session-id>` ile çağrıldığında tetiklenir.

**Önemli:** Go runner, bu fonksiyonu çağırmadan önce `backup_files` ile kaydedilen yedek dosyayı otomatik olarak yerine geri yükler. Rollback fonksiyonuna genellikle yalnızca servisi yeniden başlatmak ya da ek temizlik yapmak kalır.

```lua
function rollback(backup_path)
    log("SSH yapılandırması eski haline döndürülüyor...")
    -- backup_path'i kullanmana gerek yok: Go runner dosyayı zaten geri yükledi
    local _, code = run("systemctl reload ssh || systemctl reload sshd || service ssh reload")
    if code ~= 0 then
        return "SSH reload başarısız"
    end
    return nil
end
```

Yedek dosya yoksa (örn. yeni bir dosya oluşturduysan) silme işlemini kendin yap:

```lua
function rollback(backup_path)
    run("rm -f /etc/yeni_dosya.conf")
    return nil
end
```

---

## Yardımcı Fonksiyonlar (Lua VM API)

Tüm Lua task'larında aşağıdaki Go-bound fonksiyonlar kullanılabilir:

### `run(command)` → `(output: string, exit_code: number)`

Shell komutu çalıştırır. Çıktıyı ve exit kodunu döner.

```lua
local out, code = run("systemctl is-active fail2ban")
if code ~= 0 then
    log("fail2ban aktif değil: " .. out)
end
```

> [!IMPORTANT]
> `--dry-run` modunda `run()` komutu **çalıştırmaz**. Log'a yazar, exit kodu `0` döner. Task'ı dry-run uyumlu yazmak için özel bir şey yapman gerekmez.

---

### `log(message)` → `nil`

Kullanıcıya ilerleme mesajı gösterir. Çıktı artık TUI'da canlı akar.

```lua
log("Paket indiriliyor, bu biraz sürebilir...")
```

---

### `read_file(path)` → `(content: string, error: string | nil)`

Dosyayı okur. Hata durumunda `content` boş, `error` dolu döner.

```lua
local content, err = read_file("/etc/ssh/sshd_config")
if err ~= nil then
    return err  -- exec() içindeysen hatayı döndür
end
```

---

### `write_file(path, content)` → `(error: string | nil)`

Dosyayı oluşturur veya üzerine yazar. `nil` → başarı.

```lua
local err = write_file("/etc/myapp/config.conf", new_content)
if err ~= nil then return err end
```

> [!IMPORTANT]
> `--dry-run` modunda `write_file()` dosyaya **yazmaz**. `nil` döner ve log'a yazar. Gerçek dosya dokunulmaz.

---

### `file_exists(path)` → `(exists: boolean)`

Dosyanın var olup olmadığını kontrol eder.

```lua
if not file_exists("/etc/fail2ban/jail.local") then
    log("jail.local yok, varsayılan kullanılıyor")
end
```

---

## Yaygın Desenler

### Dosya satırı değiştirme (replace or append)

```lua
local function set_sshd_option(content, key, value)
    local pattern = "^[#%s]*" .. key .. "%s+"
    local new_line = key .. " " .. value
    local found = false
    local lines = {}

    for line in string.gmatch(content, "[^\n]+") do
        if string.find(line, pattern) then
            table.insert(lines, new_line)
            found = true
        else
            table.insert(lines, line)
        end
    end

    if not found then
        table.insert(lines, new_line)
    end

    return table.concat(lines, "\n") .. "\n"
end
```

### Birden fazla anahtar kelime kontrol etme

```lua
function pre_check()
    local out, code = run("sshd -T 2>/dev/null")
    if code ~= 0 then return true, 0 end  -- kontrol yapılamazsa çalıştır

    local interval_ok = string.find(out, "clientaliveinterval 300")
    local count_ok    = string.find(out, "clientalivecountmax 2")

    if interval_ok and count_ok then
        return false, 0  -- zaten doğru ayarlanmış
    end
    return true, 0
end
```

### Servis durumuna göre farklı davranma

```lua
local out, code = run("systemctl is-active myservice 2>/dev/null")
if code == 0 then
    run("systemctl reload myservice")
else
    run("systemctl enable --now myservice")
end
```

---

## Önemli Kısıtlamalar ve Uyarılar

### Tek dosya yedekleme sınırı

`backup_files` listesindeki dosyalardan **yalnızca ilk var olanı** yedeklenir. Birden fazla dosya yedeklemen gerekiyorsa bunu `exec()` ve `rollback()` içinde kendin yönet:

```lua
function exec()
    -- Manuel yedek al
    run("cp /etc/a.conf /tmp/a.conf.bak")
    run("cp /etc/b.conf /tmp/b.conf.bak")
    -- ... değişiklikler ...
    return nil
end

function rollback(backup_path)
    run("cp /tmp/a.conf.bak /etc/a.conf")
    run("cp /tmp/b.conf.bak /etc/b.conf")
    return nil
end
```

### Yedeksiz rollback

Eğer `backup_files` boş ve `rollback_possible: true` ise rollback fonksiyonun `backup_path` parametresi boş string gelir. Dosya restore etmeye çalışma.

### Lua 5.1 sınırları

`sur` Lua 5.1 çalıştırır (gopher-lua). Standart kütüphane mevcuttur (`string`, `table`, `math`, `io`) ama `os.execute`, `io.popen` gibi sistem çağrıları doğrudan desteklenmez — bunun yerine `run()` kullan.

---

## Tam Örnek: `ssh_idle_timeout.lua`

```lua
id = "ssh_idle_timeout"
name = "Configure SSH Idle Timeout"
description = "Sets ClientAliveInterval 300 and ClientAliveCountMax 2 in sshd_config."
rollback_possible = true
backup_files = { "/etc/ssh/sshd_config" }
risk_level = "low"
distros = {}  -- tüm dağıtımlar

function pre_check()
    local out, code = run("sshd -T 2>/dev/null")
    if code == 0 then
        local interval_ok = string.find(out, "clientaliveinterval 300")
        local count_ok    = string.find(out, "clientalivecountmax 2")
        if interval_ok and count_ok then
            return false, 0  -- zaten ayarlı
        end
    end
    return true, 0
end

function exec()
    log("SSH idle timeout ayarlanıyor...")
    local content, err = read_file("/etc/ssh/sshd_config")
    if err ~= nil then return err end

    local lines = {}
    local interval_set, count_set = false, false

    for line in string.gmatch(content, "[^\n]+") do
        if string.find(line, "^[#%s]*[Cc]lient[Aa]live[Ii]nterval%s+") then
            table.insert(lines, "ClientAliveInterval 300")
            interval_set = true
        elseif string.find(line, "^[#%s]*[Cc]lient[Aa]live[Cc]ount[Mm]ax%s+") then
            table.insert(lines, "ClientAliveCountMax 2")
            count_set = true
        else
            table.insert(lines, line)
        end
    end

    if not interval_set then table.insert(lines, "ClientAliveInterval 300") end
    if not count_set    then table.insert(lines, "ClientAliveCountMax 2") end

    local w_err = write_file("/etc/ssh/sshd_config", table.concat(lines, "\n") .. "\n")
    if w_err ~= nil then return w_err end

    log("SSH config doğrulanıyor...")
    local val_out, val_code = run("sshd -t")
    if val_code ~= 0 then return "Geçersiz SSH config: " .. val_out end

    log("SSH yeniden yükleniyor...")
    local _, rc = run("systemctl reload ssh || systemctl reload sshd || service ssh reload")
    if rc ~= 0 then return "SSH reload başarısız" end

    return nil
end

function post_check()
    local out, code = run("sshd -T 2>/dev/null")
    if code ~= 0 then return "sshd -T çalıştırılamadı" end
    if not string.find(out, "clientaliveinterval 300") then
        return "ClientAliveInterval 300 değil"
    end
    if not string.find(out, "clientalivecountmax 2") then
        return "ClientAliveCountMax 2 değil"
    end
    return nil
end

function rollback(backup_path)
    log("Orijinal sshd_config geri yüklendi, servis yeniden başlatılıyor...")
    run("systemctl reload ssh || systemctl reload sshd || service ssh reload")
    return nil
end
```

---

## YAML mı Lua mı?

| Durum | Tercih |
|---|---|
| Sadece shell komutları çalıştırma | ✅ YAML |
| Dosya içeriği okuma/yazma/regex | ✅ Lua |
| Karmaşık koşullu mantık | ✅ Lua |
| Birden fazla çıktı değeri kontrol etme | ✅ Lua |
| Basit paket kurulumu + servis aktifleştirme | ✅ YAML |
| Config satırı bulup yerinde değiştirme | ✅ Lua |
