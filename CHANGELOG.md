# Changelog

All notable changes are recorded here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html). Versions below 1.0.0 are beta: commands, output, and flags may change between minor versions.

## [Unreleased]

## [0.0.1] - 2026-11-20

First beta release.

### Added

- `session-ledger` and `--preview`: read-only inventory of local transcripts for omp, Claude Code, and Codex CLI, grouped by agent and project.
- `--clean`: interactive cleanup that moves selected, eligible transcripts to the native Trash or Recycle Bin after confirmation. There is no permanent-delete fallback.
- `--version`: prints the version, operating system, architecture, and Go version.
- Standalone binaries for Linux (including WSL), macOS, and Windows on amd64 and arm64, with `SHA256SUMS`.

[Unreleased]: https://github.com/tomimandalaputra/session-ledger/compare/v0.0.1...HEAD
[0.0.1]: https://github.com/tomimandalaputra/session-ledger/releases/tag/v0.0.1
