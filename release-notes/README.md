# Release notes

Each GitHub release's body is taken verbatim from the file in this directory
matching the tag name, e.g. `v0.1.0.md` for tag `v0.1.0`.

## Writing notes for a new release

1. Create `v<version>.md` here before tagging.
2. Follow the structure of the most recent release:
   - `# terraform-provider-canton v<version>` headline
   - One-sentence summary
   - `## What's new since <previous version>` — curated prose per feature
   - `## What's included` — resources / data sources / auth / other
   - `## Getting started` — minimal runnable HCL
   - `## Full changelog` link to `CHANGELOG.md` on the `prod` branch
3. Keep it prose-first. Commit-log dumps belong in `CHANGELOG.md`, not here.

The release workflow (`.github/workflows/release.yaml`) stages the matching
file before GoReleaser runs; if a file is missing the workflow fails loudly
rather than publishing a release with auto-generated commit-log notes.
