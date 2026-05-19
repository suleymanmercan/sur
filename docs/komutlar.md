# Komutlar

`sur`, local host üzerinde çalışan tek binary bir CLI'dır. Root gerektiren işlemler için `sudo` ile çalıştırılır.

## Audit

```bash
sur check
```

Sunucudaki temel güvenlik durumunu kontrol eder:

- SSH root login
- SSH password auth
- SSH port
- firewall durumu
- fail2ban
- automatic updates
- listening sockets
- sudoers `NOPASSWD`

Derin Lynis kontrolü:

```bash
sur check --deep
```

Lynis yoksa kurup sonra çalıştırmak için:

```bash
sudo sur check --deep --install-lynis
```

## Hardening

Önce dry-run:

```bash
sudo sur harden --dry-run
```

Interactive TUI:

```bash
sudo sur harden
```

Sadece belirli task'lar:

```bash
sudo sur harden --only enable_ufw,install_fail2ban
```

TUI açmadan tüm uygulanabilir task'lar:

```bash
sudo sur harden --yes
```

Dışarıdan özel bir task dizini belirtme:

```bash
sudo sur harden --tasks /etc/sur/custom_tasks
```

## Install / Setup

Fresh server setup task'ları:

```bash
sudo sur install
```

Örnek:

```bash
sudo sur install --only configure_swap,install_docker,install_caddy
```

Swap boyutunu değiştirmek için:

```bash
sudo SUR_SWAP_SIZE=4G sur install --only configure_swap
```

## History ve Rollback

Geçmiş session'lar:

```bash
sur history
```

Rollback:

```bash
sudo sur rollback <session-id>
```

Rollback her task için garanti değildir. Config dosyası değiştiren task'lar genelde geri alınabilir; package install ve firewall gibi task'lar manuel toparlama gerektirebilir.
