---- MODULE Session ----
EXTENDS Naturals

\* Generated from Session.machine.json by machinery tla. Control-flow model.
\*
\* ASSUMPTIONS (what this abstraction erases; the proof is conditional on them):
\*   1. Guards are erased to nondeterminism: SOUND for safety. For LIVENESS this
\*      is conditional on every fully guarded branch list being exhaustive.
\*      machine_lint requires an unguarded fallback or an _exhaustive note; where
\*      an _exhaustive note is used TLC CANNOT verify it, so the liveness result
\*      below is only as sound as these hand-checked, UNVERIFIED claims:
\*      (none here: every guarded branch list has an unguarded fallback)
\*   2. Every invoke resolves exactly once (onDone or onError; no lost or
\*      duplicated completion) and every after timer eventually fires.
\*   3. Single machine instance; no interleaving with other instances or
\*      machines, no message loss/duplication/reordering between machines.
\*   4. Context data, event payloads, action effects, and real time (the
\*      _delays values) are not modeled at this rung; the data-refined rung
\*      (refine_gen) and the implementation tests carry those.
CONSTANT MaxRetries
VARIABLES st, rc1
vars == << st, rc1 >>

States == {"Active", "Anonymous", "AuthDenied", "AuthFailed", "Authenticating", "CheckingUser", "Expired", "Invalidated", "LoggedOut", "LoggingOut", "Resolving", "SessionUnavailable", "VerifyRetry", "WritingSession"}
Domain == {"Active", "Anonymous", "AuthDenied", "AuthFailed", "Expired", "Invalidated", "LoggedOut", "SessionUnavailable"}
Overlay == {"Authenticating", "CheckingUser", "LoggingOut", "Resolving", "VerifyRetry", "WritingSession"}

TypeOK == st \in States /\ rc1 \in 0..MaxRetries
Init == st = "Anonymous" /\ rc1 = 0

  \* T1: Anonymous -on:login-> Authenticating
  \* T2: Anonymous -on:resume-> Resolving
  \* T3: Anonymous -on:logout-> Anonymous
  \* T4: Anonymous -on:useSession-> Anonymous
  \* T5: Authenticating -after:verifyTimeout-> SessionUnavailable
  \* T6: Authenticating -onDone:verifyCredentials-> AuthDenied
  \* T7: Authenticating -onDone:verifyCredentials-> WritingSession
  \* T8: Authenticating -onError:verifyCredentials-> AuthFailed
  \* T9: Authenticating -onError:verifyCredentials-> AuthDenied
  \* T10: Authenticating -onError:verifyCredentials-> VerifyRetry
  \* T11: Authenticating -onError:verifyCredentials-> SessionUnavailable
  \* T12: WritingSession -after:fileIoTimeout-> SessionUnavailable
  \* T13: WritingSession -onDone:writeSessionFile-> Active
  \* T14: WritingSession -onError:writeSessionFile-> SessionUnavailable
  \* T15: Resolving -after:fileIoTimeout-> SessionUnavailable
  \* T16: Resolving -onDone:readSessionFile-> Expired
  \* T17: Resolving -onDone:readSessionFile-> CheckingUser
  \* T18: Resolving -onError:readSessionFile-> Anonymous
  \* T19: Resolving -onError:readSessionFile-> Expired
  \* T20: Resolving -onError:readSessionFile-> SessionUnavailable
  \* T21: CheckingUser -after:loadUserTimeout-> SessionUnavailable
  \* T22: CheckingUser -onDone:loadUser-> Active
  \* T23: CheckingUser -onDone:loadUser-> Invalidated
  \* T24: CheckingUser -onError:loadUser-> VerifyRetry
  \* T25: CheckingUser -onError:loadUser-> Invalidated
  \* T26: CheckingUser -onError:loadUser-> SessionUnavailable
  \* T27: Active -on:logout-> LoggingOut
  \* T28: Active -on:useSession-> Active
  \* T29: Active -on:login-> Active
  \* T30: Active -on:resume-> Active
  \* T31: Active -after:sessionTTL-> Expired
  \* T32: LoggingOut -after:fileIoTimeout-> LoggedOut
  \* T33: LoggingOut -onDone:clearSessionFile-> LoggedOut
  \* T34: LoggingOut -onError:clearSessionFile-> LoggedOut
  \* T35: Expired -on:login-> Authenticating
  \* T36: Expired -on:resume-> Expired
  \* T37: Expired -on:logout-> Expired
  \* T38: Expired -on:useSession-> Expired
  \* T39: LoggedOut -on:login-> Authenticating
  \* T40: LoggedOut -on:resume-> LoggedOut
  \* T41: LoggedOut -on:logout-> LoggedOut
  \* T42: LoggedOut -on:useSession-> LoggedOut
  \* T43: AuthFailed -on:login-> Authenticating
  \* T44: AuthFailed -on:resume-> AuthFailed
  \* T45: AuthFailed -on:logout-> AuthFailed
  \* T46: AuthFailed -on:useSession-> AuthFailed
  \* T47: AuthDenied -on:login-> Authenticating
  \* T48: AuthDenied -on:resume-> AuthDenied
  \* T49: AuthDenied -on:logout-> AuthDenied
  \* T50: AuthDenied -on:useSession-> AuthDenied
  \* T51: Invalidated -on:login-> Authenticating
  \* T52: Invalidated -on:resume-> Invalidated
  \* T53: Invalidated -on:logout-> Invalidated
  \* T54: Invalidated -on:useSession-> Invalidated
  \* T55: SessionUnavailable -on:login-> Authenticating
  \* T56: SessionUnavailable -on:resume-> Resolving
  \* T57: SessionUnavailable -on:logout-> SessionUnavailable
  \* T58: SessionUnavailable -on:useSession-> SessionUnavailable

