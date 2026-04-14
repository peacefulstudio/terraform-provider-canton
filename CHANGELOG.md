# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0-rc.1] — 2026-04-14

Initial preview release of the Canton Terraform provider.

### Added

- `canton_party` resource — allocate a party on a Canton participant node (#9)
- `canton_user` resource — create and delete users on a Canton participant (#9)
- `canton_user_rights` resource — grant and revoke user rights with in-place updates (#9)
- `canton_party` data source — look up an existing party by ID (#11)
- `canton_user` data source — look up an existing user by ID (#11)
- `canton_parties` data source — list all known parties on the participant (#11)
- `terraform import` support for all three resources (#9)
- OAuth2 client-credentials authentication via the `oauth2` provider block (#9)
- Environment variable fallbacks for all provider configuration (`CANTON_PARTICIPANT_URL`, `CANTON_OAUTH2_*`) (#9)

### Fixed

- Provider now strips `http://` / `https://` scheme and trailing slash from `participant_url` before gRPC dial, so pasting a full URL no longer causes a connection error (#17)
- `canton_party` import validates the `hint::fingerprint` format and returns an actionable error instead of silently accepting malformed IDs (#17)

[Unreleased]: https://github.com/peacefulstudio/terraform-provider-canton/compare/v0.1.0-rc.1...HEAD
[0.1.0-rc.1]: https://github.com/peacefulstudio/terraform-provider-canton/commits/v0.1.0-rc.1
