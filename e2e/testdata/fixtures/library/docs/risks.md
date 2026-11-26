# Library risks

What can go wrong at the desk, and what the desk must then do.

## Principles

| Id | Principle |
|----|-----------|
| P1 | A lent copy is always accounted for. |

## Desk

| Risk | Failure | Current behavior | Required behavior |
|------|---------|------------------|-------------------|
| RISK-1 | A member with unpaid fines asks to borrow a copy. | The desk lends anyway. | The loan is refused with MEMBER_BLOCKED and stays reserved. |
| RISK-2 | The notice service does not answer. | The queue waits silently. | The queue gives up after the timeout and tries again on the next round. |

## Stock

Ref | Severity | What goes wrong | Expected
--- | --- | --- | ---
RISK-3 | high | A lent copy is lost. | The loan ends as lost and the member is charged once.
RISK-4 | low | The shelf list is out of date. | The desk recounts the shelf every week.
