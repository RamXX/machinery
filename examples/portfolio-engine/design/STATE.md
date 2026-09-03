# STATE: Drawdown Portfolio Recommender design ledger

| phase | status | date | notes |
|---|---|---|---|
| Phase 0 Frame | gate-passed | 2026-07-02 | Local Python tool, DuckDB, HTTP market data; 16-of-N min-drawdown. |
| Phase 1 Domain model | gate-passed | 2026-07-02 | modelith lint clean (0/0); rendered, em-dashes stripped. |
| Phase 2 Architecture | gate-passed | 2026-07-02 | G2 ok; 7 boundaries, 2 externals, 4 deps mitigated. |
| Phase 3 State machines | gate-passed | 2026-09-02 | G3 strict: 4 machines, 46 transitions, 4 fresh oracles, 39 named units; zero warnings. |
| Formal layer | gate-passed | 2026-09-02 | machinery verify-formal 7/7: control-flow x4, PortfolioData+Refinement with routingFault stutter, RecommendationRunData; zero failures. |
| Phase 4 BUILD.md | gate-passed | 2026-09-02 | strict full design check: 18 invariants = 13 unit-backed + 5 reasoned waivers; Gu 9/9 surfaces and Gv 12/12 attestations; zero warnings. |

All gates green. Design only.

## Phase-exit self-reviews (retrofit, 2026-07-22)

The self-review discipline postdates this design's original run; these lines were added in a
retrofit on 2026-07-22 by re-running the five-question adversarial pass over each phase's artifact
as it stands today. They are reviews of the committed artifacts, not reconstructions of the 2026-07-02
sessions.

- Phase 1 Domain model: `self-review: reality=clean depth=clean scope=clean coverage=clean consistency=clean`
- Phase 2 Architecture: `self-review: reality=fixed(stale Python gate-runner reference replaced with the machinery binary) depth=clean scope=clean coverage=clean consistency=clean`
- Phase 3 State machines: `self-review: reality=clean depth=clean scope=clean coverage=clean consistency=clean`
- Phase 4 BUILD.md: `self-review: reality=fixed(Python-toolchain gate commands replaced with the machinery binary; BUILD.md converted to manifest+shards, see DECISIONS.md 2026-07-22) depth=clean scope=clean coverage=clean consistency=clean`
