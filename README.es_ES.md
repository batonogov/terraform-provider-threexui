[English](/README.md) | [فارسی](/README.fa_IR.md) | [العربية](/README.ar_EG.md) | [中文](/README.zh_CN.md) | [Español](/README.es_ES.md) | [Русский](/README.ru_RU.md) | [Türkçe](/README.tr_TR.md)

# Proveedor Terraform para 3x-ui

> Gestiona inbounds, clientes, ajustes del panel y la configuración de Xray como código — haz copias de seguridad, migra y escala tu flota VPN/proxy sin tocar el panel de [3x-ui](https://github.com/MHSanaei/3x-ui).

[![CI](https://github.com/batonogov/terraform-provider-threexui/actions/workflows/ci.yml/badge.svg)](https://github.com/batonogov/terraform-provider-threexui/actions/workflows/ci.yml)
[![Terraform Registry](https://img.shields.io/badge/terraform-registry-blueviolet)](https://registry.terraform.io/providers/batonogov/threexui/latest)
[![Latest Release](https://img.shields.io/github/v/release/batonogov/terraform-provider-threexui?sort=semver)](https://github.com/batonogov/terraform-provider-threexui/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/batonogov/terraform-provider-threexui)](https://goreportcard.com/report/github.com/batonogov/terraform-provider-threexui)
[![Go Version](https://img.shields.io/github/go-mod/go-version/batonogov/terraform-provider-threexui)](go.mod)
[![Last Commit](https://img.shields.io/github/last-commit/batonogov/terraform-provider-threexui)](https://github.com/batonogov/terraform-provider-threexui/commits/main)
[![Codecov](https://codecov.io/gh/batonogov/terraform-provider-threexui/branch/main/graph/badge.svg)](https://codecov.io/gh/batonogov/terraform-provider-threexui)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

## Por qué usarlo

Operar 3x-ui en producción significa decenas de inbounds, cientos de clientes y una configuración de Xray que se rompe con facilidad. Con este proveedor puedes:

- **Tratar la configuración como código** — tu lista de inbounds vive en git, cada cambio se revisa y se versiona.
- **Migrar entre servidores de forma segura** — restaura la base de datos del panel para conservar IDs y secretos, y verifícala con Terraform.
- **Respaldar el estado de Terraform** — `terraform state pull` exporta solo los objetos gestionados por Terraform; acompáñalo con un respaldo de la base de datos del panel.
- **Onboarding masivo** — añade 100 clientes en un único PR en lugar de 100 clics en el panel.
- **Plan antes de producción** — `terraform plan` muestra exactamente lo que va a cambiar antes de aplicarlo.

## Sin proveedor vs con proveedor

| Tarea | Panel UI | Este proveedor |
| --- | --- | --- |
| Añadir 50 clientes | 50 formularios, ~30 s cada uno | un `for_each`, un `apply` |
| Migrar a un servidor nuevo | reintroducir todo a mano | restaurar la base de datos y verificar con `terraform plan` |
| Auditar quién tiene acceso | recorrer la lista de clientes | `git log` sobre un `.tf` |
| Revertir un cambio incorrecto | restaurar desde un backup JSON | `git revert` + `terraform apply` |
| Sincronizar staging ↔ producción | exportar/importar JSON, resolver conflictos | módulo compartido + variables por entorno |
| Rotar claves Reality en 10 hosts | abrir 10 paneles, clic en cada uno | un cambio de variable, un apply |

## Inicio rápido

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

> **Usuarios de OpenTofu:** usen la dirección completa del registro:
>
> ```hcl
> source = "registry.terraform.io/batonogov/threexui"
> ```
>
> El provider no está disponible en el OpenTofu Registry ([motivo](https://github.com/opentofu/registry/issues/3632)).

## Compatibilidad

**Política de soporte:** el proveedor soporta oficialmente cada parche publicado en todas las líneas menores de 3x-ui soportadas — consulta la tabla de compatibilidad a continuación. La matriz de acceptance prueba cada versión en cada push a `main` y en cada pull request.

| Versión de 3x-ui | Estado |
| --- | --- |
| v3.8.5 | Probado |
| v3.8.0 | Probado |
| v3.7.0 | Probado |
| v3.6.0 | Probado |
| v3.5.0 | Probado |
| v3.4.2 | Probado |
| v3.4.1 | Probado |
| v3.4.0 | Probado |
| v3.3.1 | Probado |
| v3.3.0 | Probado |

Las funciones de protocolos nuevos están protegidas con `requireMinVersion` y se omiten automáticamente en versiones antiguas, por lo que el proveedor funciona en toda la matriz sin forks por versión.

## Ejemplos

| Ejemplo | Descripción |
| --- | --- |
| [Proveedor con env](examples/provider-env-config/) | Configurar el proveedor con variables `THREEXUI_*` compatibles |
| [Usuario del panel](examples/panel-user/) | Rotar las credenciales del administrador |
| [Email del panel](examples/panel-email/) | Configurar notificaciones SMTP (v3.4.0+) |
| [Inbound Trojan](examples/trojan-inbound/) | Trojan sobre WebSocket |
| [Inbound Shadowsocks](examples/shadowsocks-inbound/) | Shadowsocks con cifrado AEAD |
| [Inbound + clientes](examples/inbound-with-client/) | Flujo completo: inbound + varios clientes |
| [Nodo del clúster](examples/node/) | Registrar un panel 3x-ui remoto como nodo |
| [Grupo de hosts](examples/host-group/) | Gestionar el enrutamiento masivo de hosts (v3.5.0+) |
| [Observatorio Xray](examples/observatory/) | Configurar sondas de latencia de outbounds (v3.4.2+) |
| [Versión de Xray](examples/xray-version/) | Fijar la versión instalada de Xray core |
| [Flota multi-servidor](examples/multi-server/) | Múltiples hosts 3x-ui con un módulo y `for_each` |
| [Importar recursos existentes](examples/import-existing/) | Importar recursos 3x-ui ya creados al estado |

## Guías

Walkthroughs en el propio repo para escenarios habituales:

- [Backup-as-code](docs/guides/backup-as-code.md) — combina configuración Terraform revisada con respaldos del estado y de la base de datos.
- [Migración entre servidores](docs/guides/server-migration.md) — restaura la base de datos en un VPS nuevo y verifícala con Terraform.
- [Onboarding masivo de clientes](docs/guides/bulk-clients.md) — patrones `for_each` y onboarding desde CSV.

## Documentación

La documentación completa está en el [Terraform Registry](https://registry.terraform.io/providers/batonogov/threexui/latest/docs).

### Recursos

| Recurso | Descripción |
| --- | --- |
| `threexui_inbound` | Inbound (vless, vmess, trojan, shadowsocks, http, mixed, wireguard, amneziawg, tunnel, tun, hysteria, mtproto; TUN 3.2.7+, MTProto 3.3.0+, AmneziaWG 3.7.0+) |
| `threexui_inbound_client` | Cliente dentro de un inbound |
| `threexui_node` | Nodo del clúster / registro multi-node |
| `threexui_panel_general` | Ajustes generales del panel |
| `threexui_panel_security` | Ajustes de seguridad (2FA) |
| `threexui_panel_user` | Credenciales de administrador |
| `threexui_panel_telegram` | Integración con bot de Telegram |
| `threexui_panel_email` | Notificaciones SMTP/email (v3.4.0+) |
| `threexui_panel_discord` | Notificaciones del bot de Discord (v3.8.0+) |
| `threexui_host_group` | Enrutamiento de grupos de hosts (varios hosts por inbound) |
| `threexui_panel_subscription` | Ajustes del servicio de suscripción |
| `threexui_xray_basics` | Configuración base de Xray (log, policy, api, stats) |
| `threexui_xray_dns` | Servidores y hosts DNS |
| `threexui_xray_routing` | Reglas de enrutamiento |
| `threexui_xray_balancers` | Balanceadores |
| `threexui_xray_reverse` | Proxy inverso (bridges, portals) |
| `threexui_xray_outbounds` | Outbounds |
| `threexui_xray_observatory` | Configuración de Xray Observatory / BurstObservatory |
| `threexui_xray_version` | Versión del núcleo Xray instalada |

### Data sources

| Data source | Descripción |
| --- | --- |
| `threexui_inbounds` | Lista de todos los inbounds (JSON, sensible) |
| `threexui_nodes` | Árbol de nodos del clúster / multi-node (JSON, sensible) |
| `threexui_server_status` | Estado del servidor: CPU, memoria, disco, uptime (JSON) |
| `threexui_settings` | Todos los ajustes del panel (JSON, sensible) |
| `threexui_xray_config` | Plantilla actual de Xray (JSON, sensible) |
| `threexui_xray_versions` | Versiones de Xray disponibles (lista de strings) |
| `threexui_online_clients` | Emails de clientes en línea |
| `threexui_client_traffics` | Estadísticas de tráfico por email |

## Seguridad

El proveedor maneja secretos que el panel emite automáticamente (Reality `privateKey`, WireGuard `secretKey`, UUIDs de clientes, tokens de bots de Telegram, contraseñas LDAP). Todos esos campos están marcados como `Sensitive` y nunca se registran en texto plano. Consulta [SECURITY.md](SECURITY.md) para la lista completa y para guías sobre cómo proteger tu estado de Terraform.

## Desarrollo

### Requisitos

- Go (versión fijada en [`go.mod`](go.mod))
- [Task](https://taskfile.dev/) — runner de tareas
- [golangci-lint](https://golangci-lint.run/welcome/install/) — linter
- [pre-commit](https://pre-commit.com/) — hooks de git
- Docker — para el entorno local de 3x-ui

### Comandos

```bash
task build        # Compilar el proveedor
task fmt          # Formatear código (gofmt)
task vet          # go vet
task lint         # golangci-lint
task pre-commit   # Todas las comprobaciones (fmt, vet, lint, build)
task test:unit    # Tests unitarios (sin Docker ni Terraform)
task test:acc     # Tests de aceptación (levanta docker compose)
task test         # Unitarios + aceptación
```

### Entorno local

```bash
# Levantar 3x-ui en localhost:2053
docker compose up -d

# Login: admin / admin

# Parar
docker compose down
```

## Contribuir

Mira [CONTRIBUTING.md](CONTRIBUTING.md) para configuración local, pruebas y guías de envío. Bug reports, peticiones de funcionalidades y pull requests son bienvenidos — y también las notas sobre qué versiones de 3x-ui usas en producción.

## Changelog

Las releases siguen [Conventional Commits](https://www.conventionalcommits.org/) y se publican automáticamente. Historial completo en [CHANGELOG.md](CHANGELOG.md).

## Licencia

[MIT](LICENSE)
