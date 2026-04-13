# terraform-provider-canton

Terraform provider for Canton participant node administration (parties, users, rights).

## Project Structure

```
terraform-provider-canton/
├── main.go                              # Terraform plugin server entry point
├── internal/
│   └── provider/
│       ├── provider.go                  # Provider config (endpoint, OAuth2)
│       ├── resource_party.go            # canton_party resource
│       ├── resource_user.go             # canton_user resource
│       └── resource_user_rights.go      # canton_user_rights resource
├── go.mod
└── go.sum
```

## Dependencies

- `github.com/noders-team/go-daml` — Go SDK for Canton Ledger API (gRPC stubs, admin services, auth)
- `github.com/hashicorp/terraform-plugin-framework` — Terraform Plugin Framework

## Development Commands

```bash
go build ./...            # Build
go test ./...             # Run tests
go run .                  # Run the provider
go run . -debug           # Run with debugger support (delve)
```

## Provider Configuration

```hcl
provider "canton" {
  participant_url = "https://participant.example.com"

  oauth2 {
    token_url     = "https://auth.example.com/oauth/token"
    client_id     = var.client_id
    client_secret = var.client_secret
    audience      = "https://canton.example.com"
  }
}
```

Environment variable fallbacks: `CANTON_PARTICIPANT_URL`, `CANTON_OAUTH2_TOKEN_URL`, `CANTON_OAUTH2_CLIENT_ID`, `CANTON_OAUTH2_CLIENT_SECRET`, `CANTON_OAUTH2_AUDIENCE`.

## Resources

| Resource | Description |
|----------|-------------|
| `canton_party` | Allocates a party on the participant (delete = no-op) |
| `canton_user` | Creates/deletes a user on the participant |
| `canton_user_rights` | Grants/revokes user rights (in-place updates) |

## Code Style

- **Go 1.25** with latest language features
- Copyright header: `// Copyright (c) 2026 Peaceful Studio OÜ. All rights reserved.`
- Follow Terraform Plugin Framework conventions
- Use `tflog` for structured logging
