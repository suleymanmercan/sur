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
