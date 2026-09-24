# Session machine - contract, failure catalog, and transition oracle

Component: `crm.session`. Machine: `Session.machine.json`.
Placement (ARCHITECTURE.md 7): in-process during a command; token on disk at `~/.crm/session` (user id + expiry, HMAC-signed). Concurrency: last write wins; single local user.

Session is not a Modelith entity; it is the operational credential (glossary + ARCHITECTURE.md 3). This is an operational/auth machine, not a domain lifecycle. It enforces `disabled-cannot-auth` (login) and `session-active-user` (resume) as guards.

## (a) Named-unit contract table

| name | kind | signature | pre / post | maps to |
|---|---|---|---|---|
| `verifyCredentials` | actor | `(input{username,password}) -> User \| err{ErrBadCredentials,ErrDisabled,ErrLocked,ErrUnavailable}` | pre: username present. post: returns the User iff the argon2id hash matches; never returns User on bad credentials | C4 `crm.session -> crm.repo` ("Loads the user and verifies the password hash") |
| `writeSessionFile` | actor | `(input{userId,expiresAt}) -> ok \| err` | post: HMAC-signed token written to `~/.crm/session` | C4 `crm.session -> crm.sessionfile` |
| `readSessionFile` | actor | `() -> {userId,expiresAt} \| err{ErrNoSession,ErrExpired,ErrUnreadable}` | post: parsed token or typed error | C4 `crm.session -> crm.sessionfile` |
| `loadUser` | actor | `(input{userId}) -> User \| err{ErrNotFound,ErrLocked,ErrUnavailable}` | post: returns the User with its current status | C4 `crm.session -> crm.repo` |
| `clearSessionFile` | actor | `() -> ok \| err` | post: token removed/truncated (best-effort) | C4 `crm.session -> crm.sessionfile` |
| `guardUserDisabled` | guard | `(ctx,evt) -> bool` | true iff the verified user's status == Disabled (deny path) | inv `disabled-cannot-auth` |
| `guardSessionUserActive` | guard | `(ctx,evt) -> bool` | true iff the loaded user's status == Active | inv `session-active-user` |
| `guardSessionExpired` | guard | `(ctx,evt) -> bool` | true iff token `expiresAt <= now` | Session expiry (validity window for `session-active-user`) |
| `isErrBadCredentials` / `isErrDisabled` / `isErrLocked` / `isErrNoSession` / `isErrExpired` / `isErrNotFound` | guard | `(ctx,evt) -> bool` | true iff `evt.error` is that typed error | C4 sections 5/6 error types |
| `retriesExhausted` | guard | `(ctx) -> bool` | true iff `ctx.retries >= 3` | C4 section 6 bound (retry <= 3, ~1.5s) |
| `setCredentials` | action | `(ctx,evt) -> ctx` | `username:=evt.username` (password held transiently for the invoke, never stored) | supports `password-hashed` |
| `captureUser` | action | `(ctx,evt) -> ctx` | `userId,role,teamId,userStatus := verified/loaded user` | - |
| `captureToken` | action | `(ctx,evt) -> ctx` | `userId,expiresAt := token` | - |
| `incrementRetries` | action | `(ctx) -> ctx` | `retries:=retries+1` | - |
| `recordExpired` | action | `(ctx) -> ctx` | mark session expired; drop in-memory identity | supports `session-active-user` |
| `recordDisabled` / `recordBadCredentials` / `recordUserNotActive` / `recordUserMissing` | action | `(ctx,evt) -> ctx` | set the auth-denial reason for the CLI | surfaces `disabled-cannot-auth` / `session-active-user` |
| `recordError` / `recordVerifyError` / `recordFileError` / `recordLoadError` / `recordTimeout` / `recordRetriesExhausted` | action | `(ctx,evt) -> ctx` | `lastError:=classified error` | maps repo/file errors |
| `recordLogoutWarning` | action | `(ctx,evt) -> ctx` | note best-effort logout (token may remain) | residual-risk marker |
| `recordSessionUsed` / `recordAlreadyActive` / `recordAlreadyResolved` / `recordNoSession` / `recordNoSessionToLogout` / `recordNoActiveSession` / `recordExpiredNeedsLogin` / `recordSessionExpired` | action | `(ctx,evt) -> ctx` | set a no-op/reject reason; no state change | explicit event handling |

## (b) Failure catalog

