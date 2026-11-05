# Changelog

What changed in each release of plasto. Installs update to the newest release; see
[docs/deploy.md](docs/deploy.md#updates-and-versions).

## Unreleased

- golang.org/x/net 0.62.0, which fixes GO-2026-6860.
- SECURITY.md: how to report a problem privately, and how quickly fixes ship.
- Security scans on every pull request and every Monday: CodeQL, govulncheck, Grype
  (web and worker dependencies, released images), dependency review and gitleaks.
- The containers' filesystems are read-only apart from `pb_data` and `/tmp`.
- The compose file follows the major version (`:1`) instead of `:latest`, and the
  installer fetches the compose files of the newest release instead of `main`.

## 1.0.0 (2026-11-04)

- Releases: a `vX.Y.Z` tag publishes `:X.Y.Z`, `:X.Y`, `:X` and `:latest`, and installs follow
  `:latest`, so they only update to a release. `main` publishes `:edge`.
- Every published image has a signed provenance attestation (`gh attestation verify`).
- Security and maintenance section in the README.
