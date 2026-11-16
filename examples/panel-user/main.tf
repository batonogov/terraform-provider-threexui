terraform {
  required_providers {
    threexui = {
      source = "batonogov/threexui"
    }
  }
}

variable "panel_password" {
  description = "Steady-state admin password, supplied through a secret store or TF_VAR_panel_password."
  type        = string
  sensitive   = true
}

variable "bootstrap_username" {
  description = "Initial panel username; fresh v3.11.0+ panels generate random credentials."
  type        = string
}

variable "bootstrap_password" {
  description = "Initial panel password from the first-start log or an explicit x-ui setting."
  type        = string
  sensitive   = true
}

provider "threexui" {
  endpoint = "https://panel.example.com:2053"
  username = "admin"
  password = var.panel_password

  bootstrap_username = var.bootstrap_username
  bootstrap_password = var.bootstrap_password
}

# Bootstrap credentials allow the first rotation; steady-state credentials
# keep subsequent runs authenticated.
resource "threexui_panel_user" "admin" {
  username = "admin"
  password = var.panel_password
}
