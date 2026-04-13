# terraform-provider-canton

[![CI](https://github.com/peacefulstudio/terraform-provider-canton/actions/workflows/ci.yaml/badge.svg?branch=dev)](https://github.com/peacefulstudio/terraform-provider-canton/actions/workflows/ci.yaml)
[![codecov](https://codecov.io/gh/peacefulstudio/terraform-provider-canton/branch/dev/graph/badge.svg)](https://codecov.io/gh/peacefulstudio/terraform-provider-canton)

Terraform provider for Canton participant node administration — manage parties, users, and rights declaratively.

## Usage

```hcl
terraform {
  required_providers {
    canton = {
      source = "peacefulstudio/canton"
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

# Allocate parties
resource "canton_party" "validator" {
  party_id_hint = "validator"
}

resource "canton_party" "treasury" {
  party_id_hint = "treasury"
}

resource "canton_party" "pqs" {
  party_id_hint = "pqs-reader"
}

# Create a user bound to a party
resource "canton_user" "treasury_service" {
  user_id       = "treasury-service"
  primary_party = canton_party.treasury.party_id
}

# Grant rights to the user
resource "canton_user_rights" "treasury_rights" {
  user_id = canton_user.treasury_service.user_id

  act_as  = [canton_party.treasury.party_id]
  read_as = [canton_party.validator.party_id]
}
```

## Resources

### canton_party

Allocates a party on a Canton participant node.

| Attribute | Type | Description |
|-----------|------|-------------|
| `party_id_hint` | string, required | Hint for the party ID (ForceNew) |
| `party_id` | string, computed | The allocated party ID |
| `is_local` | bool, computed | Whether the party is local |

> **Note:** Canton parties cannot be deleted. `terraform destroy` removes the party from state but does not affect the ledger.

### canton_user

Manages a user on a Canton participant node.

| Attribute | Type | Description |
|-----------|------|-------------|
| `user_id` | string, required | User identifier (ForceNew) |
| `primary_party` | string, required | Primary party (ForceNew) |

### canton_user_rights

Manages rights granted to a user. Supports in-place updates.

| Attribute | Type | Description |
|-----------|------|-------------|
| `user_id` | string, required | Target user (ForceNew) |
| `act_as` | set(string), optional | Parties the user can act as |
| `read_as` | set(string), optional | Parties the user can read as |
| `participant_admin` | bool, optional | Participant admin right |
| `identity_provider_admin` | bool, optional | IDP admin right |

## Configuration

The provider can be configured via HCL attributes or environment variables:

| HCL Attribute | Environment Variable |
|---------------|---------------------|
| `participant_url` | `CANTON_PARTICIPANT_URL` |
| `oauth2.token_url` | `CANTON_OAUTH2_TOKEN_URL` |
| `oauth2.client_id` | `CANTON_OAUTH2_CLIENT_ID` |
| `oauth2.client_secret` | `CANTON_OAUTH2_CLIENT_SECRET` |
| `oauth2.audience` | `CANTON_OAUTH2_AUDIENCE` |

## Development

### Prerequisites

- [Go 1.25+](https://go.dev/dl/)

### Getting Started

```bash
git clone https://github.com/peacefulstudio/terraform-provider-canton.git
cd terraform-provider-canton
git checkout dev
go build ./...
go test ./...
```

## Branches

| Branch | Purpose |
|--------|---------|
| dev | Development (default) |
| stage | Staging / pre-production |
| prod | Production |

## Contributing

1. Create a feature branch from dev
2. Make your changes
3. Open a PR targeting dev
4. Ensure CI checks pass
5. Request review (or assign peaceful-bot for Claude Code review)

## License

Copyright (c) 2026 Peaceful Studio OÜ. All rights reserved.
