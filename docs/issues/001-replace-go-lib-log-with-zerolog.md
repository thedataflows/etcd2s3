# 001 — Replace go-lib-log with direct zerolog

- **Status:** done
- **Date:** 2026-10-27

## Problem

The Go 1.24 → 1.27 upgrade brought go-lib-log v1.9.1, which renamed its API
(`Logger` became `func() *CustomLogger`, `SetLoggerLogLevel` →
`SetGlobalLoggerLogLevel`). The project no longer compiled.

## Decision

Drop `github.com/thedataflows/go-lib-log` and call `github.com/rs/zerolog/log`
directly. All go-lib-log helpers were thin wrappers over zerolog
(`Infof(pkg, format, args...)` ≡ `Info().Str("pkg", pkg).Msgf(format, args...)`).

Preserved behavior:

- `--log-level` / `--log-format` (console, json) flags and their env vars.
- Console format: RFC3339 timestamp, stderr, `pkg` field.
- JSON format: stderr with `pkg` and `error` fields.

Not preserved: go-lib-log's internal buffering/rate limiting (not needed for a
CLI that emits a handful of log lines).
