# 004 — Replace MinIO with RustFS and move mise tasks to files

- **Status:** done
- **Date:** 2026-10-27

## Problem

MinIO is deprecated (maintenance mode). The project used it for local
integration testing in `docker-compose.yml`, `docker-init-minio.sh`, the
`.mise.toml` tasks, and both READMEs. All test tasks were also embedded in
`.mise.toml`, which was large and duplicated flags on every `go run .` call.

## Fix

1. RustFS (`rustfs/rustfs` image, mise tool `rustfs`) replaces MinIO
   everywhere. Health endpoint is `/health`. The bucket-creation init
   container now uses `peakcom/s5cmd` — the same S3 engine `pkg/s3` uses —
   instead of MinIO's `mc`. `docker-init-minio.sh` is gone; the init runs
   inline in compose and is idempotent.
2. All tasks moved from `.mise.toml` to standalone executable scripts in
   `mise-tasks/` (mise discovers this directory by default; `#MISE`
   directives carry description and dependencies). `.mise.toml` removed.
3. Shared dev config (endpoints, bucket, credentials) moved once into
   `[env]` in `mise.toml`; the app reads those exact names via kong
   `DefaultEnvars`, so every `go run .` task lost its 7 repeated flags.
   `test-restore-local` and `test-restore-s3` merged into
   `test-restore [local|s3]`.

Verified end to end with rustfs 1.0.4: snapshot upload, list, and restore
against a live rustfs; compose file validated with `docker compose config`.
