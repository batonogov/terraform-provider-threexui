[English](/README.md) | [فارسی](/README.fa_IR.md) | [العربية](/README.ar_EG.md) | [中文](/README.zh_CN.md) | [Español](/README.es_ES.md) | [Русский](/README.ru_RU.md) | [Türkçe](/README.tr_TR.md)

# 3x-ui 的 Terraform Provider

> 以代码方式管理 [3x-ui](https://github.com/MHSanaei/3x-ui) 的 inbound、客户端、面板设置和 Xray 配置 —— 备份、迁移、横向扩展你的 VPN/代理集群,无需在面板里点点点。

[![CI](https://github.com/batonogov/terraform-provider-threexui/actions/workflows/ci.yml/badge.svg)](https://github.com/batonogov/terraform-provider-threexui/actions/workflows/ci.yml)
[![Terraform Registry](https://img.shields.io/badge/terraform-registry-blueviolet)](https://registry.terraform.io/providers/batonogov/threexui/latest)
[![Latest Release](https://img.shields.io/github/v/release/batonogov/terraform-provider-threexui?sort=semver)](https://github.com/batonogov/terraform-provider-threexui/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/batonogov/terraform-provider-threexui)](https://goreportcard.com/report/github.com/batonogov/terraform-provider-threexui)
[![Go Version](https://img.shields.io/github/go-mod/go-version/batonogov/terraform-provider-threexui)](go.mod)
[![Last Commit](https://img.shields.io/github/last-commit/batonogov/terraform-provider-threexui)](https://github.com/batonogov/terraform-provider-threexui/commits/main)
[![Codecov](https://codecov.io/gh/batonogov/terraform-provider-threexui/branch/main/graph/badge.svg)](https://codecov.io/gh/batonogov/terraform-provider-threexui)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

## 为什么用它

在生产环境跑 3x-ui,意味着要维护几十个 inbound、几百个客户端,以及一份很容易跑偏的 Xray 配置。用这个 provider 可以:

- **配置即代码** —— inbound 列表存在 git 里,每次变更都经过 review、可追溯。
- **安全地跨服务器迁移** —— 恢复面板数据库以保留 ID 和密钥，然后用 Terraform 校验。
- **备份 Terraform state** —— `terraform state pull` 只导出由 Terraform 管理的对象；灾难恢复还需要面板数据库备份。
- **批量上线** —— 一个 PR 加 100 个客户端,而不是在面板里点 100 次。
- **上线前预演** —— `terraform plan` 在执行前清楚地告诉你会改什么。

## 不用 provider vs 用 provider

| 任务 | 面板 UI | 本 provider |
| --- | --- | --- |
| 添加 50 个客户端 | 50 个表单,每个约 30 秒 | 一个 `for_each`,一次 `apply` |
| 迁移到新服务器 | 手动重新输入 | 恢复面板数据库，然后用 `terraform plan` 校验 |
| 审计当前谁有访问权 | 翻客户端列表 | 对 `.tf` 文件 `git log` |
| 回滚错误改动 | 从 JSON 备份恢复 | `git revert` + `terraform apply` |
| 同步 staging ↔ 生产 | 导出/导入 JSON,手动调和 | 共享模块 + 按环境变量 |
| 在 10 台主机上轮换 Reality 密钥 | 打开 10 个面板逐个点 | 改一个变量,执行一次 apply |

## 快速开始

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

> **OpenTofu 用户：**请使用完整的 registry 地址：
>
> ```hcl
> source = "registry.terraform.io/batonogov/threexui"
> ```
>
> 本 provider 未发布到 OpenTofu Registry（[原因](https://github.com/opentofu/registry/issues/3632)）。

## 兼容性

**支持策略:** 本 provider 正式支持所有受支持的 3x-ui 次要版本线中的每个已发布补丁版本 —— 见下方兼容性表格。acceptance 矩阵在每次 push 到 `main` 和每个 pull request 时测试每个版本。

| 3x-ui 版本 | 状态 |
| --- | --- |
| v3.8.5 | 已测试 |
| v3.8.0 | 已测试 |
| v3.7.0 | 已测试 |
| v3.6.0 | 已测试 |
| v3.5.0 | 已测试 |
| v3.4.2 | 已测试 |
| v3.4.1 | 已测试 |
| v3.4.0 | 已测试 |
| v3.3.1 | 已测试 |
| v3.3.0 | 已测试 |

新版本协议特性通过 `requireMinVersion` 守门,在老版本上自动跳过 —— 无需为每个版本拉分支。

## 示例

| 示例 | 说明 |
| --- | --- |
| [通过环境变量配置 provider](examples/provider-env-config/) | 通过支持的 `THREEXUI_*` 环境变量配置 |
| [面板用户](examples/panel-user/) | 轮换面板管理员凭据 |
| [面板邮件](examples/panel-email/) | 配置 SMTP 通知 (v3.4.0+) |
| [Trojan inbound](examples/trojan-inbound/) | Trojan 走 WebSocket |
| [Shadowsocks inbound](examples/shadowsocks-inbound/) | Shadowsocks AEAD 加密 |
| [inbound + 客户端](examples/inbound-with-client/) | 完整流程:inbound + 多个客户端 |
| [集群节点](examples/node/) | 将远程 3x-ui 面板注册为集群节点 |
| [主机组](examples/host-group/) | 管理批量主机路由 (v3.5.0+) |
| [Xray Observatory](examples/observatory/) | 配置 outbound 延迟探测 (v3.4.2+) |
| [Xray 版本](examples/xray-version/) | 固定已安装的 Xray core 版本 |
| [多服务器集群](examples/multi-server/) | 通过模块 + `for_each` 管理多台 3x-ui |
| [导入已有资源](examples/import-existing/) | 把已有的 3x-ui 资源拉进 state |

## 操作指南

仓库内的常见运维场景指南:

- [备份即代码](docs/guides/backup-as-code.md) —— 将经过审查的 Terraform 配置与 state、面板数据库备份结合使用。
- [跨服务器迁移 3x-ui](docs/guides/server-migration.md) —— 在新 VPS 上恢复面板数据库，再用 Terraform 校验。
- [批量上线客户端](docs/guides/bulk-clients.md) —— `for_each` 模式和基于 CSV 的批量上线。

## 文档

完整文档见 [Terraform Registry](https://registry.terraform.io/providers/batonogov/threexui/latest/docs)。

### Resources

| Resource | 说明 |
| --- | --- |
| `threexui_inbound` | inbound 代理(vless、vmess、trojan、shadowsocks、http、mixed、wireguard、amneziawg、tunnel、tun、hysteria、mtproto；TUN 3.2.7+，MTProto 3.3.0+，AmneziaWG 3.7.0+) |
| `threexui_inbound_client` | inbound 内的客户端 |
| `threexui_node` | 集群节点 / multi-node 注册 |
| `threexui_panel_general` | 面板通用设置 |
| `threexui_panel_security` | 安全设置(2FA) |
| `threexui_panel_user` | 管理员凭据 |
| `threexui_panel_telegram` | Telegram bot 集成 |
| `threexui_panel_email` | SMTP/email 通知 (v3.4.0+) |
| `threexui_panel_discord` | Discord 机器人通知 (v3.8.0+) |
| `threexui_host_group` | 主机组路由（每个 inbound 可有多个主机） |
| `threexui_panel_subscription` | 订阅服务设置 |
| `threexui_xray_basics` | Xray 基础配置(log、policy、api、stats) |
| `threexui_xray_dns` | DNS 服务器和 hosts |
| `threexui_xray_routing` | 路由规则 |
| `threexui_xray_balancers` | 负载均衡 |
| `threexui_xray_reverse` | 反向代理(bridges、portals) |
| `threexui_xray_outbounds` | outbound |
| `threexui_xray_observatory` | Xray Observatory / BurstObservatory 配置 |
| `threexui_xray_version` | 已安装的 Xray core 版本 |

### Data Sources

| Data Source | 说明 |
| --- | --- |
| `threexui_inbounds` | 所有 inbound(JSON,敏感) |
| `threexui_nodes` | 集群节点树 / multi-node（JSON，敏感） |
| `threexui_server_status` | 服务器状态:CPU、内存、磁盘、uptime(JSON) |
| `threexui_settings` | 所有面板设置(JSON,敏感) |
| `threexui_xray_config` | 当前 Xray 模板(JSON,敏感) |
| `threexui_xray_versions` | 可用的 Xray 版本(字符串列表) |
| `threexui_online_clients` | 当前在线客户端 email |
| `threexui_client_traffics` | 按 email 的客户端流量统计 |

## 安全

provider 会处理面板自动派发的各种密钥(Reality `privateKey`、WireGuard `secretKey`、客户端 UUID、Telegram bot token、LDAP 密码)。所有此类字段都标记为 `Sensitive`,日志里不会出现明文。完整列表和 Terraform state 保护建议见 [SECURITY.md](SECURITY.md)。

## 开发

### 依赖

- Go（版本以 [`go.mod`](go.mod) 为准）
- [Task](https://taskfile.dev/) —— 任务运行器
- [golangci-lint](https://golangci-lint.run/welcome/install/) —— linter
- [pre-commit](https://pre-commit.com/) —— git hooks 框架
- Docker —— 用于本地 3x-ui 环境

### 命令

```bash
task build        # 构建 provider
task fmt          # 格式化代码(gofmt)
task vet          # go vet
task lint         # golangci-lint
task pre-commit   # 一次性跑所有检查(fmt、vet、lint、build)
task test:unit    # 单元测试(无需 Docker / Terraform)
task test:acc     # 验收测试(自动启动 docker compose)
task test         # 单元 + 验收
```

### 本地环境

```bash
# 在 localhost:2053 启动 3x-ui
docker compose up -d

# 登录:admin / admin

# 停止
docker compose down
```

## 贡献

本地配置、测试、提交规范见 [CONTRIBUTING.md](CONTRIBUTING.md)。欢迎提 issue、提需求、提 PR —— 也欢迎告诉我们你在生产里跑的是哪个 3x-ui 版本。

## Changelog

发布走 [Conventional Commits](https://www.conventionalcommits.org/),自动发版。完整版本历史见 [CHANGELOG.md](CHANGELOG.md)。

## License

[MIT](LICENSE)
