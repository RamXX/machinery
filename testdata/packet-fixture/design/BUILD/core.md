# BUILD shard: core

Milestones: M0, M1

## 1. Charter

Core owns the deal aggregate. This section is never cited by a slice.

## 8. Test specification

### 8.1 Oracle conformance

Every row of `Deal.oracle.md` is one test, keyed on its stable id.

```text
# a fenced block with a heading-like line
## not a heading
```

### 8.2 Named units

`guardCanAdvance` and `guardCanWrite` each get a falsifying-clause test.

## 9. Build plan

- **M0** (walking skeleton). Core's slice: the advance path only.
- **M1** (deal lifecycle). Core's slice: the win path, the lose path, and the policy
  rows held to the oracle. Both paths persist inside one write transaction.

### 9.1 Notes

Nothing here is part of a milestone block.
