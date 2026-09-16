[English](/README.md) | [فارسی](/README.fa_IR.md) | [العربية](/README.ar_EG.md) | [中文](/README.zh_CN.md) | [Español](/README.es_ES.md) | [Русский](/README.ru_RU.md) | [Türkçe](/README.tr_TR.md)

# 3x-ui için Terraform Provider

> [3x-ui](https://github.com/MHSanaei/3x-ui) inbound'larını, istemcilerini, panel ayarlarını ve Xray yapılandırmasını kod olarak yönetin — panele tıklamadan VPN/proxy filonuzu yedekleyin, taşıyın ve ölçeklendirin.

[![CI](https://github.com/batonogov/terraform-provider-threexui/actions/workflows/ci.yml/badge.svg)](https://github.com/batonogov/terraform-provider-threexui/actions/workflows/ci.yml)
[![Terraform Registry](https://img.shields.io/badge/terraform-registry-blueviolet)](https://registry.terraform.io/providers/batonogov/threexui/latest)
[![Latest Release](https://img.shields.io/github/v/release/batonogov/terraform-provider-threexui?sort=semver)](https://github.com/batonogov/terraform-provider-threexui/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/batonogov/terraform-provider-threexui)](https://goreportcard.com/report/github.com/batonogov/terraform-provider-threexui)
[![Go Version](https://img.shields.io/github/go-mod/go-version/batonogov/terraform-provider-threexui)](go.mod)
[![Last Commit](https://img.shields.io/github/last-commit/batonogov/terraform-provider-threexui)](https://github.com/batonogov/terraform-provider-threexui/commits/main)
[![Codecov](https://codecov.io/gh/batonogov/terraform-provider-threexui/branch/main/graph/badge.svg)](https://codecov.io/gh/batonogov/terraform-provider-threexui)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

## Neden kullanmalısınız

Üretimde 3x-ui çalıştırmak, düzinelerce inbound, yüzlerce istemci ve kolayca bozulabilecek bir Xray yapılandırması anlamına gelir. Bu provider ile şunları yapabilirsiniz:

- **Yapılandırmayı kod olarak yönetin** — inbound listeniz git'te saklanır, her değişiklik gözden geçirilir ve sürümlenir.
- **Sunucular arasında güvenle taşıyın** — kimlikleri ve sırları korumak için panel veritabanını geri yükleyin, ardından Terraform ile doğrulayın.
- **Terraform state'i yedekleyin** — `terraform state pull` yalnızca Terraform'un yönettiği nesneleri dışa aktarır; felaket kurtarma için panel veritabanını da yedekleyin.
- **Onboarding'i ölçeklendirin** — 100 panel tıklaması yerine tek bir PR ile 100 istemci ekleyin.
- **Üretime göndermeden önce planlayın** — `terraform plan`, herhangi bir şey gönderilmeden önce nelerin değişeceğini tam olarak gösterir.

## Provider olmadan vs provider ile

| Görev | Panel UI | Bu provider |
| --- | --- | --- |
| 50 istemci eklemek | 50 form, her biri ~30 saniye | bir `for_each`, bir `apply` |
| Yeni sunucuya taşımak | manuel yeniden giriş | panel veritabanını geri yükleyip `terraform plan` ile doğrulamak |
| Bugün kimin erişimi olduğunu denetlemek | istemci listesini kaydırmak | bir `.tf` dosyasında `git log` |
| Yanlış yapılandırmayı geri almak | JSON yedekten geri yükleme | `git revert` + `terraform apply` |
| Staging ↔ Production senkronizasyonu | JSON dışa/içe aktarma, çakışmaları çözme | paylaşılan modül + ortam değişkenleri |
| 10 sunucuda Reality anahtarlarını döndürmek | 10 panel açın, her birine tıklayın | bir değişken değişikliği, bir apply |

## Hızlı Başlangıç

```hcl
terraform {
  required_providers {
    threexui = {
      source = "batonogov/threexui"
    }
  }
}

provider "threexui" {
  endpoint = "http://localhost:2053"
  username = "admin"
  password = "admin"
}

resource "threexui_inbound" "vless" {
  remark   = "VLESS Reality"
  port     = 443
  protocol = "vless"

  vless_settings {
    decryption = "none"
  }

  stream_settings {
    network  = "tcp"
    security = "reality"
    reality_settings {
      target       = "www.amazon.com:443"
      server_names = ["www.amazon.com"]
    }
  }

  sniffing {
    enabled       = true
    dest_override = ["http", "tls", "quic", "fakedns"]
  }
}

resource "threexui_inbound_client" "client_a" {
  inbound_id = threexui_inbound.vless.id
  email      = "client-a@example.com"
  enable     = true
  flow       = "xtls-rprx-vision"
}
```

> **OpenTofu kullanıcıları:** tam kayıt adresini kullanın:
>
> ```hcl
> source = "registry.terraform.io/batonogov/threexui"
> ```
>
> Provider, OpenTofu Registry'de yayınlanmamaktadır ([neden](https://github.com/opentofu/registry/issues/3632)).

## Uyumluluk

**Destek politikası:** provider resmi olarak desteklenen tüm 3x-ui minör hatlarındaki her yayımlanan yamayı destekler — aşağıdaki uyumluluk tablosuna bakın. Acceptance test matrisi her sürümü `main`'e yapılan her push'ta ve her pull request'te sınar.

| 3x-ui sürümü | Durum |
| --- | --- |
| v3.8.5 | Test edildi |
| v3.8.0 | Test edildi |
| v3.7.0 | Test edildi |
| v3.6.0 | Test edildi |
| v3.5.0 | Test edildi |
| v3.4.2 | Test edildi |
| v3.4.1 | Test edildi |
| v3.4.0 | Test edildi |
| v3.3.1 | Test edildi |
| v3.3.0 | Test edildi |

Daha yeni protokol özellikleri `requireMinVersion` ile korunur ve eski sürümlerde otomatik olarak atlanır, bu nedenle provider sürüm başına ayrı dallar olmadan tüm sürümlerde sorunsuz çalışır.

## Örnekler

| Örnek | Açıklama |
| --- | --- |
| [Provider ile ortam yapılandırması](examples/provider-env-config/) | Desteklenen `THREEXUI_*` ortam değişkenleriyle provider yapılandırması |
| [Panel kullanıcısı](examples/panel-user/) | Panel yöneticisi kimlik bilgilerini döndürme |
| [Panel e-postası](examples/panel-email/) | SMTP bildirimlerini yapılandırma (v3.4.0+) |
| [Trojan inbound](examples/trojan-inbound/) | WebSocket taşıma ile Trojan protokolü |
| [Shadowsocks inbound](examples/shadowsocks-inbound/) | AEAD şifreleme ile Shadowsocks |
| [İstemcili inbound](examples/inbound-with-client/) | Tam iş akışı: inbound + birden fazla istemci |
| [Küme düğümü](examples/node/) | Uzak bir 3x-ui panelini küme düğümü olarak kaydetme |
| [Host grubu](examples/host-group/) | Toplu host yönlendirmesini yönetme (v3.5.0+) |
| [Xray Observatory](examples/observatory/) | Outbound gecikme problarını yapılandırma (v3.4.2+) |
| [Xray sürümü](examples/xray-version/) | Kurulu Xray core sürümünü sabitleme |
| [Çok sunuculu filo](examples/multi-server/) | Yeniden kullanılabilir modül + `for_each` ile birden fazla 3x-ui barındırmasını yönetme |
| [Mevcut kaynakları içe aktarma](examples/import-existing/) | Mevcut 3x-ui kaynaklarını Terraform state'ine aktarma |

## Kılavuzlar

Yaygın operasyonel senaryolar için repo içindeki adım adım kılavuzlar:

- [Kod olarak yedekleme](docs/guides/backup-as-code.md) — gözden geçirilmiş Terraform yapılandırmasını state ve panel veritabanı yedekleriyle birleştirin.
- [3x-ui'ı sunucular arasında taşıma](docs/guides/server-migration.md) — panel veritabanını yeni VPS'e geri yükleyip Terraform ile doğrulayın.
- [Birçok istemciyi aynı anda ekleme](docs/guides/bulk-clients.md) — `for_each` desenleri ve CSV ile onboarding.

## Belgeler

Tam belgeler [Terraform Registry](https://registry.terraform.io/providers/batonogov/threexui/latest/docs)'de mevcuttur.

### Kaynaklar

| Kaynak | Açıklama |
| --- | --- |
| `threexui_inbound` | Inbound proxy (vless, vmess, trojan, shadowsocks, http, mixed, wireguard, amneziawg, tunnel, tun, hysteria, mtproto; TUN 3.2.7+, MTProto 3.3.0+, AmneziaWG 3.7.0+) |
| `threexui_inbound_client` | Bir inbound içindeki istemci |
| `threexui_node` | Küme düğümü / multi-node kaydı |
| `threexui_panel_general` | Genel panel ayarları |
| `threexui_panel_security` | Güvenlik ayarları (2FA) |
| `threexui_panel_user` | Yönetici kimlik bilgileri |
| `threexui_panel_telegram` | Telegram bot entegrasyonu |
| `threexui_panel_email` | SMTP/email bildirimleri (v3.4.0+) |
| `threexui_panel_discord` | Discord bot bildirimleri (v3.8.0+) |
| `threexui_host_group` | Host grubu yönlendirmesi (inbound başına birden fazla host) |
| `threexui_panel_subscription` | Abonelik hizmeti ayarları |
| `threexui_xray_basics` | Temel Xray yapılandırması (log, policy, api, stats) |
| `threexui_xray_dns` | DNS sunucuları ve hosts |
| `threexui_xray_routing` | Yönlendirme kuralları |
| `threexui_xray_balancers` | Yük dengeleyiciler |
| `threexui_xray_reverse` | Ters proxy (bridges, portals) |
| `threexui_xray_outbounds` | Outbound bağlantılar |
| `threexui_xray_observatory` | Xray Observatory / BurstObservatory yapılandırması |
| `threexui_xray_version` | Yüklü Xray çekirdek sürümü |

### Veri Kaynakları

| Veri Kaynağı | Açıklama |
| --- | --- |
| `threexui_inbounds` | Tüm inbound'ların listesi (JSON, hassas) |
| `threexui_nodes` | Küme düğüm ağacı / multi-node (JSON, hassas) |
| `threexui_server_status` | Sunucu durumu: CPU, bellek, disk, uptime (JSON) |
| `threexui_settings` | Tüm panel ayarları (JSON, hassas) |
| `threexui_xray_config` | Geçerli Xray şablonu (JSON, hassas) |
| `threexui_xray_versions` | Mevcut Xray sürümleri (dize listesi) |
| `threexui_online_clients` | Şu anda çevrimiçi olan istemci e-postaları |
| `threexui_client_traffics` | E-postaya göre istemci trafik istatistikleri |

## Güvenlik

Provider, panelin otomatik olarak verdiği sırları işler (Reality `privateKey`, WireGuard `secretKey`, istemci UUID'leri, Telegram bot tokenları, LDAP parolaları). Bu tür tüm alanlar `Sensitive` olarak işaretlenir ve düz metin olarak günlüğe kaydedilmez. Tam liste ve Terraform state'inizi koruma rehberi için bkz. [SECURITY.md](SECURITY.md).

## Geliştirme

### Gereksinimler

- Go (sürüm [`go.mod`](go.mod) dosyasında sabitlenmiştir)
- [Task](https://taskfile.dev/) — görev çalıştırıcı
- [golangci-lint](https://golangci-lint.run/welcome/install/) — linter
- [pre-commit](https://pre-commit.com/) — git kancaları çerçevesi
- Docker — yerel 3x-ui ortamı için

### Komutlar

```bash
task build        # Provider'ı derle
task fmt          # Kodu biçimlendir (gofmt)
task vet          # go vet çalıştır
task lint         # golangci-lint çalıştır
task pre-commit   # Tüm kontrolleri manuel olarak çalıştır (fmt, vet, lint, build)
task test:unit    # Birim testlerini çalıştır (Docker / Terraform gerekmez)
task test:acc     # Acceptance testlerini çalıştır (docker compose başlatır)
task test         # Birim + acceptance testlerini çalıştır
```

### Yerel ortam

```bash
# 3x-ui'ı localhost:2053 üzerinde başlat
docker compose up -d

# Giriş: admin / admin

# Durdur
docker compose down
```

## Katkıda bulunma

Yerel kurulum, test ve gönderim yönergeleri için bkz. [CONTRIBUTING.md](CONTRIBUTING.md). Hata raporları, özellik istekleri ve pull request'ler memnuniyetle karşılanır — üretimde hangi 3x-ui sürümlerini çalıştırdığınıza dair notlar da.

## Değişiklik günlüğü

Sürümler [Conventional Commits](https://www.conventionalcommits.org/) takip eder ve otomatik olarak yayımlanır. Tam sürüm geçmişi için bkz. [CHANGELOG.md](CHANGELOG.md).

## Lisans

[MIT](LICENSE)
