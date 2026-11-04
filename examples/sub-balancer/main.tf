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

resource "threexui_inbound" "de" {
  port     = 443
  protocol = "vless"
  enable   = true
  remark   = "DE"

  vless_settings {
    decryption = "none"
  }
}

resource "threexui_inbound" "nl" {
  port     = 8443
  protocol = "vless"
  enable   = true
  remark   = "NL"

  vless_settings {
    decryption = "none"
  }
}

# Adds one profile to the JSON subscription that balances between both
# inbounds on the client (requires 3x-ui v3.7.0+).
resource "threexui_sub_balancer" "auto" {
  remark      = "Auto · least load"
  strategy    = "leastLoad"
  inbound_ids = [threexui_inbound.de.id, threexui_inbound.nl.id]

  # Lower weight is picked more often; members without a weight count as 1.
  # member_weights requires 3x-ui v3.8.0+.
  member_weights = {
    (threexui_inbound.nl.id) = 2
  }
}
