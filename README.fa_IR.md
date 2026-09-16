[English](/README.md) | [فارسی](/README.fa_IR.md) | [العربية](/README.ar_EG.md) | [中文](/README.zh_CN.md) | [Español](/README.es_ES.md) | [Русский](/README.ru_RU.md) | [Türkçe](/README.tr_TR.md)

<div dir="rtl">

# Terraform Provider برای 3x-ui

> اینباندها، کلاینت‌ها، تنظیمات پنل و پیکربندی Xray در [3x-ui](https://github.com/MHSanaei/3x-ui) را به‌صورت کد مدیریت کنید — بدون نیاز به کلیک در پنل، از ناوگان VPN/پروکسی خود بکاپ بگیرید، آن را مهاجرت دهید و مقیاس‌پذیر کنید.

[![CI](https://github.com/batonogov/terraform-provider-threexui/actions/workflows/ci.yml/badge.svg)](https://github.com/batonogov/terraform-provider-threexui/actions/workflows/ci.yml)
[![Terraform Registry](https://img.shields.io/badge/terraform-registry-blueviolet)](https://registry.terraform.io/providers/batonogov/threexui/latest)
[![Latest Release](https://img.shields.io/github/v/release/batonogov/terraform-provider-threexui?sort=semver)](https://github.com/batonogov/terraform-provider-threexui/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/batonogov/terraform-provider-threexui)](https://goreportcard.com/report/github.com/batonogov/terraform-provider-threexui)
[![Go Version](https://img.shields.io/github/go-mod/go-version/batonogov/terraform-provider-threexui)](go.mod)
[![Last Commit](https://img.shields.io/github/last-commit/batonogov/terraform-provider-threexui)](https://github.com/batonogov/terraform-provider-threexui/commits/main)
[![Codecov](https://codecov.io/gh/batonogov/terraform-provider-threexui/branch/main/graph/badge.svg)](https://codecov.io/gh/batonogov/terraform-provider-threexui)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

## چرا از این Provider استفاده کنیم

اجرای 3x-ui در محیط پروداکشن یعنی ده‌ها اینباند، صدها کلاینت، و پیکربندی Xray که به‌راحتی خراب می‌شود. با این provider می‌توانید:

- **پیکربندی به‌صورت کد** — لیست اینباندها در git ذخیره می‌شود، هر تغییر review و نسخه‌بندی می‌شود.
- **مهاجرت امن بین سرورها** — برای حفظ شناسه‌ها و رازها، پایگاه دادهٔ پنل را بازیابی و سپس با Terraform بررسی کنید.
- **پشتیبان‌گیری از Terraform state** — `terraform state pull` فقط منابع مدیریت‌شده با Terraform را خروجی می‌دهد؛ برای بازیابی کامل، از پایگاه دادهٔ پنل هم پشتیبان بگیرید.
- **اضافه‌کردن انبوه کلاینت‌ها** — ۱۰۰ کلاینت را در یک PR اضافه کنید، نه با ۱۰۰ بار کلیک در پنل.
- **پیش‌نمایش قبل از پروداکشن** — `terraform plan` دقیقاً نشان می‌دهد چه چیزی تغییر خواهد کرد.

## بدون provider در مقابل با provider

| کار | از طریق UI پنل | از طریق این provider |
| --- | --- | --- |
| اضافه‌کردن ۵۰ کلاینت | ۵۰ فرم، هرکدام ~۳۰ ثانیه | یک `for_each`، یک `apply` |
| مهاجرت به سرور جدید | ورود دستی همه‌چیز | بازیابی پایگاه دادهٔ پنل و بررسی با `terraform plan` |
| بررسی دسترسی کلاینت‌ها | اسکرول لیست کلاینت‌ها | `git log` روی فایل `.tf` |
| بازگرداندن یک تغییر اشتباه | بازیابی از بکاپ JSON | `git revert` + `terraform apply` |
| همگام‌سازی staging و production | export/import دستی JSON | ماژول مشترک + متغیر مخصوص هر محیط |
| چرخش کلیدهای Reality روی ۱۰ هاست | باز کردن ۱۰ پنل و کلیک در هر کدام | تغییر یک متغیر، یک apply |

## شروع سریع

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

> **برای کاربران OpenTofu:** آدرس کامل رجیستری را مشخص کنید:
>
> ```hcl
> source = "registry.terraform.io/batonogov/threexui"
> ```
>
> این provider در OpenTofu Registry منتشر نشده است ([دلیل](https://github.com/opentofu/registry/issues/3632)).

</div>

## سازگاری

**سیاست پشتیبانی:** این provider به‌طور رسمی از هر patch منتشرشده در تمام خطوط مینور پشتیبانی‌شدهٔ 3x-ui پشتیبانی می‌کند — جدول سازگاری زیر را ببینید. ماتریس acceptance هر نسخه را در هر push به `main` و هر pull request اجرا می‌کند.

| نسخهٔ 3x-ui | وضعیت |
| --- | --- |
| v3.8.5 | تست‌شده |
| v3.8.0 | تست‌شده |
| v3.7.0 | تست‌شده |
| v3.6.0 | تست‌شده |
| v3.5.0 | تست‌شده |
| v3.4.2 | تست‌شده |
| v3.4.1 | تست‌شده |
| v3.4.0 | تست‌شده |
| v3.3.1 | تست‌شده |
| v3.3.0 | تست‌شده |

ویژگی‌های مربوط به نسخه‌های جدید‌تر پروتکل با `requireMinVersion` محافظت می‌شوند و روی نسخه‌های قدیمی به‌طور خودکار skip می‌شوند، بنابراین provider روی کل ماتریس بدون نیاز به branchهای جداگانه برای هر نسخه کار می‌کند.

## مثال‌ها

| مثال | توضیح |
| --- | --- |
| [پیکربندی provider با env](examples/provider-env-config/) | پیکربندی با متغیرهای پشتیبانی‌شدهٔ `THREEXUI_*` |
| [کاربر پنل](examples/panel-user/) | چرخش اعتبارنامه‌های مدیر پنل |
| [ایمیل پنل](examples/panel-email/) | پیکربندی اعلان‌های SMTP (v3.4.0+) |
| [اینباند Trojan](examples/trojan-inbound/) | Trojan روی WebSocket |
| [اینباند Shadowsocks](examples/shadowsocks-inbound/) | Shadowsocks با رمزنگاری AEAD |
| [اینباند + کلاینت‌ها](examples/inbound-with-client/) | جریان کامل: اینباند + چندین کلاینت |
| [گره کلاستر](examples/node/) | ثبت پنل راه‌دور 3x-ui به‌عنوان گره کلاستر |
| [گروه هاست](examples/host-group/) | مدیریت مسیریابی گروهی هاست‌ها (v3.5.0+) |
| [Observatory اکس‌ری](examples/observatory/) | پیکربندی سنجش تأخیر اوت‌باندها (v3.4.2+) |
| [نسخهٔ Xray](examples/xray-version/) | ثابت‌کردن نسخهٔ نصب‌شدهٔ Xray core |
| [ناوگان چند سروری](examples/multi-server/) | مدیریت چندین هاست 3x-ui با ماژول و `for_each` |
| [import منابع موجود](examples/import-existing/) | وارد کردن منابع موجود 3x-ui به state |

## راهنماها

راهنماهای داخل ریپو برای سناریوهای رایج عملیاتی:

- [Backup-as-code](docs/guides/backup-as-code.md) — پیکربندی بازبینی‌شدهٔ Terraform را با پشتیبان state و پایگاه دادهٔ پنل ترکیب کنید.
- [مهاجرت 3x-ui بین سرورها](docs/guides/server-migration.md) — پایگاه دادهٔ پنل را روی VPS جدید بازیابی و با Terraform بررسی کنید.
- [اضافه‌کردن انبوه کلاینت‌ها](docs/guides/bulk-clients.md) — الگوهای `for_each` و onboarding مبتنی بر CSV.

## مستندات

مستندات کامل در [Terraform Registry](https://registry.terraform.io/providers/batonogov/threexui/latest/docs) موجود است.

### Resources

| Resource | توضیح |
| --- | --- |
| `threexui_inbound` | اینباند (vless, vmess, trojan, shadowsocks, http, mixed, wireguard, amneziawg, tunnel, tun, hysteria, mtproto؛ TUN از 3.2.7، MTProto از 3.3.0 و AmneziaWG از 3.7.0) |
| `threexui_inbound_client` | کلاینت داخل اینباند |
| `threexui_node` | گره کلاستر / ثبت multi-node |
| `threexui_panel_general` | تنظیمات عمومی پنل |
| `threexui_panel_security` | تنظیمات امنیت (2FA) |
| `threexui_panel_user` | اعتبارنامه‌های ادمین |
| `threexui_panel_telegram` | یکپارچه‌سازی با بات تلگرام |
| `threexui_panel_email` | اعلان‌های SMTP/email (v3.4.0+) |
| `threexui_host_group` | مسیریابی گروه هاست‌ها (چند هاست برای هر اینباند) |
| `threexui_panel_subscription` | تنظیمات سرویس subscription |
| `threexui_xray_basics` | پیکربندی پایهٔ Xray (log, policy, api, stats) |
| `threexui_xray_dns` | سرورها و hostهای DNS |
| `threexui_xray_routing` | قوانین routing |
| `threexui_xray_balancers` | بالانسرها |
| `threexui_xray_reverse` | پروکسی معکوس (bridges, portals) |
| `threexui_xray_outbounds` | اوت‌باندها |
| `threexui_xray_observatory` | پیکربندی Xray Observatory / BurstObservatory |
| `threexui_xray_version` | نسخهٔ Xray core نصب‌شده |

### Data Sources

| Data Source | توضیح |
| --- | --- |
| `threexui_inbounds` | لیست همهٔ اینباندها (JSON، حساس) |
| `threexui_nodes` | درخت گره‌های کلاستر / multi-node (JSON، حساس) |
| `threexui_server_status` | وضعیت سرور: CPU، حافظه، دیسک، uptime (JSON) |
| `threexui_settings` | همهٔ تنظیمات پنل (JSON، حساس) |
| `threexui_xray_config` | قالب فعلی Xray (JSON، حساس) |
| `threexui_xray_versions` | نسخه‌های موجود Xray (لیست رشته) |
| `threexui_online_clients` | ایمیل کلاینت‌های آنلاین |
| `threexui_client_traffics` | آمار ترافیک کلاینت‌ها بر اساس ایمیل |

## امنیت

این provider با کلیدهای حساسی که خود پنل تولید می‌کند کار می‌کند (`privateKey` Reality، `secretKey` WireGuard، UUID کلاینت‌ها، توکن بات تلگرام، رمز LDAP). تمام این فیلدها به‌عنوان `Sensitive` علامت‌گذاری شده‌اند و هرگز در لاگ به‌صورت plaintext ظاهر نمی‌شوند. لیست کامل و راهنمای محافظت از Terraform state در [SECURITY.md](SECURITY.md) آمده است.

## توسعه

### پیش‌نیازها

- Go (نسخه در [`go.mod`](go.mod) ثابت شده است)
- [Task](https://taskfile.dev/) — task runner
- [golangci-lint](https://golangci-lint.run/welcome/install/) — linter
- [pre-commit](https://pre-commit.com/) — هوک‌های git
- Docker — برای محیط محلی 3x-ui

### دستورات

```bash
task build        # ساخت provider
task fmt          # فرمت کد (gofmt)
task vet          # go vet
task lint         # golangci-lint
task pre-commit   # تمام بررسی‌ها (fmt, vet, lint, build)
task test:unit    # تست‌های unit (بدون نیاز به Docker / Terraform)
task test:acc     # تست‌های acceptance (به‌صورت خودکار docker compose را بالا می‌آورد)
task test         # unit + acceptance
```

### محیط محلی

```bash
# اجرای 3x-ui روی localhost:2053
docker compose up -d

# لاگین: admin / admin

# توقف
docker compose down
```

## مشارکت

برای راه‌اندازی محلی، تست‌ها و راهنمای ارسال PR به [CONTRIBUTING.md](CONTRIBUTING.md) مراجعه کنید. گزارش باگ، درخواست ویژگی و pull request — همگی پذیرفته می‌شوند. لطفاً بنویسید چه نسخه‌ای از 3x-ui را در پروداکشن استفاده می‌کنید.

## Changelog

ریلیزها بر اساس [Conventional Commits](https://www.conventionalcommits.org/) و به‌صورت خودکار منتشر می‌شوند. تاریخچهٔ کامل نسخه‌ها در [CHANGELOG.md](CHANGELOG.md).

## مجوز

[MIT](LICENSE)

</div>
