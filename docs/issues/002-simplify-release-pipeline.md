# 002 — Simplify release pipeline (goreleaser 2.19, Dockerfile, release workflow)

- **Status:** done
- **Date:** 2026-10-27

## Changes

- `.goreleaser.yml`: removed YAML anchors, `-mod=readonly` (the default), and
  the explicit sha256 algorithm (the default). Builds-first layout. No
  behavior change; `goreleaser check` (v2.19.0) validates clean.
- `Dockerfile`: builder stage now uses `golang:1.27.3` with a plain
  `go build` instead of the 1 GB+ `goreleaser/goreleaser` image. The
  dist-path coupling (`dist/default_linux_amd64_v1/etcd2s3`) is gone; the
  version stamp comes from the same `VERSION` build-arg. Single `RUN` for
  runtime setup.
- `release.yml`: merged the two jobs into one (one checkout, sequential
  steps). GoReleaser now runs before the image push, so a failed release no
  longer ships a container image. Removed the dead `latest` tag rule
  (`is_default_branch` never fires on a tag-only trigger).
- `.dockerignore`: exclude `/.git/`.
