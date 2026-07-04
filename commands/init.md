---
description: Mark this repo as machinery-managed and configure the governance hooks
argument-hint: "[design-dir]"
allowed-tools: Bash(machinery:*), Read, Write, Edit, Glob, AskUserQuestion
---

Set this repository up for machinery governance. The plugin's hooks are a
no-op until the project root has a `.machinery.json` (or a conventional
`design/domain.modelith.yaml`), so this command is what turns them on.

1. If `.machinery.json` already exists, show it and offer to update it
   rather than overwriting.
2. Ask the user, as one batched multiple-choice question:
   - design directory (default `design`, or the first argument below),
   - staged gate list: automatic (default: gates activate as their
     artifacts appear) or an explicit list for brownfield adoption
     (day one is typically `g2,g4`; see docs/brownfield-team-guide.md),
   - enable the import-boundary gate G4 on source edits now? (sets `impl`,
     usually `.`; requires the Architecture Contract to declare boundary
     `modules:`, otherwise it will fail loudly on every file),
   - strict mode: block the end of any turn on ANY red gate finding, not
     only DRIFT and import violations (default off; right for repos whose
     design is complete and ratcheted).
3. Write `.machinery.json` with only the fields that differ from the
   defaults, for example: {"gates": "g2,g4", "impl": "."}
4. Run `machinery preflight`; if the binary or modelith is missing, point at
   the one-line installer in the README.
5. Remind the user that hooks load at session start: the governance hooks
   take effect on the next Claude Code session in this project.
6. For brownfield adoption, offer to start the archaeology run with
   /machinery:design brownfield, and point at docs/brownfield-team-guide.md
   for the team adoption ladder, baseline allow rules, and CI recipes.

Arguments: $ARGUMENTS
