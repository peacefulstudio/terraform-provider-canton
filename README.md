# terraform-provider-canton

Terraform provider for Canton participant node administration (parties, users, rights)

## Development

### Prerequisites


- [Go 1.25+](https://go.dev/dl/)


### Getting Started

```bash
git clone https://github.com/peacefulstudio/terraform-provider-canton.git
cd terraform-provider-canton
git checkout dev
```


```bash
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
