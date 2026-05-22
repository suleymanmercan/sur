# Stack Yönetimi

`sur stack`, Docker Compose tabanlı geliştirme ve izleme (monitoring) ortamlarını terminal üzerinden kolayca yönetmenizi sağlayan etkileşimli bir araçtır. 

```bash
sudo sur stack
```

Uygulamanın `stack` komutu, TUI (Terminal User Interface) üzerinden `.env` dosyalarını yönetme, şablonları indirme ve Docker Compose servislerini yönetme işlerini basitleştirir. Böylece Compose dosyaları veya şifreler arasında kaybolmadan hızlıca kurulum yapabilirsiniz.

## Nasıl Çalışır?

`sur stack`, şablonları dinamik olarak GitHub üzerinden çeker. Şablonlar yerel olarak `/var/cache/sur/catalog/` altında önbelleğe (cache) alınır. TUI içerisindeki "Fetch / update catalog" seçeneği bu önbelleği yeniler.

Kurulan stack'ler hedef host üzerinde şu dizine yerleşir:
```text
/opt/sur/stacks/<stack-id>/
```

Böylece `sur`, config değerlerini `.env` dosyasına yazar, container'lar ise bu `.env` üzerinden ayağa kalkar.

## Resmi (Official) Stack'ler

Katalogda hazır gelen bazı resmi stack'ler (örneğin PostgreSQL, Redis) bulunur. Bunlar, varsayılan olarak güvenli konfigürasyonlarla başlatılır.

Örneğin, kurulum yaptığınızda PostgreSQL şifresi otomatik olarak güvenli bir şekilde oluşturulur ve TUI'de asla düz metin olarak (açıkça) gösterilmez.

## Kullanıcı Tanımlı (Custom) Stack'ler

Kendi özel stack'lerinizi tanımlamak isterseniz, geçerli bir stack dizinini `/etc/sur/stacks/<stack-id>/` yoluna yerleştirmeniz yeterlidir.

Örnek bir dizin yapısı:
```text
/etc/sur/stacks/my-app/
  stack.yaml
  compose.yml
  stack.lua     (opsiyonel)
```

Özel tanımladığınız stack'ler, TUI ekranında `[custom]` etiketi ile gösterilir ve resmi katalog güncellemelerinden etkilenmez (üzerine yazılmaz).

## Yaşam Döngüsü (Lifecycle) İşlemleri

TUI içerisinden kurulu her bir stack için aşağıdaki işlemleri yapabilirsiniz:

| Aksiyon | Açıklama |
| --- | --- |
| **Status** | Konteynerlerin anlık durumunu (running, exited vs.) gösterir. |
| **Logs** | Compose loglarının son satırlarını ekrana basar. |
| **Edit config** | `.env` değerlerini TUI formu üzerinden güncelleyip servisi yeniler. |
| **Restart** | `docker compose restart` komutunu işletir. |
| **Backup** | `data/` ve `secrets/` dizinlerini `backups/<timestamp>` klasörüne kopyalar. |
| **Update** | En güncel imajları (image) çeker ve konteynerleri yeniden başlatır. |
| **Stop (down)** | `docker compose down` komutunu işletir (ancak `data` silinmez). |

> [!TIP]
> Güvenlik gereği, `docker compose down -v` gibi kalıcı veri silen komutlar normal akışa dahil edilmemiştir. `data/`, `secrets/` ve `backups/` klasörleri otomatik olarak ASLA silinmez.

## stack.yaml Formatı

Her stack'in bir `stack.yaml` tanımı bulunur. Bu tanım, TUI'ye hangi form alanlarının gösterileceğini söyler:

```yaml
id: mystack
name: My Stack
description: Özel stack uygulaması
risk_level: low

config:
  - id: port
    label: Port
    type: number
    default: "8080"

  - id: password
    label: Password
    type: secret
    generate: true
```

* `config` alanında tanımlı değişken türleri: `text`, `number`, `select`, `bool`, `secret`.
* `secret` olarak tanımlanan alanlar otomatik olarak generate edildiğinde `secrets/<id>.txt` dosyasına `0600` yetkisiyle yazılır ve `.env` içerisinden referans gösterilir.
