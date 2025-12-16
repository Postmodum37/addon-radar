# Development Hooks Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add best-practice pre-commit hooks with golangci-lint and Lefthook to improve developer experience.

**Architecture:** Lefthook (fast Git hooks manager) will run golangci-lint (Go linter aggregator) and gofmt before every commit. CI will also run golangci-lint to catch issues in PRs.

**Tech Stack:** golangci-lint v2.5.0, Lefthook v1.11+, GitHub Actions

---

## Background Research

### Why Lefthook over pre-commit?
- **Speed**: Written in Go, ~10x faster than Python-based pre-commit
- **No dependencies**: Single binary, no Python/pip required
- **Native Go support**: First-class support for Go projects
- **Simpler config**: YAML-based, intuitive for Go developers
- **Parallel execution**: Runs hooks in parallel by default

### Why golangci-lint?
- Industry standard Go meta-linter (aggregates 100+ linters)
- Fast parallel execution with caching
- Highly configurable via YAML
- Official GitHub Action available
- v2 is current stable version (Dec 2025)

### Linters to Enable
Default `standard` set plus:
- `gosec` - Security checks
- `errcheck` - Unchecked errors (already in standard)
- `goconst` - Repeated strings that could be constants
- `gocyclo` - Cyclomatic complexity
- `misspell` - Spelling mistakes in comments
- `unconvert` - Unnecessary type conversions

---

## Task 1: Install golangci-lint Locally

**Files:**
- Modify: `CLAUDE.md` (document the tool)

**Step 1: Install golangci-lint binary**

```bash
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.5.0
```

**Step 2: Verify installation**

Run: `golangci-lint --version`
Expected: `golangci-lint has version v2.5.0 built with go1.23.x`

**Step 3: Commit**

No commit yet - just local setup.

---

## Task 2: Create golangci-lint Configuration

**Files:**
- Create: `.golangci.yml`

**Step 1: Create the configuration file**

```yaml
# .golangci.yml
version: "2"

linters:
  default: standard
  enable:
    - gosec        # Security checks
    - goconst      # Repeated strings that could be constants
    - gocyclo      # Cyclomatic complexity
    - misspell     # Spelling mistakes
    - unconvert    # Unnecessary type conversions
  settings:
    errcheck:
      check-type-assertions: true
      check-blank: true
    gocyclo:
      min-complexity: 15
    goconst:
      min-len: 3
      min-occurrences: 3
    gosec:
      severity: medium
      confidence: medium

issues:
  max-issues-per-linter: 50
  max-same-issues: 5
  fix: false

run:
  timeout: 5m
  tests: true
  build-tags: []
  concurrency: 4

output:
  formats:
    text:
      print-linter-name: true
      print-issued-lines: true
```

**Step 2: Run linter to verify configuration**

Run: `golangci-lint run ./...`
Expected: Either clean output or list of issues to fix

**Step 3: Commit**

```bash
git add .golangci.yml
git commit -m "chore: add golangci-lint configuration

Configure standard linters plus gosec, goconst, gocyclo, misspell, unconvert.
Set cyclomatic complexity limit to 15.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 3: Fix Any Existing Lint Issues

**Files:**
- Modify: Any files with lint errors

**Step 1: Run linter and capture issues**

Run: `golangci-lint run ./... 2>&1 | head -100`
Expected: List of issues (if any)

**Step 2: Fix issues**

For each issue, make minimal changes to fix. Common fixes:
- Add error checks for unchecked errors
- Remove unused variables
- Fix spelling mistakes
- Simplify unnecessarily complex code

**Step 3: Verify fixes**

Run: `golangci-lint run ./...`
Expected: No output (clean)

**Step 4: Run tests to ensure nothing broke**

Run: `go test ./... -race -timeout=5m`
Expected: All tests pass

**Step 5: Commit**

```bash
git add -A
git commit -m "fix: resolve golangci-lint issues

Fix issues found by golangci-lint static analysis.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 4: Install Lefthook

**Files:**
- None (local installation)

**Step 1: Install Lefthook binary**

```bash
# macOS
brew install lefthook

# Or via Go
go install github.com/evilmartians/lefthook@latest
```

**Step 2: Verify installation**

Run: `lefthook version`
Expected: `1.11.x` or higher

**Step 3: No commit needed**

---

## Task 5: Create Lefthook Configuration

**Files:**
- Create: `lefthook.yml`

