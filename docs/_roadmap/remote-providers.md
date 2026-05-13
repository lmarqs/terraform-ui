---
title: Remote Source Providers (HTTP, S3)
status: planned
priority: medium
created: 2026-05-11
effort: medium
tags: [source, providers, ci]
depends_on: [composite-service]
---

## Summary

Load plan and state directly from remote URLs (S3 buckets, HTTP endpoints) without downloading files manually first.

## Need

In CI-driven workflows, plan artifacts live in S3 or artifact servers. The current workflow requires:

```bash
# Today: 3 commands, manual download
aws s3 cp s3://ci-artifacts/plans/pr-123.json ./plan.json
terraform state pull > ./state.json
tfui --plan ./plan.json --state ./state.json
```

This is friction. Users want:

```bash
# Desired: 1 command, direct
tfui --plan s3://ci-artifacts/plans/pr-123.json --state s3://terraform-state/prod.tfstate
```

## Expected UX

```bash
# S3 (uses default AWS credential chain — same as terraform)
tfui --plan s3://my-bucket/plans/pr-123.json
tfui --state s3://terraform-state/prod/terraform.tfstate

# HTTP (public or with auth)
tfui --plan https://ci.example.com/artifacts/plan.json
tfui --state https://internal.example.com/states/prod.json

# Mixed (S3 plan + local state, or HTTP plan + live terraform)
tfui --plan https://ci.example.com/plan.json --state ./local.tfstate
```

**Progress indication:**
```
Downloading plan from s3://my-bucket/plans/pr-123.json... (2.4 MB)
```

**Error messages:**
```
Error: loading plan from s3://bucket/key.json: NoSuchKey: The specified key does not exist
  Check the bucket name and key path. Use `aws s3 ls s3://bucket/` to list available files.

Error: loading state from https://example.com/state.json: 401 Unauthorized
  The server requires authentication. Set TFUI_HTTP_TOKEN or configure auth in tfui.hcl.
```

## Advantages

- **One-command PR review** — reviewer runs `tfui --plan s3://...` directly from the PR link
- **No local file management** — no temp files to clean up, no stale downloads
- **Credentials already configured** — reuses AWS credential chain (same as terraform backend)
- **Composable** — mix remote + local + live sources freely

## Effort Justification

**Medium** because:
- Source abstraction already exists (Provider interface, Resolver)
- Each provider is ~50 lines (HTTP GET + read body, S3 GetObject + read body)
- AWS SDK v2 is already in go.mod (via AI/Bedrock module)
- Need to explicitly add `aws-sdk-go-v2/service/s3` dependency
- Progress indication adds some stderr output complexity
- Authentication config (env vars for HTTP tokens) needs design

## Design

```go
// Register new providers — zero changes to consumers
resolver.Register(&HTTPProvider{client: http.DefaultClient})
resolver.Register(&S3Provider{client: s3Client})
```

**HTTPProvider:** `http.Get(uri)` → read body → return bytes. Support `TFUI_HTTP_TOKEN` env var for Bearer auth.

**S3Provider:** Parse `s3://bucket/key`, use default credential chain, `s3.GetObject` → read body.

## Open Questions

- HTTP authentication: env var only, or also support tfui.hcl config?
- Should S3 support cross-account assume-role? (or just rely on AWS profile/config)
- GCS/Azure in this phase or separate?
- Progress bar or just byte count on stderr?

## Tasks

- [ ] Add `aws-sdk-go-v2/service/s3` to go.mod
- [ ] Implement S3Provider (~50 lines)
- [ ] Implement HTTPProvider (~50 lines)
- [ ] Register providers in `buildStaticService` / `buildCompositeService`
- [ ] Progress indication on stderr
- [ ] Auth: `TFUI_HTTP_TOKEN` env var for HTTP
- [ ] Error messages with actionable suggestions
- [ ] Integration test with mock HTTP server

## References

- Current source abstraction: `internal/source/`
- AWS SDK already in go.mod: `github.com/aws/aws-sdk-go-v2`
- S3 GetObject: standard pattern with credential chain
