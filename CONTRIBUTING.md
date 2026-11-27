# Contributing

This is machinery v2. v1, the 0.x releases, lives on branch `v1`.

## Ground rules

- `docs/formats.md` is the contract. Change it first, then the code, then the tests.
- Tests are end-to-end only: `testscript` files under `e2e/testdata/script/` that run the real
  built binary against real files, real git and the real external tools. No unit tests, no
  mocks. `make check` fails on a `_test.go` file outside `e2e/`.
- A missing tool fails a test. Nothing skips.
- Output is deterministic: sorted, no timestamps, no absolute paths.
- No emojis and no em dashes in code, docs or commit messages.

## Setup

```
make tools     # modelith 0.6.0, xstate 5.33.4 and the formal jars into .bin/ (needs Go, Node, Java)
make check     # gofmt, go vet, build, and machinery check on this repository
make e2e-one S=gen_shop    # one script
make e2e       # the whole suite, once, at the end
make verify    # check, e2e and the reference project (needs Docker)
```

The scripts that exercise the backlog bridge need the `bd` command on PATH. Each creates a
throwaway database in its own work directory.

## A change, start to finish

1. Find the section of `docs/formats.md` the change belongs to. If the behavior is not there,
   write it there first.
2. Write the script: `e2e/testdata/script/<command>_<case>.txtar`. Cover the refusal too: what
   must fail, with which message and exit code. Run it and see it fail for the right reason.
3. Write the code. One file per command in `internal/cli`, one file per stage in
   `internal/stages`, one responsibility per package.
4. `make check` and the scripts your change affects. Run `make e2e` once before you hand over.
5. Commit: a short imperative subject, then what changed and why.

## Releasing

Versions are numeric, `MAJOR.MINOR.PATCH`; the tag is `vMAJOR.MINOR.PATCH`. Only a release commit
changes the version, in all of these at once:

- `VERSION` in the `Makefile`, which `make build` stamps (`machinery --version`);
- `version` in `.claude-plugin/plugin.json` and `.codex-plugin/plugin.json`, and the version
  `e2e/testdata/script/plugins_manifest.txtar` expects;
- the install and clone lines in `README.md` and `docs/hooks.md`;
- `CHANGELOG.md`: the `[Unreleased]` section becomes `## [X.Y.Z] - YYYY-MM-DD`, with an empty
  `## [Unreleased]` above it.

`make verify` passes at the release commit. The tag goes on the commit `main` points to, and the
GitHub release notes are that version's section of the changelog. A new major version also
changes the module path suffix (`/v2`) in `go.mod` and in every import.

## Layout

See `CLAUDE.md`.
