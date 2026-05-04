# Contributing to terraform-provider-canton

Thanks for your interest. This document covers everything you need to send a
patch — from cloning the repo to getting your PR merged.

## Code of Conduct

By participating in this project you agree to abide by the
[Code of Conduct](CODE_OF_CONDUCT.md).

## Getting set up

```bash
git clone https://github.com/peacefulstudio/terraform-provider-canton.git
cd terraform-provider-canton
git checkout dev
go build ./...
go test ./...
```

You'll need [Go 1.25+](https://go.dev/dl/).

### Pre-commit hook

The repo ships a pre-commit hook (lint + build + test) under `.githooks/`.
Activate it once after cloning:

```bash
git config core.hooksPath .githooks
```

## Branching model

| Branch  | Purpose                          |
|---------|----------------------------------|
| `dev`   | Default branch — open PRs here   |
| `stage` | Staging / pre-production         |
| `prod`  | Production — release tags only   |

All PRs target `dev`. Promotion to `stage` and `prod` is handled by the
maintainers.

## Test-driven development

Bug fixes and new features must follow red-green TDD:

1. **Red** — write a failing test that describes the desired behaviour.
2. **Green** — write the minimum production code to make it pass.
3. **Refactor** — clean up while keeping tests green.

Unit tests live alongside source files (e.g. `resource_party_test.go` next to
`resource_party.go`). Mock implementations of `go-daml` interfaces live in
`mock_test.go`. Test helpers for constructing Terraform-framework objects live
in `testhelper_test.go`.

```bash
go test -v -race -coverprofile=coverage.out ./internal/...
go tool cover -func=coverage.out      # coverage by function
go tool cover -html=coverage.out      # HTML report
```

> **Note:** scope to `./internal/...` rather than `./...` — `main.go` is the
> plugin server entry point with no tests by design, and including it drags
> the report to a misleading 0% on the root package. The build step
> (`go build ./...`) still validates `main.go` compiles. CI does the same.

### Acceptance tests

Acceptance tests (`*_acc_test.go`, gated behind `TF_ACC=1`) run against a real
Canton participant. They are skipped in CI by default. To run locally:

```bash
export TF_ACC=1
export CANTON_PARTICIPANT_URL="http://<participant-host>:<port>"
# Optional, if the participant requires OAuth2:
# export CANTON_OAUTH2_TOKEN_URL=...
# export CANTON_OAUTH2_CLIENT_ID=...
# export CANTON_OAUTH2_CLIENT_SECRET=...
go test -v ./internal/provider/ -run TestAcc
```

If you don't have access to a Canton participant, that's fine — open the PR
without running them and a maintainer will run them for you.

## Code style

- **Go 1.25** with the latest language features.
- Every Go source file starts with the two-line copyright header:
  ```go
  // Copyright (c) 2026 Peaceful Studio OÜ
  // SPDX-License-Identifier: Apache-2.0
  ```
- Follow Terraform Plugin Framework conventions.
- Use `tflog` for structured logging.
- Code should be expressive enough to not need comments. Add a comment only
  when the *why* is non-obvious (a workaround for an external bug, a hidden
  invariant). Don't comment on *what* the code does.

## Documentation

User-facing docs are generated from the templates and examples by
[`tfplugindocs`](https://github.com/hashicorp/terraform-plugin-docs):

```bash
go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest
tfplugindocs generate
```

If your change touches a public attribute, update the template under
`templates/` and the example under `examples/` in the same PR.

## Opening a pull request

1. Create a feature branch from `dev`.
2. Commit using the [Conventional Commits](https://www.conventionalcommits.org/)
   format (`feat:`, `fix:`, `chore:`, `docs:`, `refactor:`, `test:`).
3. Push and open a PR targeting `dev`.
4. Fill out the PR template — explicitly call out anything that affects
   provider behaviour, schema, or state migration.
5. Make sure CI passes (build, tests, lint, coverage report).
6. Request review. A maintainer will respond — the maintainers may also
   invoke an automated Claude reviewer (`peaceful-bot`) on top of human
   review; treat its comments as suggestions rather than blockers.

For user-visible changes, add an entry to the `[Unreleased]` section of
[`CHANGELOG.md`](CHANGELOG.md). Skip this for purely internal refactors,
test-only changes, CI tweaks, and dependency bumps that don't alter behaviour.

## Reporting bugs

Open an issue using the "Bug report" template. The more reproducible the
report, the faster the fix.

For security-sensitive bugs, **do not open a public issue** — see
[SECURITY.md](SECURITY.md).

## License

By contributing, you agree that your contributions will be licensed under the
[Apache License 2.0](LICENSE), the same license as the project. No CLA
required.
