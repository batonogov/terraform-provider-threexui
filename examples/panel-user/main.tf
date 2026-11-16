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
  description = "Currently active panel username; fresh v3.11.0+ panels generate random credentials."
  type        = string
}

variable "bootstrap_password" {
  description = "Currently active panel password; update this before each later password rotation."
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

# Before each later rotation, set bootstrap credentials to the currently active
# credentials, then change panel_password. Authentication happens before updates.
resource "threexui_panel_user" "admin" {
  username = "admin"
  password = var.panel_password
}
