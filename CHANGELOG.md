# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/peacefulstudio/terraform-provider-canton/compare/v0.1.0-rc.1...HEAD
[0.1.0-rc.1]: https://github.com/peacefulstudio/terraform-provider-canton/commits/v0.1.0-rc.1
[#17]: https://github.com/peacefulstudio/terraform-provider-canton/pull/17
