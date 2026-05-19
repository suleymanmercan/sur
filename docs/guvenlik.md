# Güvenlik Notları

`sur`, root yetkisiyle sistem değişikliği yapabilir. Bu yüzden güvenlik modeli "önce göster, sonra kullanıcı seçsin" yaklaşımına dayanır.

## Önerilen Akış

1. Önce kontrol et:

```bash
sur check
```

2. Değişiklikleri dry-run ile gör:

```bash
sudo sur harden --dry-run
```

3. Remote VPS üzerinde SSH/firewall değiştiriyorsan açık bir SSH oturumu bırak.

4. Riskli task'ları tek tek uygula:

```bash
sudo sur harden --only disable_root_ssh
sudo sur harden --only ssh_password_auth_off
sudo sur harden --only enable_ufw
```

## Dikkat Edilecek Task'lar

| Task | Risk |
| --- | --- |
| `enable_ufw` | Yanlış firewall kuralı uzak bağlantıyı kesebilir. |
| `ssh_password_auth_off` | SSH key yoksa giriş kaybedilebilir. |
| `disable_root_ssh` | Root ile girişe bağımlı sistemlerde erişim akışı değişir. |
| `install_docker` | Paket repo ve service değişiklikleri yapar. |
| `install_caddy` | Web server service ekler ve port kullanımını etkileyebilir. |

## Skor Ne Anlama Geliyor?

`sur check` skoru pratik bir sinyaldir. Compliance sertifikası değildir.

`ports.listening` gibi bazı uyarılar otomatik kapatılmaz. Çünkü hangi portun production için gerekli olduğunu CLI bilemez. Bu tip bulgular manuel review gerektirir.
