# Contributing

This branch is machinery v2, experimental. v1 is on `main`.

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
make check     # gofmt, go vet, build
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

## Layout

See `CLAUDE.md`.
