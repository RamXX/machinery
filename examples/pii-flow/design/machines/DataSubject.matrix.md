# DataSubject machine: named-unit contracts and failure catalog

## (a) Named-unit contract table

| name | kind | signature | pre / post | maps to | test type | fixture |
|---|---|---|---|---|---|---|
| `recordErasure` | action | `(ctx) -> ctx` | on erase, destroy the subject's stored attributes and write the erasure evidence in the same transaction as the status flip; Erased is final, so processing after erasure is structurally impossible | inv `subject-erased-terminal` (structural) | unit | none |

## (b) Failure catalog

| failure | detection | transition | recovery | bounding mitigation |
|---|---|---|---|---|
| duplicate `erase` delivery | subject already Erased | none (final state accepts nothing) | drop and log | idempotent by state |
| evidence write fails | write error | none (command rejected, caller retries) | retry with backoff | retry <= 3 |