T1 == st = "Anonymous" /\ st' = "Authenticating" /\ rc1' = 0
T2 == st = "Anonymous" /\ st' = "Resolving" /\ rc1' = 0
T3 == st = "Anonymous" /\ st' = "Anonymous" /\ rc1' = 0
T4 == st = "Anonymous" /\ st' = "Anonymous" /\ rc1' = 0
T5 == st = "Authenticating" /\ st' = "SessionUnavailable" /\ rc1' = 0
T6 == st = "Authenticating" /\ st' = "AuthDenied" /\ rc1' = 0
T7 == st = "Authenticating" /\ st' = "WritingSession" /\ rc1' = rc1
T8 == st = "Authenticating" /\ st' = "AuthFailed" /\ rc1' = 0
T9 == st = "Authenticating" /\ st' = "AuthDenied" /\ rc1' = 0
T10 == st = "Authenticating" /\ st' = "VerifyRetry" /\ rc1' = rc1
T11 == st = "Authenticating" /\ st' = "SessionUnavailable" /\ rc1' = 0
T12 == st = "WritingSession" /\ st' = "SessionUnavailable" /\ rc1' = 0
T13 == st = "WritingSession" /\ st' = "Active" /\ rc1' = 0
T14 == st = "WritingSession" /\ st' = "SessionUnavailable" /\ rc1' = 0
T15 == st = "Resolving" /\ st' = "SessionUnavailable" /\ rc1' = 0
T16 == st = "Resolving" /\ st' = "Expired" /\ rc1' = 0
T17 == st = "Resolving" /\ st' = "CheckingUser" /\ rc1' = rc1
T18 == st = "Resolving" /\ st' = "Anonymous" /\ rc1' = 0
T19 == st = "Resolving" /\ st' = "Expired" /\ rc1' = 0
T20 == st = "Resolving" /\ st' = "SessionUnavailable" /\ rc1' = 0
T21 == st = "CheckingUser" /\ st' = "SessionUnavailable" /\ rc1' = 0
T22 == st = "CheckingUser" /\ st' = "Active" /\ rc1' = 0
T23 == st = "CheckingUser" /\ st' = "Invalidated" /\ rc1' = 0
T24 == st = "CheckingUser" /\ st' = "VerifyRetry" /\ rc1' = rc1
T25 == st = "CheckingUser" /\ st' = "Invalidated" /\ rc1' = 0
T26 == st = "CheckingUser" /\ st' = "SessionUnavailable" /\ rc1' = 0
T27 == st = "Active" /\ st' = "LoggingOut" /\ rc1' = 0
T28 == st = "Active" /\ st' = "Active" /\ rc1' = 0
T29 == st = "Active" /\ st' = "Active" /\ rc1' = 0
T30 == st = "Active" /\ st' = "Active" /\ rc1' = 0
T31 == st = "Active" /\ st' = "Expired" /\ rc1' = 0
T32 == st = "LoggingOut" /\ st' = "LoggedOut" /\ rc1' = 0
T33 == st = "LoggingOut" /\ st' = "LoggedOut" /\ rc1' = 0
T34 == st = "LoggingOut" /\ st' = "LoggedOut" /\ rc1' = 0
T35 == st = "Expired" /\ st' = "Authenticating" /\ rc1' = 0
T36 == st = "Expired" /\ st' = "Expired" /\ rc1' = 0
T37 == st = "Expired" /\ st' = "Expired" /\ rc1' = 0
T38 == st = "Expired" /\ st' = "Expired" /\ rc1' = 0
T39 == st = "LoggedOut" /\ st' = "Authenticating" /\ rc1' = 0
T40 == st = "LoggedOut" /\ st' = "LoggedOut" /\ rc1' = 0
T41 == st = "LoggedOut" /\ st' = "LoggedOut" /\ rc1' = 0
T42 == st = "LoggedOut" /\ st' = "LoggedOut" /\ rc1' = 0
T43 == st = "AuthFailed" /\ st' = "Authenticating" /\ rc1' = 0
T44 == st = "AuthFailed" /\ st' = "AuthFailed" /\ rc1' = 0
T45 == st = "AuthFailed" /\ st' = "AuthFailed" /\ rc1' = 0
T46 == st = "AuthFailed" /\ st' = "AuthFailed" /\ rc1' = 0
T47 == st = "AuthDenied" /\ st' = "Authenticating" /\ rc1' = 0
T48 == st = "AuthDenied" /\ st' = "AuthDenied" /\ rc1' = 0
T49 == st = "AuthDenied" /\ st' = "AuthDenied" /\ rc1' = 0
T50 == st = "AuthDenied" /\ st' = "AuthDenied" /\ rc1' = 0
T51 == st = "Invalidated" /\ st' = "Authenticating" /\ rc1' = 0
T52 == st = "Invalidated" /\ st' = "Invalidated" /\ rc1' = 0
T53 == st = "Invalidated" /\ st' = "Invalidated" /\ rc1' = 0
T54 == st = "Invalidated" /\ st' = "Invalidated" /\ rc1' = 0
T55 == st = "SessionUnavailable" /\ st' = "Authenticating" /\ rc1' = 0
T56 == st = "SessionUnavailable" /\ st' = "Resolving" /\ rc1' = 0
T57 == st = "SessionUnavailable" /\ st' = "SessionUnavailable" /\ rc1' = 0
T58 == st = "SessionUnavailable" /\ st' = "SessionUnavailable" /\ rc1' = 0
RetryExhausted_VerifyRetry == st = "VerifyRetry" /\ rc1 >= MaxRetries /\ st' = "SessionUnavailable" /\ rc1' = rc1
RetryAgain_VerifyRetry == st = "VerifyRetry" /\ rc1 < MaxRetries /\ st' = "Authenticating" /\ rc1' = rc1 + 1

DomainNext == T1 \/ T2 \/ T3 \/ T4 \/ T27 \/ T28 \/ T29 \/ T30 \/ T31 \/ T35 \/ T36 \/ T37 \/ T38 \/ T39 \/ T40 \/ T41 \/ T42 \/ T43 \/ T44 \/ T45 \/ T46 \/ T47 \/ T48 \/ T49 \/ T50 \/ T51 \/ T52 \/ T53 \/ T54 \/ T55 \/ T56 \/ T57 \/ T58
OverlayNext == T5 \/ T6 \/ T7 \/ T8 \/ T9 \/ T10 \/ T11 \/ T12 \/ T13 \/ T14 \/ T15 \/ T16 \/ T17 \/ T18 \/ T19 \/ T20 \/ T21 \/ T22 \/ T23 \/ T24 \/ T25 \/ T26 \/ T32 \/ T33 \/ T34 \/ RetryExhausted_VerifyRetry \/ RetryAgain_VerifyRetry
Next == DomainNext \/ OverlayNext

Spec == Init /\ [][Next]_vars /\ WF_vars(OverlayNext)

Live_OverlayResolves == (st \in Overlay) ~> (st \in Domain)
====
