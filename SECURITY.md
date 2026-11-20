# Security policy

Session Ledger reads local agent transcripts and can move them to the native Trash. A flaw that moves a transcript it should have kept, follows a path outside an agent's session store, or exposes transcript content is a security issue.

## Supported versions

The project is in release-candidate status (versions before 1.0.0). Only the latest release receives fixes.

## Report a vulnerability

Use [private vulnerability reporting](https://github.com/tomimandalaputra/session-ledger/security/advisories/new). Do not open a public issue or attach real transcripts.

Include the version (`session-ledger --version`), operating system, the agent whose sessions are affected, and steps to reproduce with disposable files. This is a solo-maintained project: expect an acknowledgement within a week, and a fix timeline once the report is confirmed.

## What the tool does not do

It has no network access, telemetry, or background service. Cleanup is not secure erasure: moved files stay recoverable from Trash until it is emptied.
