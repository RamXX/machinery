# Generated transition oracle: `deal`

Generated from `Deal.machine.json` by `machinery oracle`. DO NOT EDIT BY HAND.

## Transitions

| test id | stable id | source | trigger | guard | target | actions |
|---|---|---|---|---|---|---|
| T-DEAL-01 | DEAL-eb0c40 | Lead | on:advanceStage | guardCanAdvance | persisting | setPendingAdvance |
| T-DEAL-02 | DEAL-38ba11 | Lead | on:win | guardCanWrite | persisting | setPendingWin |
| T-DEAL-03 | DEAL-1fe825 | Lead | on:lose | guardCanWrite | persisting | setPendingLose |
