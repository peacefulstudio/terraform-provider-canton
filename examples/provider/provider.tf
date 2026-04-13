terraform {
  required_providers {
    canton = {
      source  = "peacefulstudio/canton"
      version = "~> 0.1"
    }
  }
}

provider "canton" {
  participant_url = "https://participant.example.com"

  oauth2 {
    token_url     = "https://auth.example.com/oauth/token"
    client_id     = var.canton_client_id
    client_secret = var.canton_client_secret
    audience      = "https://canton.example.com"
  }
}
