terraform {
  required_providers {
    threexui = {
      source = "batonogov/threexui"
    }
  }
}

provider "threexui" {
  endpoint = "https://panel.example.com:2053"
  username = "admin"
  password = "admin"
}

# Pin the Xray core version bundled with 3x-ui v3.11.0.
# Check compatibility with your panel and configuration before changing it.
# The provider calls InstallXray and polls until the panel reports
# the requested version. Delete is a no-op (does not revert the version).
resource "threexui_xray_version" "pinned" {
  version = "v26.10.21"
}