| failure | detection | transition | recovery | bounding mitigation / residual risk |
|---|---|---|---|---|
| Bad password | `verifyCredentials` onError `isErrBadCredentials` | `Authenticating -> AuthFailed` | user retries `crm login` (NOT auto-retried, section 5) | Residual: none; brute-force slowed by argon2id |
| Disabled user logs in | `verifyCredentials` onDone `guardUserDisabled`, or onError `isErrDisabled` | `Authenticating -> AuthDenied` | admin must `enable` the user | inv `disabled-cannot-auth`. Residual: none |
| Store locked during verify | `verifyCredentials` onError `isErrLocked` | `Authenticating -> VerifyRetry -> Authenticating` then `SessionUnavailable` when `retriesExhausted` | bounded retry, then surface | C4 6: retry <= 3, ~1.5s. Residual: refused after 3 |
| Store unavailable/corrupt during verify | `verifyCredentials` onError (else) | `Authenticating -> SessionUnavailable` | surface; envelope reports DBError/Corrupt | C4 6: fatal at envelope for Corrupt. Residual: restore from backup |
| Verify timeout | `after verifyTimeout` (5s) | `Authenticating -> SessionUnavailable` | surface | Residual: none |
| Token write fails | `writeSessionFile` onError / `after fileIoTimeout` | `WritingSession -> SessionUnavailable` | verify passed but no token persisted; user retries login | Residual: no session established (fail closed) |
| No session on resume | `readSessionFile` onError `isErrNoSession` | `Resolving -> Anonymous` | require `crm login` | C4 6: user re-authenticates. Residual: none |
| Token expired | `readSessionFile` onDone `guardSessionExpired`, or onError `isErrExpired`, or `after sessionTTL` on Active | `-> Expired` | require `crm login` | signed expiry authoritative. Residual: none |
| Token unreadable | `readSessionFile` onError (else) / `after fileIoTimeout` | `Resolving -> SessionUnavailable` | surface; re-login | Residual: corrupt token; delete and re-login |
| User no longer Active on resume | `loadUser` onDone `!guardSessionUserActive`, or onError `isErrNotFound` | `CheckingUser -> Invalidated` | re-auth (denied if still disabled) | inv `session-active-user`. Residual: none |
| Store locked / timeout on resume load | `loadUser` onError `isErrLocked` -> VerifyRetry; `after loadUserTimeout` (10s) -> SessionUnavailable | as noted | bounded retry / surface | C4 6: retry <= 3, timeout 10s |
| Logout cannot clear token | `clearSessionFile` onError / `after fileIoTimeout` | `LoggingOut -> LoggedOut` (best-effort) | in-memory identity dropped regardless | Residual: stale token file; mitigated by HMAC signature + expiry + resume-time re-validation |

## (c) Transition matrix (hard-TDD oracle - one row per transition and per guard branch)

