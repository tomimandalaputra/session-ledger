# Contributing

Bug reports, fixes, and support for more agents are welcome. Read [AGENTS.md](AGENTS.md) for the project decisions; they apply to people as well as coding agents.

## Report a bug or ask for a feature

Open an [issue](https://github.com/tomimandalaputra/session-ledger/issues/new/choose) and fill in the template. Include the output of `session-ledger --version`. Never attach transcripts, and redact project paths you do not want public. Report security problems privately, as described in [SECURITY.md](SECURITY.md).

For anything larger than a small fix, open an issue before writing code so the approach can be agreed first.

## Set up

You need Go 1.24 or newer.

```sh
git clone https://github.com/tomimandalaputra/session-ledger.git
cd session-ledger
go vet ./...
go test ./...
go run . --preview
```

## Rules for changes

- Keep the safety boundaries in the README: preview never writes, cleanup needs confirmation, and there is no permanent-delete fallback.
- Never test deletion against real agent history. Point the agent's environment variable (`PI_CODING_AGENT_DIR`, `CLAUDE_CONFIG_DIR`, `CODEX_HOME`) at a temporary directory with disposable transcripts. See `inventory_test.go` for the pattern.
- Add a test for behavior you change. Tests must pass on Linux, macOS, and Windows; CI runs all three.
- A cross-build does not prove native Trash behavior. Say which platforms you ran the change on.
- Run `gofmt -l .` (it should print nothing) and `go mod tidy` before pushing.
- Add a line under `[Unreleased]` in [CHANGELOG.md](CHANGELOG.md) for every user-visible change.
- Commit messages use [Conventional Commits](https://www.conventionalcommits.org/), for example `fix(cleanup): keep transcripts modified after selection`. The full type list is in AGENTS.md.

Open the pull request against `main`. CI must pass before merge.

## Versioning

The project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

- `0.x.y` is beta. Flags, output, and behavior may change in a minor version. The first release is `0.0.1`.
- `1.0.0` is the first stable release. From then on, breaking changes to flags, output meant for scripts, or cleanup behavior need a major version.
- Backward-compatible features raise the minor version; fixes raise the patch version.
- A pre-release candidate uses a suffix, for example `1.0.0-rc.1`.

## Release checklist (maintainer)

1. Move the `[Unreleased]` entries in `CHANGELOG.md` under a new `## [X.Y.Z] - YYYY-MM-DD` heading and update the compare links at the bottom.
2. Commit that change to `main` and wait for CI to pass.
3. Tag and push: `git tag -a vX.Y.Z -m "vX.Y.Z" && git push origin vX.Y.Z`.
4. The workflow tests on Linux (amd64 and arm64), macOS, and Windows, builds all six binaries with `SHA256SUMS`, and publishes the release with the changelog entry as its notes. It rejects tags that are not valid semantic versions or have no changelog entry. Versions below 1.0.0 and tags with a suffix are marked as pre-releases.
5. Download one binary from the release, check it against `SHA256SUMS`, and run `--version`.

A published tag is permanent: the Go module proxy caches it. Fix a bad release with a new patch version, not by moving the tag.
