[English](/README.md) | [فارسی](/README.fa_IR.md) | [العربية](/README.ar_EG.md) | [中文](/README.zh_CN.md) | [Español](/README.es_ES.md) | [Русский](/README.ru_RU.md) | [Türkçe](/README.tr_TR.md)

# Terraform-провайдер для 3x-ui

> Управляйте инбаундами, клиентами, настройками панели и конфигурацией Xray как кодом — делайте бэкапы, миграции и масштабируйте флот VPN/прокси, не кликая по панели [3x-ui](https://github.com/MHSanaei/3x-ui).

[![CI](https://github.com/batonogov/terraform-provider-threexui/actions/workflows/ci.yml/badge.svg)](https://github.com/batonogov/terraform-provider-threexui/actions/workflows/ci.yml)
[![Terraform Registry](https://img.shields.io/badge/terraform-registry-blueviolet)](https://registry.terraform.io/providers/batonogov/threexui/latest)
[![Latest Release](https://img.shields.io/github/v/release/batonogov/terraform-provider-threexui?sort=semver)](https://github.com/batonogov/terraform-provider-threexui/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/batonogov/terraform-provider-threexui)](https://goreportcard.com/report/github.com/batonogov/terraform-provider-threexui)
[![Go Version](https://img.shields.io/github/go-mod/go-version/batonogov/terraform-provider-threexui)](go.mod)
[![Last Commit](https://img.shields.io/github/last-commit/batonogov/terraform-provider-threexui)](https://github.com/batonogov/terraform-provider-threexui/commits/main)
[![Codecov](https://codecov.io/gh/batonogov/terraform-provider-threexui/branch/main/graph/badge.svg)](https://codecov.io/gh/batonogov/terraform-provider-threexui)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

## Зачем это нужно

В продакшене 3x-ui — это десятки инбаундов, сотни клиентов и хрупкая конфигурация Xray, которую легко сломать. С этим провайдером вы получаете:

- **Конфигурация как код** — список инбаундов лежит в git, каждое изменение проходит ревью и версионируется.
- **Безопасная миграция между серверами** — восстановите базу панели, чтобы сохранить ID и секреты, затем проверьте её через Terraform.
- **Бэкап состояния Terraform** — `terraform state pull` экспортирует только управляемые Terraform объекты; для аварийного восстановления нужен также бэкап базы панели.
- **Массовое подключение** — добавьте 100 клиентов одним PR вместо 100 кликов в панели.
- **План перед применением** — `terraform plan` покажет точный дифф изменений до того, как они уйдут в прод.

## Без провайдера vs с провайдером

| Задача | Через UI панели | Через провайдер |
| --- | --- | --- |
| Добавить 50 клиентов | 50 форм по ~30 секунд | один `for_each`, один `apply` |
| Переехать на новый сервер | вручную всё перенести | восстановить базу панели и проверить через `terraform plan` |
| Аудит — у кого сейчас доступ | скроллить список клиентов | `git log` по `.tf`-файлу |
| Откатить кривое изменение | восстановить из JSON-бэкапа | `git revert` + `terraform apply` |
| Синхронизировать stage и prod | экспорт/импорт JSON, разруливать конфликты | общий модуль + переменные на окружение |
| Поменять Reality-ключи на 10 хостах | открыть 10 панелей, кликать в каждой | одна правка переменной, один apply |

## Быстрый старт

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

> **Для пользователей OpenTofu:** указывайте полный адрес реестра:
>
> ```hcl
> source = "registry.terraform.io/batonogov/threexui"
> ```
>
> Провайдер не публикуется в OpenTofu Registry ([причины](https://github.com/opentofu/registry/issues/3632)).

## Совместимость

**Политика поддержки:** провайдер официально поддерживает каждый выпущенный патч во всех поддерживаемых минорных линиях 3x-ui — см. таблицу совместимости ниже. Матрица приёмочного тестирования проверяет каждую версию при каждом пуше в `main` и каждом pull request.

| Версия 3x-ui | Статус |
| --- | --- |
| v3.8.5 | Тестируется |
| v3.8.0 | Тестируется |
| v3.7.0 | Тестируется |
| v3.6.0 | Тестируется |
| v3.5.0 | Тестируется |
| v3.4.2 | Тестируется |
| v3.4.1 | Тестируется |
| v3.4.0 | Тестируется |
| v3.3.1 | Тестируется |
| v3.3.0 | Тестируется |

Новые фичи протоколов помечаются через `requireMinVersion` и автоматически пропускаются на старых версиях, поэтому провайдер живёт на всей матрице без отдельных веток под версии.

## Примеры

| Пример | Описание |
| --- | --- |
| [Конфиг провайдера через env](examples/provider-env-config/) | Настройка через поддерживаемые переменные `THREEXUI_*` |
| [Пользователь панели](examples/panel-user/) | Ротация учётных данных администратора |
| [Email панели](examples/panel-email/) | Настройка SMTP-уведомлений (v3.4.0+) |
| [Trojan-инбаунд](examples/trojan-inbound/) | Trojan через WebSocket |
| [Shadowsocks-инбаунд](examples/shadowsocks-inbound/) | Shadowsocks с AEAD-шифром |
| [Инбаунд + клиенты](examples/inbound-with-client/) | Полный сценарий: инбаунд + несколько клиентов |
| [Узел кластера](examples/node/) | Регистрация удалённой панели 3x-ui как узла кластера |
| [Группа хостов](examples/host-group/) | Массовое управление маршрутизацией хостов (v3.5.0+) |
| [Xray Observatory](examples/observatory/) | Настройка проверок задержки аутбаундов (v3.4.2+) |
| [Версия Xray](examples/xray-version/) | Фиксация установленной версии Xray core |
| [Флот серверов](examples/multi-server/) | Управление несколькими 3x-ui через переиспользуемый модуль и `for_each` |
| [Импорт существующих ресурсов](examples/import-existing/) | Затащить уже существующие ресурсы 3x-ui в state |

## Гайды

Пошаговые инструкции для типовых сценариев:

- [Бэкап как код](docs/guides/backup-as-code.md) — сочетайте проверяемую Terraform-конфигурацию с бэкапами state и базы панели.
- [Миграция 3x-ui между серверами](docs/guides/server-migration.md) — восстановите базу панели на новом VPS и проверьте её через Terraform.
- [Массовое подключение клиентов](docs/guides/bulk-clients.md) — паттерны `for_each` и онбординг из CSV.

## Документация

Полная документация — в [Terraform Registry](https://registry.terraform.io/providers/batonogov/threexui/latest/docs).

### Ресурсы

| Ресурс | Описание |
| --- | --- |
| `threexui_inbound` | Инбаунд (vless, vmess, trojan, shadowsocks, http, mixed, wireguard, amneziawg, tunnel, tun, hysteria, mtproto; TUN с 3.2.7, MTProto с 3.3.0, AmneziaWG с 3.7.0) |
| `threexui_inbound_client` | Клиент внутри инбаунда |
| `threexui_node` | Узел кластера / регистрация multi-node |
| `threexui_panel_general` | Общие настройки панели |
| `threexui_panel_security` | Безопасность (2FA) |
| `threexui_panel_user` | Учётные данные администратора |
| `threexui_panel_telegram` | Интеграция с Telegram-ботом |
| `threexui_panel_email` | SMTP/email-уведомления (v3.4.0+) |
| `threexui_panel_discord` | Discord-уведомления через бота (v3.8.0+) |
| `threexui_host_group` | Маршрутизация групп хостов (несколько хостов на инбаунд) |
| `threexui_panel_subscription` | Настройки подписочного сервиса |
| `threexui_xray_basics` | Базовый Xray (log, policy, api, stats) |
| `threexui_xray_dns` | DNS-серверы и hosts |
| `threexui_xray_routing` | Правила маршрутизации |
| `threexui_xray_balancers` | Балансировщики |
| `threexui_xray_reverse` | Reverse-проксирование (bridges, portals) |
| `threexui_xray_outbounds` | Аутбаунды |
| `threexui_xray_observatory` | Настройки Xray Observatory / BurstObservatory |
| `threexui_xray_version` | Установленная версия Xray-ядра |

### Источники данных

| Data source | Описание |
| --- | --- |
| `threexui_inbounds` | Список всех инбаундов (JSON, sensitive) |
| `threexui_nodes` | Дерево узлов кластера / multi-node (JSON, sensitive) |
| `threexui_server_status` | Статус сервера: CPU, память, диск, uptime (JSON) |
| `threexui_settings` | Все настройки панели (JSON, sensitive) |
| `threexui_xray_config` | Текущий шаблон Xray (JSON, sensitive) |
| `threexui_xray_versions` | Доступные версии Xray (список строк) |
| `threexui_online_clients` | Email'ы клиентов, которые сейчас онлайн |
| `threexui_client_traffics` | Статистика трафика клиентов по email |

## Безопасность

Провайдер работает с секретами, которые выдаёт сама панель (Reality `privateKey`, WireGuard `secretKey`, UUID клиентов, токены Telegram-ботов, пароли LDAP). Все такие поля помечены как `Sensitive` и не попадают в логи. Полный список и рекомендации по защите Terraform state — в [SECURITY.md](SECURITY.md).

## Разработка

### Что нужно

- Go (версия закреплена в [`go.mod`](go.mod))
- [Task](https://taskfile.dev/) — раннер задач
- [golangci-lint](https://golangci-lint.run/welcome/install/) — линтер
- [pre-commit](https://pre-commit.com/) — git-хуки
- Docker — для локального 3x-ui

### Команды

```bash
task build        # Собрать провайдер
task fmt          # gofmt
task vet          # go vet
task lint         # golangci-lint
task pre-commit   # Все проверки разом (fmt, vet, lint, build)
task test:unit    # Юнит-тесты (без Docker и Terraform)
task test:acc     # Acceptance-тесты (поднимает docker compose)
task test         # Юнит + acceptance
```

### Локальное окружение

```bash
# Поднять 3x-ui на localhost:2053
docker compose up -d

# Логин: admin / admin

# Остановить
docker compose down
```

## Контрибьютинг

Локальная сборка, тесты, гайдлайны для PR — в [CONTRIBUTING.md](CONTRIBUTING.md). Баги, запросы фич и pull request'ы приветствуются — как и сообщения о том, какие версии 3x-ui вы крутите в продакшене.

## Changelog

Релизы идут через [Conventional Commits](https://www.conventionalcommits.org/) и публикуются автоматически. Полная история версий — в [CHANGELOG.md](CHANGELOG.md).

## Лицензия

[MIT](LICENSE)
