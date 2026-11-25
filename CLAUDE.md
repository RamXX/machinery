# Conventions for agents in this repository

This branch, `experimental/v2`, is machinery v2. Nothing is released from it. v1 lives on `main`:
read v1 files with `git show main:<path>`, never check `main` out here, never change it.

## What this repository is

`machinery` is one Go command-line tool. `docs/formats.md` is the contract for every file format,
command, output line and exit code. Read the sections your task names before you write code. A
behavior the contract does not describe is not built: change the contract first.

## Layout

| Path | Holds |
|---|---|
| `cmd/machinery/` | `main.go` only. |
| `internal/cli/` | One file per command. |
| `internal/stages/` | One file per check stage. |
| `internal/<package>/` | One responsibility per package. |
| `pkg/receipt/` | The only public Go package: a helper for tests that emit receipts. |
| `e2e/` | The test suite: `testdata/script/*.txtar`, `testdata/fixtures/`. |
| `examples/crm/` | The reference project, a separate Go module with its own Makefile. |
| `docs/` | `formats.md` (the contract) and guides derived from it. |

## Commands

| Command | Does |
|---|---|
| `make` | Lists the targets. |
| `make tools` | Installs the pinned external tools into `.bin/`. |
| `make check` | Formatting, `go vet`, build. Run it before every commit. |
| `make e2e-one S=<name>` | One script. Run the scripts your change affects. |
| `make e2e` | The whole suite. Run it once, at the end, not per change. |
| `make verify` | Full verification, including the reference project (needs Docker). |

## Testing

- End-to-end only. Every test runs the real built binary against real files, real git and the
  real external tools. There are no unit tests: no `_test.go` file exists outside `e2e/` and
  `examples/*/e2e/`. No mocks and no stand-ins. Fixture data is fine.
- A test is a `testscript` file, `e2e/testdata/script/<command>_<case>.txtar`. Helpers:
  `fixture <name>` copies a fixture project into the work directory; `gitinit` makes it a git
  repository and commits; `gitcommit <message>` commits everything; `exec mrun <code> <args...>`
  runs machinery and fails unless it exits with exactly that code.
- Every behavior has a script, including the refusals: what must fail, with which message.
- Never weaken, skip or delete a test to get green. A missing tool fails; it never skips.

## Code

- Go, standard library first. Small units, one responsibility per package.
- Output is deterministic: sorted, no timestamps, no absolute paths, project-relative names.
- A check that found nothing to check is not demonstrated; it never passes on absence.
- Error text says what is wrong and what to do about it.
- No emojis and no em dashes, anywhere.

## Git

- Small commits: a short imperative subject, then what changed and why.
- Never push. Never run `git clean`, `git reset --hard` or `git checkout main`.
- The backlog lives in a local tracker outside git. Never commit tracker files, and do not put
  tracker ids or anything about other projects in code, docs or commit messages.
