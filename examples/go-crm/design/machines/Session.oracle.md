# Generated transition oracle: `session`

Generated from `Session.machine.json` by tools/oracle_gen.py. DO NOT EDIT BY HAND.
Single source of truth for the hard-TDD transition tests: one transition row is one test case.

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

| test id | source | trigger | guard | target | actions |
|---|---|---|---|---|---|
| T-SESS-01 | Anonymous | on:login | - | Authenticating | setCredentials |
| T-SESS-02 | Anonymous | on:resume | - | Resolving | - |
| T-SESS-03 | Anonymous | on:logout | - | (internal) | recordNoSessionToLogout |
| T-SESS-04 | Anonymous | on:useSession | - | (internal) | recordNoActiveSession |
| T-SESS-05 | Authenticating | after:verifyTimeout | - | SessionUnavailable | recordTimeout |
| T-SESS-06 | Authenticating | onDone:verifyCredentials | guardUserDisabled | AuthDenied | recordDisabled |
| T-SESS-07 | Authenticating | onDone:verifyCredentials | - | WritingSession | captureUser |
| T-SESS-08 | Authenticating | onError:verifyCredentials | isErrBadCredentials | AuthFailed | recordBadCredentials |
| T-SESS-09 | Authenticating | onError:verifyCredentials | isErrDisabled | AuthDenied | recordDisabled |
| T-SESS-10 | Authenticating | onError:verifyCredentials | isErrLocked | VerifyRetry | recordError |
| T-SESS-11 | Authenticating | onError:verifyCredentials | - | SessionUnavailable | recordVerifyError |
| T-SESS-12 | VerifyRetry | after:verifyRetryBackoff | - | Authenticating | incrementRetries |
| T-SESS-13 | VerifyRetry | always | retriesExhausted | SessionUnavailable | recordRetriesExhausted |
| T-SESS-14 | WritingSession | after:fileIoTimeout | - | SessionUnavailable | recordTimeout |
| T-SESS-15 | WritingSession | onDone:writeSessionFile | - | Active | - |
| T-SESS-16 | WritingSession | onError:writeSessionFile | - | SessionUnavailable | recordFileError |
| T-SESS-17 | Resolving | after:fileIoTimeout | - | SessionUnavailable | recordTimeout |
| T-SESS-18 | Resolving | onDone:readSessionFile | guardSessionExpired | Expired | recordExpired |
| T-SESS-19 | Resolving | onDone:readSessionFile | - | CheckingUser | captureToken |
| T-SESS-20 | Resolving | onError:readSessionFile | isErrNoSession | Anonymous | recordNoSession |
| T-SESS-21 | Resolving | onError:readSessionFile | isErrExpired | Expired | recordExpired |
| T-SESS-22 | Resolving | onError:readSessionFile | - | SessionUnavailable | recordFileError |
| T-SESS-23 | CheckingUser | after:loadUserTimeout | - | SessionUnavailable | recordTimeout |
| T-SESS-24 | CheckingUser | onDone:loadUser | guardSessionUserActive | Active | captureUser |
| T-SESS-25 | CheckingUser | onDone:loadUser | - | Invalidated | recordUserNotActive |
| T-SESS-26 | CheckingUser | onError:loadUser | isErrLocked | VerifyRetry | recordError |
| T-SESS-27 | CheckingUser | onError:loadUser | isErrNotFound | Invalidated | recordUserMissing |
| T-SESS-28 | CheckingUser | onError:loadUser | - | SessionUnavailable | recordLoadError |
| T-SESS-29 | Active | on:logout | - | LoggingOut | - |
| T-SESS-30 | Active | on:useSession | - | (internal) | recordSessionUsed |
| T-SESS-31 | Active | on:login | - | (internal) | recordAlreadyActive |
| T-SESS-32 | Active | on:resume | - | (internal) | recordAlreadyResolved |
| T-SESS-33 | Active | after:sessionTTL | - | Expired | recordExpired |
| T-SESS-34 | LoggingOut | after:fileIoTimeout | - | LoggedOut | recordLogoutWarning |
| T-SESS-35 | LoggingOut | onDone:clearSessionFile | - | LoggedOut | - |
| T-SESS-36 | LoggingOut | onError:clearSessionFile | - | LoggedOut | recordLogoutWarning |
| T-SESS-37 | Expired | on:login | - | Authenticating | setCredentials |
| T-SESS-38 | Expired | on:resume | - | (internal) | recordExpiredNeedsLogin |
| T-SESS-39 | Expired | on:logout | - | (internal) | recordNoSessionToLogout |
| T-SESS-40 | Expired | on:useSession | - | (internal) | recordSessionExpired |
| T-SESS-41 | LoggedOut | on:login | - | Authenticating | setCredentials |
| T-SESS-42 | LoggedOut | on:resume | - | (internal) | recordNoSession |
| T-SESS-43 | LoggedOut | on:logout | - | (internal) | recordNoSessionToLogout |
| T-SESS-44 | LoggedOut | on:useSession | - | (internal) | recordNoActiveSession |
| T-SESS-45 | AuthFailed | on:login | - | Authenticating | setCredentials |
| T-SESS-46 | AuthFailed | on:resume | - | (internal) | recordNoSession |
| T-SESS-47 | AuthFailed | on:logout | - | (internal) | recordNoSessionToLogout |
| T-SESS-48 | AuthFailed | on:useSession | - | (internal) | recordNoActiveSession |
| T-SESS-49 | AuthDenied | on:login | - | Authenticating | setCredentials |
| T-SESS-50 | AuthDenied | on:resume | - | (internal) | recordNoSession |
| T-SESS-51 | AuthDenied | on:logout | - | (internal) | recordNoSessionToLogout |
| T-SESS-52 | AuthDenied | on:useSession | - | (internal) | recordNoActiveSession |
| T-SESS-53 | Invalidated | on:login | - | Authenticating | setCredentials |
| T-SESS-54 | Invalidated | on:resume | - | (internal) | recordNoSession |
| T-SESS-55 | Invalidated | on:logout | - | (internal) | recordNoSessionToLogout |
| T-SESS-56 | Invalidated | on:useSession | - | (internal) | recordNoActiveSession |
| T-SESS-57 | SessionUnavailable | on:login | - | Authenticating | setCredentials |
| T-SESS-58 | SessionUnavailable | on:resume | - | Resolving | - |
| T-SESS-59 | SessionUnavailable | on:logout | - | (internal) | recordNoSessionToLogout |
| T-SESS-60 | SessionUnavailable | on:useSession | - | (internal) | recordNoActiveSession |

Total transitions (test cases): 60
