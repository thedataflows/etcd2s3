# 005 — README: use mise tasks for local testing

- **Status:** done
- **Date:** 2026-10-27

## Problem

The README Development section still showed manual `go run` commands with
the outdated bucket name `etcd-snapshots-test`, and the Testing section
only linked elsewhere instead of showing the task flow.

## Fix

Development and Testing sections now document the `mise run` task flow
(`dev-start`, `test-setup`, `test-populate-etcd`, `test-snapshot`,
`test-list`, `test-restore`, `dev-stop`); noted `mise install` and the
mise-pinned toolchain; Go dependency line corrected to 1.27+.
