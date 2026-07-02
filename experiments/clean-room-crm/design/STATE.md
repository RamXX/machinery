# STATE: TermCRM design session ledger

One line per phase: status (pending / in-progress / gate-passed), date, open questions.

| phase | status | date | notes |
|---|---|---|---|
| Phase 0 Frame | gate-passed | 2026-07-02 | Terminal CRM, single Go binary, embedded LadybugDB. |
| Phase 1 Domain model | gate-passed | 2026-07-02 | modelith lint clean (0/0); rendered, em-dashes stripped. |
| Phase 2 Architecture | gate-passed | 2026-07-02 | G2-c4 ok; 5 boundaries, 1 external, mitigation covered. |
| Phase 3 State machines | gate-passed | 2026-07-02 | G3 clean: 3 machines, 60 transitions, oracles fresh, 49 named units. Gx structural checks pass; 8 invariants pending BUILD.md traceability. |
| Formal layer | gate-passed | 2026-07-02 | verify_formal.sh: 5 passed, 0 failed (Deal, Task, CommandExecution control-flow; DealData; DealRefinement). |
| Phase 4 BUILD.md | gate-passed | 2026-07-02 | Full machinery_check ok (G2, G3, Gx); 18 invariants enforced. verify_formal 5/5. |

All gates green. Design complete (design only; no implementation).
