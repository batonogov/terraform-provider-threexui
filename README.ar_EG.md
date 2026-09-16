[English](/README.md) | [فارسی](/README.fa_IR.md) | [العربية](/README.ar_EG.md) | [中文](/README.zh_CN.md) | [Español](/README.es_ES.md) | [Русский](/README.ru_RU.md) | [Türkçe](/README.tr_TR.md)

<div dir="rtl">

# مزود Terraform لـ 3x-ui

> أدِر inbounds و العملاء و إعدادات اللوحة و إعدادات Xray في [3x-ui](https://github.com/MHSanaei/3x-ui) كأكواد — اعمل نسخ احتياطية، انقل أسطول الـ VPN/البروكسي بين السيرفرات و وسّعه دون الحاجة للضغط داخل اللوحة يدويًا.

[![CI](https://github.com/batonogov/terraform-provider-threexui/actions/workflows/ci.yml/badge.svg)](https://github.com/batonogov/terraform-provider-threexui/actions/workflows/ci.yml)
[![Terraform Registry](https://img.shields.io/badge/terraform-registry-blueviolet)](https://registry.terraform.io/providers/batonogov/threexui/latest)
[![Latest Release](https://img.shields.io/github/v/release/batonogov/terraform-provider-threexui?sort=semver)](https://github.com/batonogov/terraform-provider-threexui/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/batonogov/terraform-provider-threexui)](https://goreportcard.com/report/github.com/batonogov/terraform-provider-threexui)
[![Go Version](https://img.shields.io/github/go-mod/go-version/batonogov/terraform-provider-threexui)](go.mod)
[![Last Commit](https://img.shields.io/github/last-commit/batonogov/terraform-provider-threexui)](https://github.com/batonogov/terraform-provider-threexui/commits/main)
[![Codecov](https://codecov.io/gh/batonogov/terraform-provider-threexui/branch/main/graph/badge.svg)](https://codecov.io/gh/batonogov/terraform-provider-threexui)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

## ليه نستخدمه

تشغيل 3x-ui في الإنتاج معناه عشرات الـ inbounds، مئات العملاء، و إعدادات Xray من السهل إنها تتكسر. باستخدام هذا الـ provider:

- **الإعدادات بقت كود** — قائمة الـ inbounds موجودة في git، كل تعديل بيتم مراجعته و عمل version له.
- **النقل الآمن بين السيرفرات** — استرجع قاعدة بيانات اللوحة للحفاظ على المعرّفات والأسرار، ثم راجعها باستخدام Terraform.
- **نسخة احتياطية من Terraform state** — `terraform state pull` يصدّر فقط الموارد التي يديرها Terraform؛ احتفظ أيضًا بنسخة من قاعدة بيانات اللوحة للتعافي الكامل.
- **إضافة عدد كبير من العملاء** — ضيف 100 عميل في PR واحد بدل ما تضغط 100 مرة في اللوحة.
- **مراجعة قبل التطبيق** — `terraform plan` بيوريك بالظبط هيتغير إيه قبل ما تطبق أي حاجة.

## بدون الـ Provider مقابل مع الـ Provider

| المهمة | من واجهة اللوحة | بهذا الـ Provider |
| --- | --- | --- |
| إضافة 50 عميل | 50 فورم بـ ~30 ثانية لكل واحد | `for_each` واحد، `apply` واحد |
| الانتقال لسيرفر جديد | إعادة إدخال يدوي | استرجاع قاعدة بيانات اللوحة ثم المراجعة بـ `terraform plan` |
| مراجعة مين عنده صلاحيات | scroll في قائمة العملاء | `git log` على ملف `.tf` |
| التراجع عن تعديل غلط | استرجاع من نسخة JSON احتياطية | `git revert` + `terraform apply` |
| مزامنة staging و production | export/import يدوي للـ JSON | module مشترك + متغيرات لكل بيئة |
| تدوير مفاتيح Reality على 10 سيرفرات | فتح 10 لوحات و الضغط في كل واحدة | تعديل متغير واحد، apply واحد |

## بداية سريعة

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

<div dir="rtl">

> **لمستخدمي OpenTofu:** استخدموا العنوان الكامل للسجل:
>
> ```hcl
> source = "registry.terraform.io/batonogov/threexui"
> ```
>
> هذا الـprovider غير متوفر في OpenTofu Registry ([السبب](https://github.com/opentofu/registry/issues/3632)).

</div>

## التوافق

**سياسة الدعم:** الـ provider بيدعم رسميًا كل patch متنزّل في كل سطور 3x-ui الفرعية المدعومة — شوف جدول التوافقية تحت. matrix الـ acceptance بيختبر كل نسخة في كل push على `main` و في كل pull request.

| إصدار 3x-ui | الحالة |
| --- | --- |
| v3.8.5 | تم اختباره |
| v3.8.0 | تم اختباره |
| v3.7.0 | تم اختباره |
| v3.6.0 | تم اختباره |
| v3.5.0 | تم اختباره |
| v3.4.2 | تم اختباره |
| v3.4.1 | تم اختباره |
| v3.4.0 | تم اختباره |
| v3.3.1 | تم اختباره |
| v3.3.0 | تم اختباره |

مزايا البروتوكولات الجديدة محمية بـ `requireMinVersion` و بتتعدى تلقائيًا على الإصدارات الأقدم، يعني الـ provider بيشتغل على المصفوفة كاملة من غير ما تحتاج فروع منفصلة لكل إصدار.

## أمثلة

| المثال | الوصف |
| --- | --- |
| [إعداد Provider من env](examples/provider-env-config/) | إعداد الـ provider بمتغيرات `THREEXUI_*` المدعومة |
| [مستخدم اللوحة](examples/panel-user/) | تدوير بيانات اعتماد مسؤول اللوحة |
| [بريد اللوحة](examples/panel-email/) | إعداد إشعارات SMTP (v3.4.0+) |
| [Trojan inbound](examples/trojan-inbound/) | Trojan فوق WebSocket |
| [Shadowsocks inbound](examples/shadowsocks-inbound/) | Shadowsocks بتشفير AEAD |
| [Inbound + عملاء](examples/inbound-with-client/) | تدفق كامل: inbound + كذا عميل |
| [عقدة الكتلة](examples/node/) | تسجيل لوحة 3x-ui بعيدة كعقدة |
| [مجموعة hosts](examples/host-group/) | إدارة توجيه hosts بالجملة (v3.5.0+) |
| [Xray Observatory](examples/observatory/) | إعداد فحوصات زمن وصول outbounds (v3.4.2+) |
| [إصدار Xray](examples/xray-version/) | تثبيت إصدار Xray core المُثبّت |
| [أسطول متعدد السيرفرات](examples/multi-server/) | إدارة كذا 3x-ui من خلال module و `for_each` |
| [Import موارد موجودة](examples/import-existing/) | جلب موارد 3x-ui الموجودة فعلًا للـ state |

## الأدلة

أدلة جوّه الريبو لسيناريوهات تشغيلية شائعة:

- [Backup-as-code](docs/guides/backup-as-code.md) — اجمع بين إعدادات Terraform المراجَعة ونسخ state وقاعدة بيانات اللوحة.
- [نقل 3x-ui بين السيرفرات](docs/guides/server-migration.md) — استرجع قاعدة بيانات اللوحة على VPS جديد وراجعها باستخدام Terraform.
- [إضافة عملاء بالجملة](docs/guides/bulk-clients.md) — أنماط `for_each` و onboarding من ملف CSV.

## التوثيق

التوثيق الكامل موجود على [Terraform Registry](https://registry.terraform.io/providers/batonogov/threexui/latest/docs).

### Resources

| Resource | الوصف |
| --- | --- |
| `threexui_inbound` | inbound (vless, vmess, trojan, shadowsocks, http, mixed, wireguard, amneziawg, tunnel, tun, hysteria, mtproto؛ TUN من 3.2.7 وMTProto من 3.3.0 وAmneziaWG من 3.7.0) |
| `threexui_inbound_client` | عميل داخل inbound |
| `threexui_node` | عقدة الكتلة / تسجيل multi-node |
| `threexui_panel_general` | إعدادات اللوحة العامة |
| `threexui_panel_security` | إعدادات الأمان (2FA) |
| `threexui_panel_user` | بيانات اعتماد المسؤول |
| `threexui_panel_telegram` | تكامل بوت تليجرام |
| `threexui_panel_email` | إشعارات SMTP/email (v3.4.0+) |
| `threexui_host_group` | توجيه مجموعات hosts (أكثر من host لكل inbound) |
| `threexui_panel_subscription` | إعدادات خدمة الاشتراك |
| `threexui_xray_basics` | إعدادات Xray الأساسية (log, policy, api, stats) |
| `threexui_xray_dns` | سيرفرات و hosts الـ DNS |
| `threexui_xray_routing` | قواعد التوجيه |
| `threexui_xray_balancers` | الـ balancers |
| `threexui_xray_reverse` | reverse proxy (bridges, portals) |
| `threexui_xray_outbounds` | outbounds |
| `threexui_xray_observatory` | إعداد Xray Observatory / BurstObservatory |
| `threexui_xray_version` | إصدار Xray core المثبّت |

### Data Sources

| Data Source | الوصف |
| --- | --- |
| `threexui_inbounds` | كل الـ inbounds (JSON، حساس) |
| `threexui_nodes` | شجرة عقد الكتلة / multi-node (JSON، حساس) |
| `threexui_server_status` | حالة السيرفر: CPU، الذاكرة، القرص، uptime (JSON) |
| `threexui_settings` | كل إعدادات اللوحة (JSON، حساس) |
| `threexui_xray_config` | قالب Xray الحالي (JSON، حساس) |
| `threexui_xray_versions` | إصدارات Xray المتاحة (قائمة strings) |
| `threexui_online_clients` | إيميلات العملاء أونلاين دلوقتي |
| `threexui_client_traffics` | إحصائيات ترافيك العملاء بالإيميل |

## الأمان

الـ provider بيتعامل مع secrets بتطلعها اللوحة نفسها (`privateKey` لـ Reality، `secretKey` لـ WireGuard، UUIDs العملاء، token بوت تليجرام، باسوردات LDAP). كل هذه الحقول معلّمة `Sensitive` و عمرها ما بتتسجل في اللوج كـ plaintext. القائمة الكاملة و إرشادات حماية Terraform state موجودة في [SECURITY.md](SECURITY.md).

## التطوير

### المتطلبات

- Go (الإصدار مثبّت في [`go.mod`](go.mod))
- [Task](https://taskfile.dev/) — task runner
- [golangci-lint](https://golangci-lint.run/welcome/install/) — linter
- [pre-commit](https://pre-commit.com/) — git hooks
- Docker — لبيئة 3x-ui محليًا

### الأوامر

```bash
task build        # بناء الـ provider
task fmt          # تنسيق الكود (gofmt)
task vet          # go vet
task lint         # golangci-lint
task pre-commit   # كل الفحوصات (fmt, vet, lint, build)
task test:unit    # unit tests (بدون Docker أو Terraform)
task test:acc     # acceptance tests (بيشغل docker compose تلقائيًا)
task test         # unit + acceptance
```

### البيئة المحلية

```bash
# تشغيل 3x-ui على localhost:2053
docker compose up -d

# تسجيل الدخول: admin / admin

# إيقاف
docker compose down
```

## المساهمة

لإعداد البيئة المحلية، الاختبارات، و إرشادات الإرسال راجع [CONTRIBUTING.md](CONTRIBUTING.md). تقارير الأخطاء و طلبات الميزات و pull requests كلها موضع ترحيب — و كمان قول لنا أنهي إصدار من 3x-ui بتشغّل في الإنتاج.

## Changelog

الإصدارات بتتبع [Conventional Commits](https://www.conventionalcommits.org/) و بتتنشر تلقائيًا. التاريخ الكامل في [CHANGELOG.md](CHANGELOG.md).

## الترخيص

[MIT](LICENSE)

</div>
