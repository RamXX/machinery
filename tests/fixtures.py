"""A complete minimal synthetic design + impl for machinery_check tests.

Every artifact is consistent by construction; each test mutates one thing and
asserts the gate catches it. This encodes the review's vacuity experiments as
permanent regressions.
"""
import json
import os

from conftest import minimal_machine

MODELITH = """\
kind: modelith
version: 1
title: Widget
enums:
  WidgetStatus:
    values:
      - name: Draft
      - name: Published
entities:
  Widget:
    attributes:
      - name: status
        type: WidgetStatus
    actions:
      - name: publish
    invariants:
      - id: widget-owned
invariants: []
scenarios: []
"""

WORKSPACE_DSL = """\
workspace "Widget" "Test system." {
  model {
    user = person "User" "Uses it."
    sys = softwareSystem "Widget System" "The system." {
      app      = component "App" "Application logic." "Go"
      storelib = component "Store" "Persistence." "Go"
      db       = container "DB" "The database." "SQLite" "Database"
    }
    user -> app "Uses"
    app -> storelib "Persists"
    storelib -> db "SQL"
  }
}
"""

ARCHITECTURE = """\
# Architecture: Widget

## 1. Shape

One binary.

## 2. Deployment example (not the contract)

```yaml
replicas: 3
```

## 4. Architecture Contract

```yaml
contract_version: 2
boundaries:
  - id: widget.app
    kind: component
    element: app
    code: [ "internal/app/**" ]
  - id: widget.store
    kind: component
    element: storelib
    code: [ "internal/store/**" ]
    exposes: [ "internal/store/store.go" ]
externals:
  - id: external.db
    element: db
    imports: [ "example.com/dbdriver" ]
ignore: [ "internal/scaffold/**" ]
dependency_rules:
  allow:
    - widget.app -> widget.store
    - widget.store -> external.db
  deny:
    - "widget.app -> external.db"
```

## 6. Dependency mitigation posture

| dependency | failure modes | mitigation | residual | bound |
|---|---|---|---|---|
| `db` | unavailable, corrupt | retry, backup | surface after retries | retry <= 3 |

## 7. Persistence and placement

| component | placement | persistence | concurrency |
|---|---|---|---|
| `Widget` | in-process | db row | single writer |
"""

MATRIX = """\
# Widget machine - contract and oracle

## (a) Named-unit contract table

| name | kind | signature | pre / post | maps to |
|---|---|---|---|---|
| `saveWidget` | actor | `(input) -> row \\| err` | atomic persist | `db` |
| `guardCanPublish` | guard | `(ctx,evt) -> bool` | actor may publish | inv `widget-owned` |
| `setPending` | action | `(ctx) -> ctx` | stash pending | - |
| `recordDenied` | action | `(ctx) -> ctx` | rejection reason | surfaces `widget-owned` |
| `commit` | action | `(ctx) -> ctx` | commit status | - |
| `recordError` / `recordTimeout` | action | `(ctx) -> ctx` | classify error | - |

## (c) Transition matrix

| # | source | event / after / always | guard | target | actions |
|---|---|---|---|---|---|
| 1 | Draft | publish | guardCanPublish | persisting | setPending |
| 2 | Draft | publish | !guardCanPublish | Draft (internal) | recordDenied |
| 3 | persisting | invoke onDone | - | Published | commit |
| 4 | persisting | invoke onError | - | Draft | recordError |
| 5 | persisting | after persistTimeout | - | Draft | recordTimeout |
"""

BUILD = """\
# BUILD

## Traceability

| invariant | enforced by |
|---|---|
| widget-owned | guardCanPublish |
"""

GO_MOD = "module widget\n\ngo 1.22\n"

APP_GO = """\
package app

import (
\t"fmt"

\t"widget/internal/store"
)

func Run() { fmt.Println(store.Open()) }
"""

STORE_GO = """\
package store

import "example.com/dbdriver"

func Open() string { return dbdriver.Name }
"""


def write_design(root):
    d = os.path.join(root, "design")
    os.makedirs(os.path.join(d, "machines"), exist_ok=True)
    with open(os.path.join(d, "widget.modelith.yaml"), "w", encoding="utf-8") as f:
        f.write(MODELITH)
    with open(os.path.join(d, "workspace.dsl"), "w", encoding="utf-8") as f:
        f.write(WORKSPACE_DSL)
    with open(os.path.join(d, "ARCHITECTURE.md"), "w", encoding="utf-8") as f:
        f.write(ARCHITECTURE)
    with open(os.path.join(d, "BUILD.md"), "w", encoding="utf-8") as f:
        f.write(BUILD)
    m = minimal_machine()
    mp = os.path.join(d, "machines", "Widget.machine.json")
    with open(mp, "w", encoding="utf-8") as f:
        json.dump(m, f, indent=1)
    with open(os.path.join(d, "machines", "Widget.matrix.md"), "w", encoding="utf-8") as f:
        f.write(MATRIX)
    import oracle_gen
    with open(os.path.join(d, "machines", "Widget.oracle.md"), "w", encoding="utf-8") as f:
        f.write(oracle_gen.generate(mp))
    return d


def write_impl(root):
    impl = os.path.join(root, "impl")
    os.makedirs(os.path.join(impl, "internal", "app"), exist_ok=True)
    os.makedirs(os.path.join(impl, "internal", "store"), exist_ok=True)
    with open(os.path.join(impl, "go.mod"), "w", encoding="utf-8") as f:
        f.write(GO_MOD)
    with open(os.path.join(impl, "internal", "app", "app.go"), "w", encoding="utf-8") as f:
        f.write(APP_GO)
    with open(os.path.join(impl, "internal", "store", "store.go"), "w", encoding="utf-8") as f:
        f.write(STORE_GO)
    return impl
