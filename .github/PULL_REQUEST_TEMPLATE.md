## What changed and why

<!-- Link the issue this closes, for example "Closes #12". Open an issue first for anything larger than a small fix. -->

## How you checked it

<!-- Commands you ran and what you saw. Say which operating systems you could not test. -->

- [ ] `go vet ./...` and `go test ./...` pass
- [ ] `go run . --preview` runs without errors
- [ ] Cleanup changes were exercised against a temporary agent store with disposable transcripts, never real history
- [ ] `README.md` and `CHANGELOG.md` (under `[Unreleased]`) are updated for user-visible changes
- [ ] The title follows Conventional Commits, for example `fix(cleanup): keep transcripts modified after selection`
