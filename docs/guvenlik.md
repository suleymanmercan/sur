# Güvenlik Notları

`sur`, root yetkisiyle sistem değişikliği yapabilir. Bu yüzden güvenlik modeli "önce göster, sonra kullanıcı seçsin" yaklaşımına dayanır.

## Güvenlik Modeli

| Katman | Ne sağlar? |
| --- | --- |
| `pre_check` | Zaten uygulanmış task'ları gizler ve tekrar çalışmayı azaltır. |
| `--dry-run` | Dosya yazmadan ve komutları uygulamadan planı gösterir. |
| TUI seçimi | Operatörün hangi task'ın çalışacağını açıkça seçmesini sağlar. |
| `backup_files` | Desteklenen config dosyalarını değişiklik öncesi saklar. |
| SQLite history | Session, task sonucu ve rollback verisini yerel olarak tutar. |
| `rollback_possible` | Hangi task'ın geri alınabilir olduğunu görünür yapar. |

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

> [!IMPORTANT]
> SSH veya firewall değiştiren task'larda mevcut bağlantıyı kapatma. Ayrı bir SSH oturumu açık kalsın; mümkünse cloud provider console erişimin de hazır olsun.

## Dikkat Edilecek Task'lar

| Task | Risk |
| --- | --- |
| `enable_ufw` | Yanlış firewall kuralı uzak bağlantıyı kesebilir. |
| `ssh_password_auth_off` | SSH key yoksa giriş kaybedilebilir. |
| `disable_root_ssh` | Root ile girişe bağımlı sistemlerde erişim akışı değişir. |
| `install_docker` | Paket repo ve service değişiklikleri yapar. |
| `install_caddy` | Web server service ekler ve port kullanımını etkileyebilir. |

## Stack Güvenliği

`sur stack` komutu ile yönetilen ortamlar için şu güvenlik prensipleri geçerlidir:
- **Varsayılan Local Bağlantı:** Servislerin host bağlantı noktası (`bind_host`) varsayılan olarak `127.0.0.1`'dir, böylece veri tabanları dışarıdan (public) erişime kapalı kalır. Açmak isterseniz TUI üzerinden `0.0.0.0` seçilmesi gerekir.
- **Otomatik ve Güvenli Sırlar:** Şifreler (örn. `POSTGRES_PASSWORD`) boş bırakıldığında rastgele güvenli şekilde oluşturulur. Bu değerler `/opt/sur/stacks/<id>/secrets/` altına `0600` yetkisiyle kaydedilir; loglarda veya TUI ekranında asla düz metin olarak basılmaz.
- **Sabit Sürümler:** Docker imajlarında asla `latest` kullanılmaz, majör sürümler (örn. `postgres:16`) sabitlenir.
- **Silme Koruması:** Normal kullanımda `docker compose down -v` asla çalıştırılmaz, kalıcı veri dizinleri (`data/`, `secrets/`) her zaman güvendedir.

## Rollback Sınırları

Rollback dosya değişikliklerinde güçlüdür; paket kurulumu, firewall state'i, servis enable/disable davranışı ve dış paket repository ekleme gibi işlemler her dağıtımda tam tersine çevrilemeyebilir.

| Durum | Beklenti |
| --- | --- |
| SSH config satırı değişti | Yedek dosya geri yazılabilir |
| `/etc/fstab` güncellendi | Yedek dosya geri yazılabilir |
| Paket kuruldu | Manuel temizlik gerekebilir |
| Firewall aktif edildi | Kural ve erişim durumu manuel doğrulanmalı |
| Docker/Caddy service eklendi | Servis ve paket yönetimi ayrı kontrol edilmeli |

## Skor Ne Anlama Geliyor?

`sur check` skoru pratik bir sinyaldir. Compliance sertifikası değildir.

`ports.listening` gibi bazı uyarılar otomatik kapatılmaz. Çünkü hangi portun production için gerekli olduğunu CLI bilemez. Bu tip bulgular manuel review gerektirir.

## İlan Edilen Güvenli Kullanım Cümlesi

`sur`, sunucuyu tek komutla körlemesine değiştiren bir araç değildir. Önce audit ve dry-run üretir, sonra operatörün seçtiği task'ları yerel olarak çalıştırır. Bu yüzden güvenli kullanımın parçası olan karar noktaları dokümanda açık tutulur.
