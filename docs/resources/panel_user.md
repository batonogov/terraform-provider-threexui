---
page_title: "threexui_panel_user Resource - 3x-ui"
subcategory: "Panel Settings"
description: |-
  Manages the admin username and password for the 3x-ui panel.
---

# threexui_panel_user (Resource)

Manages the admin username and password for the 3x-ui panel.

This is a singleton resource -- only one instance should exist per provider. Deleting this resource only removes it from Terraform state; it does not revert the credentials.

~> **Warning:** After applying changes, keep the provider's steady-state `username` and `password` in sync with this resource. For first-run bootstrap of a fresh panel, the provider-level `bootstrap_username` and `bootstrap_password` arguments can authenticate with the initial credentials once, then this resource can rotate the panel to the steady-state credentials during the same apply.

Before each later rotation, set the provider's bootstrap credentials to the panel's currently active username and password, then change the desired credentials in both the provider and this resource. The provider authenticates before updating resources, so the original installation credentials cannot bootstrap another rotation.

## Example Usage

```hcl
resource "threexui_panel_user" "admin" {
  username = "myadmin"
  password = "s3cureP@ss"
}
```

### Write-Only Password (Terraform 1.11+ / OpenTofu 1.11+)

```hcl
resource "threexui_panel_user" "admin" {
  username            = "myadmin"
  password_wo         = "s3cureP@ss"
  password_wo_version = 1
}
```

## Argument Reference

- `username` (Required, String) - The desired admin username.
- `password` (Optional, String, Sensitive) - The desired admin password. Prefer `password_wo` on Terraform 1.11+ / OpenTofu 1.11+.
- `password_wo` (Optional, String, WriteOnly) - Write-only version of `password`. Not persisted in state. Terraform 1.11+ / OpenTofu 1.11+.
- `password_wo_version` (Optional, Number) - Increment to trigger re-send of `password_wo`. Must be set together with `password_wo`.

## Attribute Reference

- `id` (String) - Always `"user"`.

## Import

```shell
terraform import threexui_panel_user.admin user
```

~> **Note:** Because the panel has no API to read credentials, `username` and `password` are unknown after import. You must configure both attributes in your Terraform configuration and run `terraform apply` to synchronize the state. The next `terraform plan` after import will show changes for these attributes — this is expected.
