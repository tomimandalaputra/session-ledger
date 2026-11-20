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

## Branches and pull requests

`main` is protected: nobody pushes to it directly, and nothing merges until CI passes.

1. Branch from an up-to-date `main`. Name the branch after the commit type and topic, for example `fix/cleanup-recheck-mtime` or `docs/restore-transcript`.
2. Open a pull request against `main`. Write the title as a Conventional Commit, because the squash merge uses it as the single commit message on `main`.
3. Merge only after the required checks pass, the branch is up to date with `main`, and every review conversation is resolved. Squash is the only merge method.

A fix for a released version follows the same path. There is no `develop` or `release/*` branch: `main` stays releasable, and a patch release is a new tag.

## Repository protection (maintainer)

The rules live in [`.github/rulesets/`](.github/rulesets) so they can be reviewed and reapplied.

- `protect-main.json`: pull request required, squash merge only, linear history, no force push or deletion, branch up to date, and `test (ubuntu-latest)`, `test (ubuntu-24.04-arm)`, `test (macos-latest)`, `test (windows-latest)`, and `cross-build` must pass. No review approval is required, so a solo maintainer can merge their own pull request. Set `required_approving_review_count` to `1` once there is a second maintainer.
- `protect-release-tags.json`: `v*` tags cannot be moved or deleted once created.

Apply them with the GitHub CLI, signed in as a repository admin:

```sh
gh api -X POST repos/tomimandalaputra/session-ledger/rulesets --input .github/rulesets/protect-main.json
gh api -X POST repos/tomimandalaputra/session-ledger/rulesets --input .github/rulesets/protect-release-tags.json
```

Or use Settings → Rules → Rulesets → New ruleset → Import a ruleset. To change an existing ruleset, find its ID with `gh api repos/tomimandalaputra/session-ledger/rulesets` and send the edited file with `gh api -X PUT repos/tomimandalaputra/session-ledger/rulesets/<id> --input <file>`.

Also set these in Settings → General → Pull Requests: turn off merge commits and rebase merging, turn on squash merging with "Pull request title and description" as the default message, and turn on "Automatically delete head branches".

Notes:

- GitHub enforces rulesets on a private repository only on a paid plan (Pro for a personal account, Team for an organization). This repository is public, so they apply on the Free plan. If it goes private on the Free plan, the rules stay saved but stop being enforced.
- The ruleset has no bypass list, so administrators follow it too. If a required check is renamed in `release.yml`, update `protect-main.json` in the same change, or every pull request will wait on a check that never runs.
- `test (ubuntu-24.04-arm)` runs only while the repository is public. If the repository becomes private again, remove it from `protect-main.json`, or pull requests will wait on a job that never starts.

## Versioning

The project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

- `1.0.0-rc.N` is a release candidate and the first release is `1.0.0-rc.1`. Flags, output, and behavior may still change between candidates. Tags with a suffix are published as pre-releases.
- `1.0.0` is the first stable release. From then on, breaking changes to flags, output meant for scripts, or cleanup behavior need a major version.
- Backward-compatible features raise the minor version; fixes raise the patch version.

## Release checklist (maintainer)

1. Open a pull request titled `chore(release): vX.Y.Z` that moves the `[Unreleased]` entries in `CHANGELOG.md` under a new `## [X.Y.Z] - YYYY-MM-DD` heading and updates the compare links at the bottom.
2. Merge it once CI passes, then update your local `main`.
3. Tag the merge commit and push the tag: `git tag -a vX.Y.Z -m "vX.Y.Z" && git push origin vX.Y.Z`.
4. The workflow tests on Linux (amd64 and arm64), macOS, and Windows, builds all six binaries with `SHA256SUMS`, and publishes the release with the changelog entry as its notes. It rejects tags that are not valid semantic versions or have no changelog entry. Tags with a suffix, and versions below 1.0.0, are marked as pre-releases.
5. Download one binary from the release, check it against `SHA256SUMS`, and run `--version`.

A published tag is permanent: the Go module proxy caches it. Fix a bad release with a new patch version, not by moving the tag.
