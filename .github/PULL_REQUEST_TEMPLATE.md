<!--
Thanks for the PR. Please fill out the sections below — empty PRs slow down review.
For trivial changes (typo, formatting), feel free to delete inapplicable sections.
-->

## Summary

<!-- 1–3 bullets: what does this change, and why? Link to an issue if there is one. -->

-

## Type of change

- [ ] Bug fix (non-breaking)
- [ ] New feature (non-breaking)
- [ ] Breaking change (provider behaviour, schema, or state)
- [ ] Refactor / internal-only
- [ ] Docs / examples
- [ ] CI / build / chore

## Test plan

<!-- How did you verify this works? Acceptance tests against a real participant get bonus points. -->

- [ ] `go test ./...` passes locally
- [ ] Acceptance tests run against a Canton participant (`TF_ACC=1`) — describe which ones, or N/A
- [ ] Manual `terraform plan` / `apply` performed — describe the scenario, or N/A

## Checklist

- [ ] Followed red-green TDD (test added or updated before/with the production change)
- [ ] All Go files include the standard copyright header
- [ ] Updated `templates/` and/or `examples/` if the public schema changed
- [ ] Added a `[Unreleased]` entry in `CHANGELOG.md` for user-visible changes
- [ ] Verified `golangci-lint` is clean
- [ ] No secrets or credentials in code, fixtures, or commit messages

## Breaking changes / migration notes

<!--
If this is a breaking change, describe:
- What breaks (HCL syntax, attribute names, defaults, state shape)
- How users migrate (state mv, manual edit, automatic upgrade)
Otherwise: "None."
-->

None.
