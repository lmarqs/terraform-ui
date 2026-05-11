---
title: Remote Source Providers (HTTP, S3, GCS)
status: planned
priority: medium
created: 2026-05-11
effort: medium
tags: [source, providers]
depends_on: [composite-service]
---

## Summary

Add HTTP and S3 providers to the source abstraction so plan/state can be loaded from remote URLs without downloading manually.

## Problem

Users store plan artifacts in CI (S3 buckets, HTTP artifact servers). Currently they must download first, then point tfui at the local file. This adds friction to the PR review workflow.

## Design

Register new providers in the Resolver:

```go
resolver.Register(&HTTPProvider{client: http.DefaultClient})
resolver.Register(&S3Provider{client: s3Client})
```

**URI examples:**
```bash
tfui --plan s3://ci-artifacts/plans/pr-123.json
tfui --plan https://ci.example.com/artifacts/plan.json
tfui --state s3://terraform-state/prod/global.tfstate
```

**HTTPProvider:** Simple `http.Get()` + read body. Support `Authorization` header via env var or config.

**S3Provider:** Reuse `aws-sdk-go-v2` (already in deps via AI module). Add `service/s3` explicitly. Use default credential chain.

**Progress:** Show download progress for large files (spinner or byte count on stderr).

## Open Questions

- Authentication for HTTP (Bearer token? Basic auth? Header from env var?)
- Should S3 support assume-role for cross-account access?
- GCS and Azure in same phase or separate?

## Tasks

- [ ] HTTPProvider (GET + read body)
- [ ] S3Provider (aws-sdk-go-v2/service/s3)
- [ ] Progress indication for downloads
- [ ] Authentication config (env vars or tfui.yaml)
- [ ] Integration tests with mock HTTP server
