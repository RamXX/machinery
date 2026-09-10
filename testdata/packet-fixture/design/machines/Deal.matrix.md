# Deal machine - contract and failure catalog

## (a) Named-unit contract table

| name | kind | signature | pre / post | maps to |
|---|---|---|---|---|
| `guardCanAdvance` | guard | `(ctx,evt) -> bool` | true iff the next stage is forward | inv `deal-stage-forward` |
| `guardCanWrite` | guard | `(ctx,evt) -> bool` | true iff the actor may write | inv `rbac-write-scope` |
