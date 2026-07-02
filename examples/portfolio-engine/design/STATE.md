# STATE: Drawdown Portfolio Recommender design ledger

| phase | status | date | notes |
|---|---|---|---|
| Phase 0 Frame | gate-passed | 2026-07-02 | Local Python tool, DuckDB, HTTP market data; 16-of-N min-drawdown. |
| Phase 1 Domain model | gate-passed | 2026-07-02 | modelith lint clean (0/0); rendered, em-dashes stripped. |
| Phase 2 Architecture | gate-passed | 2026-07-02 | G2 ok; 7 boundaries, 2 externals, 4 deps mitigated. |
| Phase 3 State machines | gate-passed | 2026-07-02 | G3 ok: 3 machines, 33 transitions, oracles fresh, 31 named units. |
| Formal layer | gate-passed | 2026-07-02 | verify_formal 6/6: control-flow x3, PortfolioData+Refinement (linear-lifecycle, non-default overlay names), RecommendationRunData (terminal-lifecycle). |
| Phase 4 BUILD.md | gate-passed | 2026-07-02 | full machinery_check ok; 18 invariants = 6 unit-backed + 12 attested. |

All gates green. Design only.
