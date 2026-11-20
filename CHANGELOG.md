# Changelog

All notable changes are recorded here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html). Versions with an `-rc` suffix are release candidates: commands, output, and flags may still change before `1.0.0`.

## [Unreleased]

### Changed

- `--clean` lists only projects with transcripts that can be moved. Sessions that are in use or not verified are left out, and a note says how many.
- `--clean` reports why sessions were not moved: project not verified, in use or not verified by the agent, or changed since the scan.

## [1.0.0-rc.1] - 2026-11-20

First release candidate.

### Fixed

- Windows: `--clean` now detects a transcript that was replaced by another file of the same size and modification time after the scan. The file identity was read lazily, so the check compared the new file with itself.

### Added

- `session-ledger` and `--preview`: read-only inventory of local transcripts for omp, Claude Code, and Codex CLI, grouped by agent and project.
- `--clean`: interactive cleanup that moves selected, eligible transcripts to the native Trash or Recycle Bin after confirmation. There is no permanent-delete fallback.
- `--version`: prints the version, operating system, architecture, and Go version.
- Standalone binaries for Linux (including WSL), macOS, and Windows on amd64 and arm64, with `SHA256SUMS`.

[Unreleased]: https://github.com/tomimandalaputra/session-ledger/compare/v1.0.0-rc.1...HEAD
[1.0.0-rc.1]: https://github.com/tomimandalaputra/session-ledger/releases/tag/v1.0.0-rc.1
