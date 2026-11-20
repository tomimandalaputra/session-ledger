# Session Ledger

A local-first terminal inventory for coding-agent sessions. Run `session-ledger` to see which projects have transcripts and which sessions are protected. No login, telemetry, or background service. Cleanup is opt-in.

## Status

Release candidate. The project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html); until `1.0.0`, flags and output may change between candidates. See the [changelog](CHANGELOG.md).

## Install

Session Ledger is a single static binary. Pick one route.

**Release binary** (no Go needed). Download the file for your system from the [latest release](https://github.com/tomimandalaputra/session-ledger/releases/latest):

| System | amd64 (x86-64) | arm64 |
|---|---|---|
| Linux, including WSL | `session-ledger-linux-amd64` | `session-ledger-linux-arm64` |
| macOS | `session-ledger-darwin-amd64` (Intel) | `session-ledger-darwin-arm64` (Apple silicon) |
| Windows | `session-ledger-windows-amd64.exe` | `session-ledger-windows-arm64.exe` |

Download `SHA256SUMS` from the same release and verify the file before running it.

```sh
# Linux
sha256sum --check --ignore-missing SHA256SUMS
# macOS
shasum -a 256 --check --ignore-missing SHA256SUMS
```

```powershell
# Windows: compare the output with the matching line in SHA256SUMS
Get-FileHash .\session-ledger-windows-amd64.exe -Algorithm SHA256
```

Rename the binary to `session-ledger` (`session-ledger.exe` on Windows) and put it in a directory on your PATH. On Linux and macOS, run `chmod +x session-ledger`. On macOS, a downloaded binary is not notarized, so Gatekeeper may block it; installing with `go install` avoids that, or run `xattr -d com.apple.quarantine session-ledger` if you trust the download.

Use the Linux build inside WSL for agents running in WSL. It does not scan Windows profiles; run the Windows build for those.

**Go install** (Go 1.24 or newer):

```sh
go install github.com/tomimandalaputra/session-ledger@latest
```

The binary lands in `$(go env GOPATH)/bin`, or `GOBIN` if set. Put that directory on your PATH.

Check the install with `session-ledger --version`. To uninstall, delete the binary. Session Ledger keeps no configuration or state.

## Use

```sh
session-ledger            # Global, read-only inventory; no prompts
session-ledger --preview  # Same output, suitable for piping
session-ledger --clean    # Interactive cleanup with confirmation
session-ledger --version  # Version, OS, architecture, Go version
```

The preview groups local transcripts by agent and project. **CAN MOVE** means the session passed activity checks at preview time, not that Trash is available. **KEEP** includes busy clients, unknown projects, and failed checks. Only project paths and counts are displayed, not conversations. Paths may still be sensitive when sharing output.

In cleanup, choose an agent, then one or more projects. Use ↑/↓ to move, Space to toggle projects, Enter or → to continue, ← to go back, and Esc to cancel. With no checkmarks, Enter selects the highlighted row. Confirmation defaults to **No**. Session Ledger checks activity again after confirmation and skips transcripts changed since discovery.

| Inventory & cleanup | Recognized store under the current user profile |
|---|---|
| omp | `~/.omp/agent/sessions`, respects `PI_CODING_AGENT_DIR` |
| Claude Code | `~/.claude/projects`, respects `CLAUDE_CONFIG_DIR` |
| Codex CLI | `~/.codex/sessions` and `~/.codex/archived_sessions`, respects `CODEX_HOME` |

OpenCode, Gemini CLI, GitHub Copilot CLI, Antigravity, Cursor Agent, and Aider are **detection only** (known command or directory). Their sessions are not counted or moved. Only recognized local JSONL layouts are inventoried; this is not a cloud history manager.

## Cleanup boundaries

- omp requires a working `omp session lease --json` check. For Claude and Codex, close the agent first. Process detection is conservative, not an exclusive lock: do not start or resume clients during cleanup.
- Only the selected transcript files move. Sibling artifacts, indexes, configuration, credentials, project files, and unverified records are left alone. Cleanup is not secure erasure.
- Linux/WSL uses `gio trash` (Ubuntu: `libglib2.0-bin`); macOS uses Foundation via `osascript`; Windows requests a move to the Shell Recycle Bin through Windows PowerShell. Missing helpers, unsupported filesystems, or corporate policy can prevent cleanup. No permanent-delete fallback. Restore through the native Trash/Recycle Bin (`gio trash --list` on Linux).
- Moves are per file, not transactional. If Trash reports a failure, the batch stops and reports how many transcripts it already moved.

## Troubleshooting

- **No sessions listed.** The agent may store sessions elsewhere. Check the table above and the environment variable for your agent. Detection-only agents are never counted.
- **Everything shows KEEP.** An agent is probably running, or the project path could not be verified. Close the agent and run the preview again.
- **Cleanup fails on Linux.** Install `gio` (`libglib2.0-bin` on Ubuntu and Debian) and check that `gio trash` works for a test file.
- **Cleanup fails on Windows.** Cleanup calls Windows PowerShell. Check that `powershell.exe` runs and that your policy allows moves to the Recycle Bin.
- **`interactive cleanup requires a terminal`.** `--clean` needs an interactive terminal on both input and output. Run it directly, not through a pipe.

Still stuck? Open a [bug report](https://github.com/tomimandalaputra/session-ledger/issues/new/choose) with the `--version` output.

## Contributing

Bug reports, feature requests, and pull requests are welcome. Read [CONTRIBUTING.md](CONTRIBUTING.md) first. Report security problems privately, following [SECURITY.md](SECURITY.md). Participation is covered by the [code of conduct](CODE_OF_CONDUCT.md).

## Development

Run `go vet ./...`, `go test ./...`, and `go run . --preview`. The GitHub Actions workflow runs these on Linux (amd64 and arm64), macOS, and Windows, and cross-builds all six release targets. Pushing a `v*` tag publishes a release only after those jobs pass. The release process and versioning rules are in [CONTRIBUTING.md](CONTRIBUTING.md#release-checklist-maintainer).

## License

[MIT](LICENSE). The files under `.agents/skills` are third-party coding-agent rules with their own [license notice](.agents/skills/LICENSE-antislop).
