# Generated transition oracle: `session`

Generated from `Session.machine.json` by `machinery oracle`. DO NOT EDIT BY HAND.
<!-- machinery-version: v0.3.4-dev -->
Single source of truth for the hard-TDD transition tests: one transition row is one
test case. Key tests on the STABLE id, not the row number; row numbers renumber when
the design changes, stable ids do not.

## State entry / exit actions

| state | kind | entry | exit |
|---|---|---|---|
| Anonymous | atomic | - | - |
| Authenticating | atomic | - | - |
| VerifyRetry | atomic | - | - |
| WritingSession | atomic | - | - |
| Resolving | atomic | - | - |
| CheckingUser | atomic | - | - |
| Active | atomic | - | - |
| LoggingOut | atomic | - | - |
| Expired | atomic | - | - |
| LoggedOut | atomic | - | - |
| AuthFailed | atomic | - | - |
| AuthDenied | atomic | - | - |
| Invalidated | atomic | - | - |
| SessionUnavailable | atomic | - | - |

## Transitions

| test id | stable id | source | trigger | guard | target | actions |
|---|---|---|---|---|---|---|
| T-SESS-01 | SESS-ee5c17 | Anonymous | on:login | - | Authenticating | setCredentials |
| T-SESS-02 | SESS-f3cc5e | Anonymous | on:resume | - | Resolving | - |
| T-SESS-03 | SESS-27e1b6 | Anonymous | on:logout | - | (internal) | recordNoSessionToLogout |
| T-SESS-04 | SESS-fd60bd | Anonymous | on:useSession | - | (internal) | recordNoActiveSession |
| T-SESS-05 | SESS-63844a | Authenticating | after:verifyTimeout | - | SessionUnavailable | recordTimeout |
| T-SESS-06 | SESS-abcf53 | Authenticating | onDone:verifyCredentials | guardUserDisabled | AuthDenied | recordDisabled |
| T-SESS-07 | SESS-c78e53 | Authenticating | onDone:verifyCredentials | - | WritingSession | captureUser |
| T-SESS-08 | SESS-fbf094 | Authenticating | onError:verifyCredentials | isErrBadCredentials | AuthFailed | recordBadCredentials |
| T-SESS-09 | SESS-e12d58 | Authenticating | onError:verifyCredentials | isErrDisabled | AuthDenied | recordDisabled |
| T-SESS-10 | SESS-2dee87 | Authenticating | onError:verifyCredentials | isErrLocked | VerifyRetry | recordError |
| T-SESS-11 | SESS-9c998d | Authenticating | onError:verifyCredentials | - | SessionUnavailable | recordVerifyError |
| T-SESS-12 | SESS-316622 | VerifyRetry | after:verifyRetryBackoff | - | Authenticating | incrementRetries |
| T-SESS-13 | SESS-aaa861 | VerifyRetry | always | retriesExhausted | SessionUnavailable | recordRetriesExhausted |
| T-SESS-14 | SESS-402e07 | WritingSession | after:fileIoTimeout | - | SessionUnavailable | recordTimeout |
| T-SESS-15 | SESS-b95638 | WritingSession | onDone:writeSessionFile | - | Active | - |
| T-SESS-16 | SESS-7b8023 | WritingSession | onError:writeSessionFile | - | SessionUnavailable | recordFileError |
| T-SESS-17 | SESS-c552bf | Resolving | after:fileIoTimeout | - | SessionUnavailable | recordTimeout |
| T-SESS-18 | SESS-e01d44 | Resolving | onDone:readSessionFile | guardSessionExpired | Expired | recordExpired |
| T-SESS-19 | SESS-e6484d | Resolving | onDone:readSessionFile | - | CheckingUser | captureToken |
| T-SESS-20 | SESS-4f7245 | Resolving | onError:readSessionFile | isErrNoSession | Anonymous | recordNoSession |
| T-SESS-21 | SESS-dddfd5 | Resolving | onError:readSessionFile | isErrExpired | Expired | recordExpired |
| T-SESS-22 | SESS-12a601 | Resolving | onError:readSessionFile | - | SessionUnavailable | recordFileError |
| T-SESS-23 | SESS-aafc2f | CheckingUser | after:loadUserTimeout | - | SessionUnavailable | recordTimeout |
| T-SESS-24 | SESS-85613f | CheckingUser | onDone:loadUser | guardSessionUserActive | Active | captureUser |
| T-SESS-25 | SESS-55a08c | CheckingUser | onDone:loadUser | - | Invalidated | recordUserNotActive |
| T-SESS-26 | SESS-10b95d | CheckingUser | onError:loadUser | isErrLocked | VerifyRetry | recordError |
| T-SESS-27 | SESS-54c2ea | CheckingUser | onError:loadUser | isErrNotFound | Invalidated | recordUserMissing |
| T-SESS-28 | SESS-a337ab | CheckingUser | onError:loadUser | - | SessionUnavailable | recordLoadError |
| T-SESS-29 | SESS-69da3c | Active | on:logout | - | LoggingOut | - |
| T-SESS-30 | SESS-448752 | Active | on:useSession | - | (internal) | recordSessionUsed |
| T-SESS-31 | SESS-fd231f | Active | on:login | - | (internal) | recordAlreadyActive |
| T-SESS-32 | SESS-7fed3c | Active | on:resume | - | (internal) | recordAlreadyResolved |
| T-SESS-33 | SESS-87f3f3 | Active | after:sessionTTL | - | Expired | recordExpired |
| T-SESS-34 | SESS-927687 | LoggingOut | after:fileIoTimeout | - | LoggedOut | recordLogoutWarning |
| T-SESS-35 | SESS-706156 | LoggingOut | onDone:clearSessionFile | - | LoggedOut | - |
| T-SESS-36 | SESS-51ded9 | LoggingOut | onError:clearSessionFile | - | LoggedOut | recordLogoutWarning |
| T-SESS-37 | SESS-90aa72 | Expired | on:login | - | Authenticating | setCredentials |
| T-SESS-38 | SESS-5e1b28 | Expired | on:resume | - | (internal) | recordExpiredNeedsLogin |
| T-SESS-39 | SESS-2587f3 | Expired | on:logout | - | (internal) | recordNoSessionToLogout |
| T-SESS-40 | SESS-25fc00 | Expired | on:useSession | - | (internal) | recordSessionExpired |
| T-SESS-41 | SESS-bb2221 | LoggedOut | on:login | - | Authenticating | setCredentials |
| T-SESS-42 | SESS-3cee4a | LoggedOut | on:resume | - | (internal) | recordNoSession |
| T-SESS-43 | SESS-656e49 | LoggedOut | on:logout | - | (internal) | recordNoSessionToLogout |
| T-SESS-44 | SESS-f09ff1 | LoggedOut | on:useSession | - | (internal) | recordNoActiveSession |
| T-SESS-45 | SESS-61d379 | AuthFailed | on:login | - | Authenticating | setCredentials |
| T-SESS-46 | SESS-f821b7 | AuthFailed | on:resume | - | (internal) | recordNoSession |
| T-SESS-47 | SESS-65b326 | AuthFailed | on:logout | - | (internal) | recordNoSessionToLogout |
| T-SESS-48 | SESS-aca1a6 | AuthFailed | on:useSession | - | (internal) | recordNoActiveSession |
| T-SESS-49 | SESS-5a47c2 | AuthDenied | on:login | - | Authenticating | setCredentials |
| T-SESS-50 | SESS-22bc20 | AuthDenied | on:resume | - | (internal) | recordNoSession |
| T-SESS-51 | SESS-3e386e | AuthDenied | on:logout | - | (internal) | recordNoSessionToLogout |
| T-SESS-52 | SESS-75e8c1 | AuthDenied | on:useSession | - | (internal) | recordNoActiveSession |
| T-SESS-53 | SESS-f59ab7 | Invalidated | on:login | - | Authenticating | setCredentials |
| T-SESS-54 | SESS-134dea | Invalidated | on:resume | - | (internal) | recordNoSession |
| T-SESS-55 | SESS-de6aa7 | Invalidated | on:logout | - | (internal) | recordNoSessionToLogout |
| T-SESS-56 | SESS-19fd5c | Invalidated | on:useSession | - | (internal) | recordNoActiveSession |
| T-SESS-57 | SESS-e7bef7 | SessionUnavailable | on:login | - | Authenticating | setCredentials |
| T-SESS-58 | SESS-0488b7 | SessionUnavailable | on:resume | - | Resolving | - |
| T-SESS-59 | SESS-f6a536 | SessionUnavailable | on:logout | - | (internal) | recordNoSessionToLogout |
| T-SESS-60 | SESS-b830a0 | SessionUnavailable | on:useSession | - | (internal) | recordNoActiveSession |

Total transitions (test cases): 60
