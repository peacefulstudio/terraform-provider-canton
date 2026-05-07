# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.2] — 2026-05-07

### Fixed

- **Registry-rendered documentation** — `docs/` is now committed to the source
  tree, so the [Terraform Registry](https://registry.terraform.io/providers/peacefulstudio/canton/latest/docs)
  renders provider, resource, and data-source reference pages instead of
  *"Documentation Unavailable"*. Previously only `templates/` and `examples/`
  shipped; the Registry reads the rendered `docs/` directory directly, not the
  templates. Docs-only change; provider behaviour is unchanged from v0.1.1.

### Changed

- **`templates/` filenames stripped of `canton_` prefix.** The
  `tfplugindocs` generator strips the provider name when matching templates to
  schema entries, so `templates/resources/canton_party.md.tmpl` was orphaned
  and is now `party.md.tmpl`. Same for the other resource and data-source
  templates.

## [0.1.1] — 2026-05-07

### Fixed

- **Terraform Registry compatibility** — `terraform-provider-canton_<version>_manifest.json`
  is now included in `SHA256SUMS`. The Registry validates that every release
  file is checksummed before accepting a version; v0.1.0 omitted the manifest
  and could not be published. Pipeline-only change; provider behaviour is
  unchanged from v0.1.0.

### Changed

- **License changed to Apache-2.0.** The project is now open source under the
  Apache License, Version 2.0. Per-file copyright headers were updated from
  "All rights reserved" to an `SPDX-License-Identifier: Apache-2.0` tag.

## [0.1.0] — 2026-04-17

First stable release.

### Added

- **TLS connectivity through edge proxies** — the provider now dials TLS itself
  via a `grpc.WithContextDialer` that tolerates proxies (Envoy Gateway, Qovery)
  which don't echo `h2` back on ALPN. This is required to talk to Canton
  participants exposed through most ingress setups since grpc-go started
  enforcing ALPN in 1.67.
- **Configure-time health check** — the provider now issues a `Ping`
  (`VersionService.GetLedgerAPIVersion`) at configure time with a 10 s timeout,
  so connectivity, TLS, and OAuth2 misconfigurations surface immediately
  rather than on the first CRUD operation.
- **OAuth2 token refresh** — bearer tokens are now attached per RPC from a
  refreshing `oauth2.TokenSource`, so long-running `terraform apply` runs no
  longer 401 when a short-lived token expires mid-apply.

### Changed

- **Scheme parsing is case-insensitive** and leading/trailing whitespace is
  trimmed. `HTTPS://` and `https://` behave identically.
- **Plaintext + OAuth2 emits a warning** — when `participant_url` is plaintext
  but an OAuth2 token is configured, the provider now warns that the bearer
  token will be transmitted unencrypted.

### Fixed

- **Non-h2 ALPN no longer silently succeeds** — if an upstream negotiates a
  non-h2 ALPN (e.g. an HTTP/1.1 TLS proxy in front of the participant), the
  dialer rejects the connection with a clear error instead of letting it fail
  deep in the HTTP/2 framer on the first RPC.

## [0.1.0-rc.1] — 2026-04-14

First preview release of `terraform-provider-canton` — a Terraform provider for
administering parties, users, and rights on a [Canton](https://www.canton.network/)
participant node via the Ledger API.

### Added

- **Resources** — manage Canton participant state declaratively:
  - `canton_party` — allocate a party on the participant (delete is a no-op, as Canton parties are permanent)
  - `canton_user` — create and delete a user
  - `canton_user_rights` — grant and revoke user rights with in-place updates
- **Data sources** — query existing participant state:
  - `canton_party` — look up a party by ID
  - `canton_user` — look up a user by ID
  - `canton_parties` — list all known parties
- **Import support** for all three resources via `terraform import`
- **OAuth2 authentication** using client credentials, configured via the `oauth2` provider block or environment variables (`CANTON_OAUTH2_TOKEN_URL`, `CANTON_OAUTH2_CLIENT_ID`, `CANTON_OAUTH2_CLIENT_SECRET`, `CANTON_OAUTH2_AUDIENCE`, `CANTON_OAUTH2_SCOPE`)
- **Environment variable fallbacks** for all provider settings, including `CANTON_PARTICIPANT_URL`

### Fixed

- `participant_url` now accepts full URLs (`https://host:port`) — the provider strips the scheme and trailing slash before dialing gRPC ([#17])
- `canton_party` import validates the `hint::fingerprint` format and returns a clear error instead of silently accepting malformed IDs ([#17])

[Unreleased]: https://github.com/peacefulstudio/terraform-provider-canton/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/peacefulstudio/terraform-provider-canton/compare/v0.1.0-rc.1...v0.1.0
[0.1.0-rc.1]: https://github.com/peacefulstudio/terraform-provider-canton/commits/v0.1.0-rc.1
[#17]: https://github.com/peacefulstudio/terraform-provider-canton/pull/17
