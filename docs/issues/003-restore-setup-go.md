# 003 — Restore Go toolchain setup in release workflow

- **Status:** done
- **Date:** 2026-10-27

## Problem

The Go 1.27 upgrade dropped `actions/setup-go` from the release workflow,
assuming GoReleaser brings its own toolchain. It does not. Verified locally
with `act`: goreleaser v2.19.0 fails with
`exec: "go": executable file not found in $PATH`. GitHub-hosted runners
preinstall an uncontrolled Go version that may not satisfy the `go 1.27`
directive in `go.mod`.

## Fix

Added `actions/setup-go@v7` with `go-version-file: go.mod`, so the workflow
always builds with the exact toolchain the module declares, with module/build
caching on by default.
