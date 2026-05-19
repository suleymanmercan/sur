# Kurulum ve Güncelleme

`sur`, tek bir binary dosyasından oluşur. Kurulum script'i Linux mimarisini (`amd64` veya `arm64`) otomatik algılar, GitHub release asset'ini indirir ve checksum doğrulaması yapar.

## Hızlı Kurulum (Tavsiye Edilen)

```bash
curl -fsSL https://raw.githubusercontent.com/suleymanmercan/sur/main/install.sh | sudo bash
```

Kurulum sonrası:

```bash
sur check
```

## Güncelleme

Mevcut kurulumu son release'e güncellemek için:

```bash
curl -fsSL https://raw.githubusercontent.com/suleymanmercan/sur/main/install.sh | sudo bash -s -- --update
```

Bu işlem `/usr/local/bin/sur` binary'sini değiştirir. `/var/lib/sur/sur.db` state dosyasını silmez.

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