**Step 1: Create Lefthook configuration**

```yaml
# lefthook.yml
# Fast Git hooks manager - https://github.com/evilmartians/lefthook

pre-commit:
  parallel: true
  commands:
    lint:
      glob: "*.go"
      run: golangci-lint run --new-from-rev=HEAD~1 {staged_files}
      stage_fixed: true
    fmt:
      glob: "*.go"
      run: gofmt -l -w {staged_files}
      stage_fixed: true

pre-push:
  parallel: true
  commands:
    test:
      run: go test ./... -race -timeout=5m
    lint-all:
      run: golangci-lint run ./...
```

**Step 2: Install Lefthook hooks**

Run: `lefthook install`
Expected: `Lefthook installed`

**Step 3: Verify hooks are installed**

Run: `ls -la .git/hooks/pre-commit`
Expected: File exists and contains lefthook reference

**Step 4: Commit**

```bash
git add lefthook.yml
git commit -m "chore: add Lefthook git hooks configuration

Add pre-commit hooks for:
- golangci-lint (incremental, only changed files)
- gofmt (auto-format)

Add pre-push hooks for:
- Full test suite with race detection
- Full lint check

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 6: Add GitHub Actions Lint Workflow

**Files:**
- Create: `.github/workflows/lint.yml`

**Step 1: Create lint workflow**

```yaml
name: Lint

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

permissions:
  contents: read

jobs:
  golangci-lint:
    name: golangci-lint
    runs-on: ubuntu-latest
    timeout-minutes: 10
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.25'

      - name: golangci-lint
        uses: golangci/golangci-lint-action@v8
        with:
          version: v2.5.0
          args: --timeout=5m
```

**Step 2: Commit**

```bash
git add .github/workflows/lint.yml
git commit -m "ci: add golangci-lint GitHub Action

Run golangci-lint on push to main and on pull requests.
Uses official golangci-lint-action v8 with v2.5.0.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 7: Update Documentation

**Files:**
- Modify: `CLAUDE.md`
- Modify: `TODO.md`

**Step 1: Update CLAUDE.md with dev setup instructions**

Add to CLAUDE.md after "Environment Variables" section:

```markdown
## Development Setup

### Prerequisites
- Go 1.25+
- golangci-lint v2.5.0+
- Lefthook v1.11+

### Install Development Tools

```bash
# Install golangci-lint
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.5.0

# Install Lefthook (macOS)
brew install lefthook

# Install git hooks
lefthook install
```

### Git Hooks (via Lefthook)

| Hook | Commands | Description |
|------|----------|-------------|
| pre-commit | lint, fmt | Run golangci-lint and gofmt on staged Go files |
| pre-push | test, lint-all | Run full test suite and lint check |

### Manual Commands

```bash
# Run linter
golangci-lint run ./...

# Run tests
go test ./... -race -timeout=5m

# Format code
gofmt -l -w .
```
```

**Step 2: Update TODO.md**

Add completed section for dev hooks.

**Step 3: Commit**

```bash
git add CLAUDE.md TODO.md
git commit -m "docs: add development setup instructions

Document golangci-lint and Lefthook installation and usage.
Update TODO.md with completed dev hooks task.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 8: Test the Full Workflow

**Files:**
- None (verification only)

**Step 1: Test pre-commit hook**

Make a small change to a Go file:
```bash
echo "// test comment" >> internal/config/config.go
git add internal/config/config.go
git commit -m "test: verify pre-commit hook"
```
Expected: Lefthook runs lint and fmt, commit succeeds

**Step 2: Revert test commit**

```bash
git reset --hard HEAD~1
```

**Step 3: Test pre-push hook (optional)**

```bash
git push --dry-run origin main
```
Expected: Tests and full lint run

**Step 4: Verify CI workflow**

Push all commits and verify GitHub Actions lint workflow runs successfully.

---

## Summary

After completing all tasks, the project will have:

1. **`.golangci.yml`** - Linter configuration with sensible defaults
2. **`lefthook.yml`** - Git hooks for pre-commit (lint + fmt) and pre-push (test + lint)
3. **`.github/workflows/lint.yml`** - CI lint job
4. **Updated docs** - Developer setup instructions in CLAUDE.md

### Benefits
- Catch issues before commit (fast feedback)
- Consistent code formatting (gofmt)
- Security scanning (gosec)
- Complexity limits (gocyclo)
- CI enforcement (GitHub Actions)
