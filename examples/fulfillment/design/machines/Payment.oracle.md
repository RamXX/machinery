# Generated transition oracle: `payment`

Generated from `Payment.machine.json` by tools/oracle_gen.py. DO NOT EDIT BY HAND.
Single source of truth for the hard-TDD transition tests: one transition row is one
test case. Key tests on the STABLE id, not the row number; row numbers renumber when
the design changes, stable ids do not.

## State entry / exit actions

| state | kind | entry | exit |
|---|---|---|---|
| Pending | atomic | - | - |
| Authorized | atomic | - | - |
| Captured | atomic | - | - |
| Failed | final | - | - |
| Refunded | final | - | - |
| authorizing | atomic | - | - |
| capturing | atomic | - | - |
| refunding | atomic | - | - |
| gatewayRetry | atomic | - | - |
| gatewayResume | atomic | - | - |
| persisting | atomic | - | - |
| persistRetry | atomic | - | - |
| rolledBack | atomic | - | - |

## Transitions

| test id | stable id | source | trigger | guard | target | actions |
|---|---|---|---|---|---|---|
| T-PAYM-01 | PAYM-e44cea | Pending | on:authorize | guardAmountNonneg | authorizing | setGatewayAuthorize |
| T-PAYM-02 | PAYM-fe697d | Pending | on:authorize | - | (internal) | recordAuthorizeDenied |
| T-PAYM-03 | PAYM-1f186e | Pending | on:fail | - | persisting | setPendingFailed |
| T-PAYM-04 | PAYM-8610dc | Authorized | on:capture | - | capturing | setGatewayCapture |
| T-PAYM-05 | PAYM-a32f2f | Authorized | on:fail | - | persisting | setPendingFailed |
| T-PAYM-06 | PAYM-8835f6 | Captured | on:refund | - | refunding | setGatewayRefund |
| T-PAYM-07 | PAYM-41d625 | authorizing | after:gatewayTimeout | - | gatewayRetry | recordGatewayTimeout |
| T-PAYM-08 | PAYM-682fd2 | authorizing | onDone:gatewayAuthorize | - | persisting | setPendingAuthorized |
| T-PAYM-09 | PAYM-1a6d14 | authorizing | onError:gatewayAuthorize | isErrRetryable | gatewayRetry | recordGatewayError |
| T-PAYM-10 | PAYM-64cfd2 | authorizing | onError:gatewayAuthorize | - | persisting | recordGatewayRejected, setPendingFailed |
| T-PAYM-11 | PAYM-d6b62d | capturing | after:gatewayTimeout | - | gatewayRetry | recordGatewayTimeout |
| T-PAYM-12 | PAYM-f8577b | capturing | onDone:gatewayCapture | - | persisting | setPendingCaptured |
| T-PAYM-13 | PAYM-4971d0 | capturing | onError:gatewayCapture | isErrRetryable | gatewayRetry | recordGatewayError |
| T-PAYM-14 | PAYM-916a6f | capturing | onError:gatewayCapture | - | persisting | recordGatewayRejected, setPendingFailed |
| T-PAYM-15 | PAYM-0bf695 | refunding | after:gatewayTimeout | - | gatewayRetry | recordGatewayTimeout |
| T-PAYM-16 | PAYM-d948b7 | refunding | onDone:gatewayRefund | - | persisting | setPendingRefunded |
| T-PAYM-17 | PAYM-fcd945 | refunding | onError:gatewayRefund | isErrRetryable | gatewayRetry | recordGatewayError |
| T-PAYM-18 | PAYM-6dd4ae | refunding | onError:gatewayRefund | - | rolledBack | recordRefundFailed |
| T-PAYM-19 | PAYM-882299 | gatewayRetry | after:gatewayRetryBackoff | - | gatewayResume | incrementGatewayRetries |
| T-PAYM-20 | PAYM-086f68 | gatewayRetry | always | gatewayRetriesExhausted | rolledBack | recordGatewayExhausted |
| T-PAYM-21 | PAYM-47dcf2 | gatewayResume | always | gatewayForAuthorize | authorizing | - |
| T-PAYM-22 | PAYM-744d43 | gatewayResume | always | gatewayForCapture | capturing | - |
| T-PAYM-23 | PAYM-84cdf1 | gatewayResume | always | gatewayForRefund | refunding | - |
| T-PAYM-24 | PAYM-da0b09 | persisting | after:persistTimeout | - | rolledBack | recordTimeout |
| T-PAYM-25 | PAYM-636129 | persisting | onDone:persistPayment | pendingIsAuthorized | Authorized | commitStatus |
| T-PAYM-26 | PAYM-57e9ea | persisting | onDone:persistPayment | pendingIsCaptured | Captured | commitStatus |
| T-PAYM-27 | PAYM-aa8a9f | persisting | onDone:persistPayment | pendingIsRefunded | Refunded | commitStatus |
| T-PAYM-28 | PAYM-c85fef | persisting | onDone:persistPayment | pendingIsFailed | Failed | commitStatus |
| T-PAYM-29 | PAYM-a50d14 | persisting | onDone:persistPayment | - | rolledBack | recordRoutingError |
| T-PAYM-30 | PAYM-54ed9b | persisting | onError:persistPayment | isErrUnavailable | persistRetry | recordError |
| T-PAYM-31 | PAYM-e1e392 | persisting | onError:persistPayment | isErrConflict | persistRetry | recordConflict |
| T-PAYM-32 | PAYM-869145 | persisting | onError:persistPayment | - | rolledBack | recordUnknownError |
| T-PAYM-33 | PAYM-8a344a | persistRetry | after:persistRetryBackoff | - | persisting | incrementRetries |
| T-PAYM-34 | PAYM-825191 | persistRetry | always | retriesExhausted | rolledBack | recordRetriesExhausted |
| T-PAYM-35 | PAYM-384d1b | rolledBack | always | priorIsPending | Pending | - |
| T-PAYM-36 | PAYM-50a826 | rolledBack | always | priorIsAuthorized | Authorized | - |
| T-PAYM-37 | PAYM-e19ba2 | rolledBack | always | priorIsCaptured | Captured | - |

Total transitions (test cases): 37
