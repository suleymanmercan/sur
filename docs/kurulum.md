# Kurulum

`sur`, tek bir binary dosyasından oluşur ve hiçbir bağımlılığa (dependency) ihtiyaç duymaz. İşlemci mimarinizi (amd64 veya arm64) otomatik algılar.

## Hızlı Kurulum (Tavsiye Edilen)

Kurulum betiğini kullanarak hızlıca kurabilirsiniz:

```bash
curl -fsSL https://raw.githubusercontent.com/suleymanmercan/sur/main/install.sh | sudo bash
```

> **Not:** Sisteminizde `curl` ve yetkili bir kullanıcı (`sudo`) olmalıdır. Desteklenen sistemler: Linux amd64, arm64, Debian, Ubuntu, Rocky, AlmaLinux, Fedora.

## Kaldırma İşlemi

Sadece binary'i silmek ancak ayarları ve history'i bırakmak için:

```bash
curl -fsSL https://raw.githubusercontent.com/suleymanmercan/sur/main/install.sh | sudo bash -s -- --uninstall
```

Her şeyi (konfigürasyonlar ve veritabanı dahil) tamamen silmek (purge) için:

```bash
curl -fsSL https://raw.githubusercontent.com/suleymanmercan/sur/main/install.sh | sudo bash -s -- --uninstall --purge
```
