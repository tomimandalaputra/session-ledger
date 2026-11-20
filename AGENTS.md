# Session Ledger

## Start every task here

Load the [antislop core](.agents/skills/antislop/SKILL.md) before planning, editing, or reviewing any change. If your agent cannot invoke skills by name, open the linked file and follow it. Use the local copy; contributors do not need a global install.

Load the [writing skill](.agents/skills/antislop-copywriting/SKILL.md) for prose, including CLI prompts, documentation, and commit messages. Use the [code skill](.agents/skills/antislop-code/SKILL.md) for implementation and tests. For interface work, also consult the [UI](.agents/skills/antislop-ui/SKILL.md) and [accessibility](.agents/skills/antislop-human/SKILL.md) skills. The [mobile layout skill](.agents/skills/antislop-layoutmobile/SKILL.md) applies to mobile interfaces. Before handing off, review what changed against the core delivery gate and the relevant skill checklist. Report what you checked; do not claim a pass for a check you did not run.

The skills come from [miqdadbadjuber/anti-slop](https://github.com/miqdadbadjuber/anti-slop/tree/1969b459f843603ba2efbbb24e8e1bf963c889b3). When updating them, review the upstream changes, keep the license notices, and update this revision link.

## Project decisions

Session Ledger is a Go CLI with a Bubble Tea interface for Linux (including WSL), macOS, and Windows. Write CLI text and repository documentation in English. Keep this file for decisions that future contributors need, not session logs, personal paths, or transcript contents.

- `session-ledger` and `--preview` show sessions across agents and projects without prompts or writes. Detecting an installed agent does not establish that we can read or clean its sessions.
- `--clean` requires explicit confirmation. Default to No. Recheck eligibility after confirmation, skip changed or unverified transcripts, and never fall back to permanent deletion. Leave project files and sibling artifacts alone.
- `inventory.go` handles discovery, `cleanup.go` checks activity and moves transcripts to native Trash, and `main.go` handles command-line arguments and the terminal interface.
- Use `go test ./...` for Go changes. Exercise preview with `go run . --preview`. For cleanup changes, use a temporary agent store with disposable transcripts. Never test deletion on a contributor's real history. A cross-build does not verify native Trash behavior.

## Versioning and releases

Follow Semantic Versioning. The first release is `0.0.1` (beta); `1.0.0` marks the first stable release. Tags are `vX.Y.Z`; the release workflow builds `main.version` from the tag, requires a matching `CHANGELOG.md` section, and marks versions below 1.0.0 as pre-releases. The module path is `github.com/tomimandalaputra/session-ledger`. The release steps are in [CONTRIBUTING.md](CONTRIBUTING.md). Skills for Claude Code are reached through the `.claude/skills` symlink to `.agents/skills`.

## Commit messages

Use Conventional Commits: `<type>(<optional scope>): <summary>`. Omit the parentheses when there is no scope. Write the summary in English, start with an imperative verb, and name the change. Choose the type by intent:

| Type | Use for |
| --- | --- |
| `feat` | New user-facing behavior |
| `fix` | A correction to existing behavior |
| `docs` | Documentation only |
| `refactor` | Code restructuring without a behavior change |
| `perf` | A performance improvement |
| `test` | Test coverage or fixtures only |
| `build` | Dependencies, compilation, or packaging |
| `ci` | Continuous integration or release workflows |
| `style` | Formatting only |
| `chore` | Repository maintenance that fits none of the above |
| `revert` | Reversal of an earlier commit; identify it in the body |

Examples: `fix(cleanup): keep transcripts modified after selection`, `docs: explain how to restore a transcript`, `chore(skills): update the anti-slop rules`. For a breaking change, put `!` before the colon and explain the migration in a `BREAKING CHANGE:` footer.
