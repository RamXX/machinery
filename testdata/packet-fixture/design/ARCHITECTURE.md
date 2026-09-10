# Deal desk architecture

## 2. Architecture Contract

```yaml
contract_version: 2
boundaries:
  - id: crm.domain
    kind: component
    element: domain
    code: [ "internal/domain/**" ]
  # the repository sits below the domain
  - id: crm.repo
    kind: component
    element: repo
    code: [ "internal/repo/**" ]
externals:
  - id: external.db
    element: db
    imports: [ "github.com/example/db" ]
dependency_rules:
  allow:
    - crm.domain -> crm.repo
    - crm.repo -> external.db
  deny:
    - "crm.repo -> crm.domain"
```

## 3. Dependency mitigation posture

| dependency | failure modes | deployment mitigation | residual behavior the FSM must handle | bound | operator signal |
|---|---|---|---|---|---|
| `db` (the one store) | locked, timeout | single writer | retry then rolledBack | retry <= 3 | lock-wait gauge |

## 6. Interface contracts

| edge | shape | errors | idempotency |
|---|---|---|---|
| `crm.domain -> crm.repo` | SaveDeal(row) | ErrLocked, ErrTimeout | keyed on deal id |
