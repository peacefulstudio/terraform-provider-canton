# Security Policy

## Supported versions

`terraform-provider-canton` follows [Semantic Versioning][semver]. Security
fixes are issued for the latest minor release on the `prod` branch. Older
minor releases do not receive backports unless coordinated case-by-case with
the maintainers.

[semver]: https://semver.org/spec/v2.0.0.html

| Version  | Supported          |
|----------|--------------------|
| 0.1.x    | :white_check_mark: |
| < 0.1    | :x:                |

## Reporting a vulnerability

**Please do not open a public issue for security-sensitive bugs.**

Report vulnerabilities privately via GitHub's
[private vulnerability reporting][gh-pvr]:

> Repo → **Security** tab → **Report a vulnerability**

[gh-pvr]: https://docs.github.com/en/code-security/security-advisories/guidance-on-reporting-and-writing-information-about-vulnerabilities/privately-reporting-a-security-vulnerability

If you cannot use GitHub for any reason, email **security@peaceful.studio**
instead. Encrypt sensitive details with the maintainers' PGP key on request.

### What to include

- A clear description of the issue and its impact.
- Steps to reproduce, or a minimal proof-of-concept.
- The provider version, Canton participant version, and any relevant
  configuration (with secrets redacted).
- Your preferred contact method and whether you want public credit in the
  advisory.

### What to expect

- **Acknowledgement** within 5 business days.
- **Initial assessment** (severity, affected versions) within 10 business days.
- **Fix or mitigation plan** communicated to the reporter before public
  disclosure.
- **Coordinated disclosure** — we will agree a public-disclosure date with the
  reporter. Default embargo is 90 days from initial report unless a fix is
  released sooner.

## Out of scope

- Vulnerabilities in the upstream [Canton ledger][canton] — please report those
  to Digital Asset.
- Vulnerabilities in third-party dependencies — please report those upstream.
  We will pull in fixes as they become available.
- Misconfiguration of a deployer's own Canton participant or OAuth2 provider
  (e.g. weak client secret, exposing the participant on the public internet
  without auth).

[canton]: https://www.canton.network/

## Credit

We are happy to credit reporters in the public advisory and the changelog
unless you prefer to remain anonymous.
