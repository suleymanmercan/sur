# Kurulum ve Güncelleme

`sur`, tek bir binary dosyasından oluşur. Kurulum script'i işletim sistemi ve mimariyi algılar, GitHub Release arşivini indirir, `checksums.txt` üzerinden doğrular ve binary'yi `/usr/local/bin/sur` altına yerleştirir.

| Platform | Mimari | Release asset |
| --- | --- | --- |
| Linux | `amd64`, `arm64` | `sur_<version>_linux_<arch>.tar.gz` |
| macOS | `amd64`, `arm64` | `sur_<version>_darwin_<arch>.tar.gz` |

> [!NOTE]
> Ana hedef Linux/VPS kullanımıdır. macOS build'i lokal deneme ve geliştirme kolaylığı içindir; hardening task'ları Linux sistem dosyalarına göre tasarlanır.

## Hızlı Kurulum (Tavsiye Edilen)

```bash
curl -fsSL https://raw.githubusercontent.com/suleymanmercan/sur/main/install.sh | sudo bash
```

Kurulum sonrası:

```bash
sur --version
sur check
```

## İlk Sunucu Akışı

Yeni bir VPS üzerinde önce durumu gör, sonra TUI içinde seçerek ilerle:

```bash
sur check
sudo sur harden
sudo sur install
```

Emin değilsen uygulamadan önce preview alabilirsin:

```bash
sudo sur harden --dry-run
```

## Güncelleme

Mevcut kurulumu son release'e güncellemek için:

```bash
curl -fsSL https://raw.githubusercontent.com/suleymanmercan/sur/main/install.sh | sudo bash -s -- --update
```

Bu işlem yalnızca `/usr/local/bin/sur` binary'sini değiştirir. `/var/lib/sur/sur.db` state dosyasını ve eski session kayıtlarını silmez.

## Kaldırma İşlemi

Sadece binary'i silmek, history/state dosyalarını bırakmak için:

```bash
curl -fsSL https://raw.githubusercontent.com/suleymanmercan/sur/main/install.sh | sudo bash -s -- --uninstall
```

Binary ve state dosyalarını temizlemek için:

```bash
curl -fsSL https://raw.githubusercontent.com/suleymanmercan/sur/main/install.sh | sudo bash -s -- --uninstall --purge
```

## Kaynaktan Build

```bash
git clone https://github.com/suleymanmercan/sur.git
cd sur
make build
sudo make install
```

## Kurulum Sonrası Kontrol

| Kontrol | Komut |
| --- | --- |
| Binary PATH'te mi? | `command -v sur` |
| Sürüm doğru mu? | `sur --version` |
| Temel audit çalışıyor mu? | `sur check` |
| TUI açılıyor mu? | `sudo sur harden --dry-run` |