| # | source | event / after / always | guard | target | actions | derived-from |
|---|---|---|---|---|---|---|
| 1 | Anonymous | login | - | Authenticating | setCredentials | login |
| 2 | Anonymous | resume | - | Resolving | - | Current() resolution |
| 3 | Anonymous | logout | - | Anonymous (internal) | recordNoSessionToLogout | explicit ignore |
| 4 | Anonymous | useSession | - | Anonymous (internal) | recordNoActiveSession | explicit reject (no session) |
| 5 | Authenticating | invoke onDone | guardUserDisabled | AuthDenied | recordDisabled | disabled-cannot-auth |
| 6 | Authenticating | invoke onDone | (else) | WritingSession | captureUser | verify ok |
| 7 | Authenticating | invoke onError | isErrBadCredentials | AuthFailed | recordBadCredentials | ErrBadCredentials |
| 8 | Authenticating | invoke onError | isErrDisabled | AuthDenied | recordDisabled | disabled-cannot-auth |
| 9 | Authenticating | invoke onError | isErrLocked | VerifyRetry | recordError | C4 6 store-locked |
| 10 | Authenticating | invoke onError | (else) | SessionUnavailable | recordVerifyError | store unavailable/corrupt |
| 11 | Authenticating | after verifyTimeout | - | SessionUnavailable | recordTimeout | verify timeout 5s |
| 12 | VerifyRetry | always | retriesExhausted | SessionUnavailable | recordRetriesExhausted | C4 6 bound retry<=3 |
| 13 | VerifyRetry | after verifyRetryBackoff | - | Authenticating | incrementRetries | C4 6 backoff ~0.5s |
| 14 | WritingSession | invoke onDone | - | Active | - | token written |
| 15 | WritingSession | invoke onError | - | SessionUnavailable | recordFileError | token write failed |
| 16 | WritingSession | after fileIoTimeout | - | SessionUnavailable | recordTimeout | file io timeout 2s |
| 17 | Resolving | invoke onDone | guardSessionExpired | Expired | recordExpired | token expiry |
| 18 | Resolving | invoke onDone | (else) | CheckingUser | captureToken | token valid |
| 19 | Resolving | invoke onError | isErrNoSession | Anonymous | recordNoSession | ErrNoSession |
| 20 | Resolving | invoke onError | isErrExpired | Expired | recordExpired | ErrExpired |
| 21 | Resolving | invoke onError | (else) | SessionUnavailable | recordFileError | token unreadable |
| 22 | Resolving | after fileIoTimeout | - | SessionUnavailable | recordTimeout | file io timeout 2s |
| 23 | CheckingUser | invoke onDone | guardSessionUserActive | Active | captureUser | session-active-user |
| 24 | CheckingUser | invoke onDone | (else) | Invalidated | recordUserNotActive | session-active-user (deny) |
| 25 | CheckingUser | invoke onError | isErrLocked | VerifyRetry | recordError | C4 6 store-locked |
| 26 | CheckingUser | invoke onError | isErrNotFound | Invalidated | recordUserMissing | user deleted |
| 27 | CheckingUser | invoke onError | (else) | SessionUnavailable | recordLoadError | store unavailable |
| 28 | CheckingUser | after loadUserTimeout | - | SessionUnavailable | recordTimeout | query timeout 10s |
| 29 | Active | logout | - | LoggingOut | - | logout |
| 30 | Active | useSession | - | Active (internal) | recordSessionUsed | command uses the session |
| 31 | Active | login | - | Active (internal) | recordAlreadyActive | explicit ignore |
| 32 | Active | resume | - | Active (internal) | recordAlreadyResolved | explicit ignore |
| 33 | Active | after sessionTTL | - | Expired | recordExpired | token expiry (after) |
| 34 | LoggingOut | invoke onDone | - | LoggedOut | - | token cleared |
| 35 | LoggingOut | invoke onError | - | LoggedOut | recordLogoutWarning | best-effort logout |
| 36 | LoggingOut | after fileIoTimeout | - | LoggedOut | recordLogoutWarning | best-effort logout |
| 37 | Expired | login | - | Authenticating | setCredentials | re-auth |
| 38 | Expired | resume | - | Expired (internal) | recordExpiredNeedsLogin | explicit reject |
| 39 | Expired | logout | - | Expired (internal) | recordNoSessionToLogout | explicit ignore |
| 40 | Expired | useSession | - | Expired (internal) | recordSessionExpired | explicit reject |
| 41 | LoggedOut | login | - | Authenticating | setCredentials | re-auth |
| 42 | LoggedOut | resume | - | LoggedOut (internal) | recordNoSession | explicit reject |
| 43 | LoggedOut | logout | - | LoggedOut (internal) | recordNoSessionToLogout | explicit ignore |
| 44 | LoggedOut | useSession | - | LoggedOut (internal) | recordNoActiveSession | explicit reject |
| 45 | AuthFailed | login | - | Authenticating | setCredentials | retry auth |
| 46 | AuthFailed | resume | - | AuthFailed (internal) | recordNoSession | explicit reject |
| 47 | AuthFailed | logout | - | AuthFailed (internal) | recordNoSessionToLogout | explicit ignore |
| 48 | AuthFailed | useSession | - | AuthFailed (internal) | recordNoActiveSession | explicit reject |
| 49 | AuthDenied | login | - | Authenticating | setCredentials | retry (denied if still disabled) |
| 50 | AuthDenied | resume | - | AuthDenied (internal) | recordNoSession | explicit reject |
| 51 | AuthDenied | logout | - | AuthDenied (internal) | recordNoSessionToLogout | explicit ignore |
| 52 | AuthDenied | useSession | - | AuthDenied (internal) | recordNoActiveSession | explicit reject |
| 53 | Invalidated | login | - | Authenticating | setCredentials | re-auth |
| 54 | Invalidated | resume | - | Invalidated (internal) | recordNoSession | explicit reject |
| 55 | Invalidated | logout | - | Invalidated (internal) | recordNoSessionToLogout | explicit ignore |
| 56 | Invalidated | useSession | - | Invalidated (internal) | recordNoActiveSession | explicit reject |
| 57 | SessionUnavailable | login | - | Authenticating | setCredentials | retry auth |
| 58 | SessionUnavailable | resume | - | Resolving | - | retry resolution |
| 59 | SessionUnavailable | logout | - | SessionUnavailable (internal) | recordNoSessionToLogout | explicit ignore |
| 60 | SessionUnavailable | useSession | - | SessionUnavailable (internal) | recordNoActiveSession | explicit reject |

Event completeness: every state handles all of {login, logout, resume, useSession} explicitly; transient invoke/after/always states (`Authenticating`, `VerifyRetry`, `WritingSession`, `Resolving`, `CheckingUser`, `LoggingOut`) auto-advance and cannot receive user events within one command.
