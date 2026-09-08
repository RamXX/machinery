---
id: MAC-33sp
title: "Directory ABA witness blind on Linux 6.8 in containers: readRootDirectory accepts a create-delete ABA"
status: open
priority: 1
type: bug
labels: [security, linux, dirscan, gates]
created_at: 2026-09-08T16:46:10Z
created_by: ramirosalas
updated_at: 2026-09-08T16:46:10Z
content_hash: "sha256:d3802085c83d686997dad10e5346bf1fc629a58a37a29ac3924977841adc5724"
---

## Description
## Symptom
`TestReadRootDirectoryRejectsCreateDeleteABA` (internal/gates/inventory_hardening_test.go:36) fails deterministically on Linux inside a Docker container, with the temp directory both on overlayfs and on an ext4 bind mount, as root and as uid 1000:

```
inventory_hardening_test.go:36: readRootDirectory accepted same-directory ABA: <nil>
```

Host: Ubuntu, kernel 6.8.0-138-generic x86_64, Docker 29.7.2, image golang:1.27.1, `--init --cpus=2`. Commit f939081 (0.7.0 candidate). The same test passes on the hosted ubuntu-latest runner (ci test job, run 34225004428 shows no gates failure) and on darwin.

## Why it matters
`readRootDirectory` (internal/gates/confinement.go:290) is the fail-closed inventory primitive: its witness must detect a create-then-delete between enumeration and use. If the witness is blind under a common Linux configuration (container, kernel 6.8, coarse timestamps) the ABA defense silently degrades to accepting the enumeration, which is the fail-open direction. Machinery will increasingly run inside containers (the Dagger migration, MAC-zafm).

## Ask
Determine which witness component is blind here (directory ctime granularity on 6.8 without multigrain timestamps, the mutation-event witness from MAC-o82q not being available or not being consulted, statx change cookie support, bind-mount semantics), reproduce on the VM (ssh root@5.78.87.91, workspace /root/machinery-ci only, do not disturb other containers), and either make the witness granularity-independent on this configuration or make readRootDirectory fail closed (refuse the enumeration with a diagnostic) when no reliable witness is available, never accept silently.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments
