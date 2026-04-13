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

### Git Hooks

The repo uses a pre-commit hook (lint + build + test) in `.githooks/`. Activate it after cloning:

```bash
git config core.hooksPath .githooks
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

## Testing

### Red-Green TDD (mandatory for new features)

All new features and bug fixes must follow red-green TDD:

1. **Red** — Write a failing test that describes the desired behavior
2. **Green** — Write the minimum production code to make the test pass
3. **Refactor** — Clean up while keeping tests green

### Test Architecture

- Unit tests live alongside source files: `resource_party_test.go` next to `resource_party.go`
- Mock implementations of go-daml interfaces are in `mock_test.go`
- Test helpers for constructing Terraform framework objects are in `testhelper_test.go`
- Resource CRUD methods are tested by injecting mock services and constructing `tfsdk.Plan`/`tfsdk.State` from `tftypes.Value`
- `provider.Configure` is not unit-tested (requires real OAuth2/gRPC) — to be covered by integration tests

### Running Tests

```bash
go test ./...                                    # Run all tests
go test -v -race -coverprofile=coverage.out ./...  # With race detection + coverage
go tool cover -func=coverage.out                 # Coverage report by function
go tool cover -html=coverage.out                 # HTML coverage report
```

## Code Style

- **Go 1.25** with latest language features
- Copyright header: `// Copyright (c) 2026 Peaceful Studio OÜ. All rights reserved.`
- Follow Terraform Plugin Framework conventions
- Use `tflog` for structured logging
