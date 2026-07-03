# CLI Port Contract (historical)

Historical: this froze the observable CLI behavior at the Python-to-Go migration (tag
`pre-go-migration`). The Python names below are the provenance of each subcommand; the Python
tools are deleted and the golden corpus under `testdata/golden` is now the regression net.
Behavior may evolve past this document; changes land in the golden corpus.

## Tools and their Go subcommands

| Python tool | Go subcommand | Args |
|-------------|---------------|------|
| `machine_lint.py` | `lint` | `<machines-dir>` |
| `oracle_gen.py` | `oracle` | `<machines-dir>` |
| `tla_gen.py` | `tla` | `<machine.json> [out-dir]` |
| `refine_gen.py` | `refine` | `<machine.json> <semantics.yaml> [out-dir]` |
| `compose_gen.py` | `compose` | `<composition.yaml> <coordinator.machine.json> [out-dir]` |
| `machinery_check.py` | `check` | `<design-dir> [--impl <code-dir>] [--gate g2,g3,gx,g4]` |
| `verify_formal.sh` | `verify-formal` | `<design-dir>` |
| (new) `doctor` | `doctor` | (none) |
| (new) `preflight` | `preflight` | (none) |
| (new) `ir-dump` | `ir-dump` (hidden) | `<machine.json>` |

## Output format rules

- **lint**: per-file `== <base>: N states ==` header; `  ERROR  `, `  DRIFT  `,
  `  warn   ` lines; `  ok` when clean; trailing blank line + summary
  `N error/drift finding(s) across M machine(s)`. Exit 1 if any ERROR/DRIFT, else 0.
  `ERROR  no *.machine.json under <dir>: nothing to lint is a failure, not a pass` + exit 1 if empty.
- **oracle**: writes `<name>.oracle.md` per machine; stdout
  `generated <name>.oracle.md  (<N> transition rows)`. Exit 1 if no machines
  (`no *.machine.json under <dir>`). Exit with `oracle_gen: <err>` message on bad machine.
- **tla**: writes `<Mid>.tla` and `<Mid>.cfg`; stdout
  `wrote <Mid>.tla and <Mid>.cfg to <outdir>`. Hard errors exit with
  `tla_gen: <Mid>: <reason>` (nested, type, multi-after, no-target).
- **refine**: stdout `refine_gen: reconciled <Mid> against the machine: ...` then
  `generated N files for <Mid> (<pattern>)`. Hard errors exit with
  `refine_gen: RECONCILIATION FAILED: <msg>` or `refine_gen: unsupported pattern <p>`.
- **compose**: stdout `compose_gen: validated <Name> against <machine>: ...` then
  `generated <Name>.tla + <Name>.cfg`. Hard errors exit with
  `compose_gen: VALIDATION FAILED: <msg>`.
- **check**: per-gate `== <title> ==` with `  ERROR  / DRIFT / warn / note` lines,
  `  checked: <counts>` line, `  ok` when clean; trailing
  `\nN blocking (ERROR/DRIFT) finding(s)`. Exit 1 if fail, else 0.
- **verify-formal**: per-spec `  PASS  <name>` / `  FAIL  <name>`; trailing
  `<pass> passed, <fail> failed`; exit 1 if any fail.

## Exit codes

0 = clean; non-zero on any ERROR/DRIFT finding, missing input, or parse failure.
Specific `sys.exit` messages are reproduced verbatim (prefixed `tool:`).
