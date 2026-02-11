# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

OIDC Relying Party (Authorization Code Flow) implemented in Go without external OIDC libraries, for educational purposes. Go 1.25+, module name `oidc-tutorial`. No source code exists yet; detailed specs are in `.claude/rules/`.

See @README.md for project overview.

## Canonical Docs (Source of Truth)

The single source of truth lives under `.claude/rules/`. If anything in this file conflicts with the docs below, **prefer `.claude/rules/`**.

- @.claude/rules/coding-archtecture.md — layering, responsibilities, directory layout
- @.claude/rules/system-archtecture.md — system-level architecture
- @.claude/rules/security.md — OIDC flow and security requirements
- @.claude/rules/shared-store.md — shared store schema, keys, and TTLs
- @.claude/rules/coding-rules.md — Go conventions and project-specific coding rules
- @.claude/rules/git-workflow.md — git workflow

## Build & Development Commands

```bash
# Build
go build ./...

# Run tests
go test ./...

# Run a single test
go test ./internal/domain/model -run TestFunctionName

# Format + import management (run before commit)
goimports -w .

# Lint
golangci-lint run

# Static analysis
go vet ./...
```

Install required tools:
```bash
go install golang.org/x/tools/cmd/goimports@latest
```
