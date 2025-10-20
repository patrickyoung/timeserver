# Version Management Guide

This document describes the version management practices for the Time Service project to prevent version misconfigurations that can break builds.

## Overview

The project uses multiple version identifiers across different files:
- **go.mod**: Go language version requirement
- **Dockerfile**: Go compiler and Alpine base image versions
- **.github/workflows/ci.yml**: CI test matrix Go versions
- **README.md**: Documentation of image versions

All these must be kept consistent and use **real, released versions only**.

## Version Files Reference

| File | Purpose | Example |
|------|---------|---------|
| `go.mod` | Go language version | `go 1.24` |
| `Dockerfile` | Build and runtime images | `FROM golang:1.24-alpine` |
| `.github/workflows/ci.yml` | CI test matrix | `go-version: ['1.22', '1.23', '1.24']` |
| `README.md` | Documentation | References to image versions |

## Rules

### 1. Use Only Released Versions

❌ **WRONG:**
```dockerfile
FROM golang:1.24.8-alpine3.21  # Version 1.24.8 doesn't exist
```

✅ **CORRECT:**
```dockerfile
FROM golang:1.24-alpine  # Use major.minor tag or verify exact version exists
```

**How to verify:**
- Go versions: https://go.dev/dl/
- Docker golang images: https://hub.docker.com/_/golang/tags
- Docker alpine images: https://hub.docker.com/_/alpine/tags

### 2. Avoid Toolchain Directives

❌ **WRONG:**
```go
go 1.24.0
toolchain go1.24.8  // Can cause build failures
```

✅ **CORRECT:**
```go
go 1.24.0  // Go will auto-select appropriate toolchain
```

The `toolchain` directive should be avoided as it forces a specific toolchain version that may not exist.

### 3. Keep CI Matrix in Sync

The CI workflow must test the version specified in go.mod:

```yaml
# go.mod specifies: go 1.24
strategy:
  matrix:
    go-version: ['1.22', '1.23', '1.24']  # Must include 1.24
```

### 4. Use Stable Alpine Versions

Use released Alpine versions (3.19, 3.20, etc.), not unreleased ones:

❌ **WRONG:** `alpine:3.21.0` (when 3.21 isn't released yet)
✅ **CORRECT:** `alpine:3.20`

## Automated Validation

The project includes automated validation to catch version issues early:

### 1. Validation Script

Run manually:
```bash
./scripts/validate-versions.sh
```

This checks:
- ✓ go.mod version format is valid
- ✓ Dockerfile image tags are valid
- ✓ CI includes go.mod version
- ✓ Docker images exist on Docker Hub
- ✓ No invalid toolchain directives
- ✓ README documentation matches

### 2. Make Target

The validation runs automatically before builds:
```bash
make validate-versions  # Manual run
make build              # Validates, then builds
make docker             # Validates, then builds image
make ci-local           # Validates as part of CI checks
```

### 3. Pre-commit Hook

Install pre-commit hooks:
```bash
pip install pre-commit
pre-commit install
```

The validation runs automatically when committing changes to:
- `go.mod`
- `Dockerfile`
- `.github/workflows/*.yml`
- `README.md`

### 4. CI Pipeline

Every push/PR automatically validates versions before running tests:
- Job: `validate-versions`
- Runs first, before all other jobs
- Fails fast if versions are invalid

## Updating Versions

### Updating Go Version

**When**: New Go release or upgrading project

**Steps:**

1. **Check available versions:**
   ```bash
   # Visit https://go.dev/dl/
   # Or check Docker Hub: https://hub.docker.com/_/golang/tags
   ```

2. **Update go.mod:**
   ```bash
   # Edit go.mod
   go 1.24  # Use X.Y or X.Y.Z format

   # Verify it works
   go mod tidy
   go build ./...
   ```

3. **Update Dockerfile:**
   ```dockerfile
   FROM golang:1.24-alpine AS builder
   ```

   Verify image exists:
   ```bash
   docker pull golang:1.24-alpine
   ```

4. **Update CI workflow:**
   ```yaml
   strategy:
     matrix:
       go-version: ['1.22', '1.23', '1.24']  # Include new version
   ```

   Also update standalone jobs:
   ```yaml
   lint:
     steps:
       - uses: actions/setup-go@v5
         with:
           go-version: '1.24'  # Use latest stable
   ```

5. **Update README.md:**
   - Search for old version references
   - Update Docker image version examples

6. **Validate:**
   ```bash
   make validate-versions
   ```

7. **Test locally:**
   ```bash
   make build
   make docker
   make ci-local
   ```

8. **Commit:**
   ```bash
   git add go.mod Dockerfile .github/workflows/ci.yml README.md
   git commit -m "chore: upgrade Go to 1.24"
   ```

### Updating Alpine Version

**When**: Security updates or new Alpine release

**Steps:**

1. **Check available versions:**
   ```bash
   # Visit https://hub.docker.com/_/alpine/tags
   # Or: docker pull alpine:3.21
   ```

2. **Update Dockerfile:**
   ```dockerfile
   FROM alpine:3.21  # Runtime image
   ```

3. **Verify Go image compatibility:**
   ```bash
   # Check if golang:<version>-alpine<alpine-version> exists
   docker pull golang:1.24-alpine3.21
   ```

   If specific combination doesn't exist, use:
   ```dockerfile
   FROM golang:1.24-alpine  # Uses latest compatible Alpine
   ```

4. **Validate and test:**
   ```bash
   make validate-versions
   make docker
   ```

## Troubleshooting

### Error: "go.mod requires go >= X.Y.Z (running go X.Y.Z)"

**Cause:** Dockerfile Go image version is lower than go.mod requirement

**Fix:**
```bash
# Check go.mod
grep '^go ' go.mod  # e.g., "go 1.24.0"

# Update Dockerfile to match or higher
FROM golang:1.24-alpine  # Must be >= 1.24
```

### Error: "Docker image not found: golang:X.Y.Z-alpineA.B.C"

**Cause:** Specified image tag doesn't exist on Docker Hub

**Fix:**
```bash
# Check available tags: https://hub.docker.com/_/golang/tags
# Use a tag that exists:
FROM golang:1.24-alpine  # Generic alpine (recommended)
# OR
FROM golang:1.24-alpine3.20  # Specific alpine version (verify exists)
```

### Error: "CI workflow does not test go.mod version"

**Cause:** CI matrix doesn't include version from go.mod

**Fix:**
```yaml
# go.mod has: go 1.24
strategy:
  matrix:
    go-version: ['1.22', '1.23', '1.24']  # Add 1.24
```

### Error: "updates to go.mod needed; to update it: go mod tidy"

**Cause:** go.mod version conflicts with dependencies or toolchain

**Fix:**
```bash
# Remove toolchain directive if present
sed -i '/^toolchain/d' go.mod

# Run go mod tidy
go mod tidy

# Verify
make validate-versions
```

## Best Practices

### ✅ DO

- **Use major.minor format** in go.mod: `go 1.24` (not `go 1.24.8`)
- **Use generic alpine tags** in Dockerfile: `golang:1.24-alpine`
- **Test locally** before pushing: `make validate-versions && make ci-local`
- **Check Docker Hub** before using image tags
- **Keep CI matrix current** with at least 3 recent Go versions
- **Run validation** as part of pre-commit workflow

### ❌ DON'T

- **Use unreleased versions** (e.g., Go 1.99, Alpine 3.99)
- **Add toolchain directives** unless absolutely necessary
- **Skip validation** when updating versions
- **Mix version formats** (be consistent across files)
- **Assume versions exist** without verification
- **Update one file** without updating others

## Checklist for Version Updates

Before committing version changes:

- [ ] Verified Go version exists: https://go.dev/dl/
- [ ] Verified Docker golang image exists: `docker pull golang:X.Y-alpine`
- [ ] Verified Docker alpine image exists: `docker pull alpine:X.Y`
- [ ] Updated go.mod
- [ ] Updated Dockerfile (both golang and alpine images)
- [ ] Updated .github/workflows/ci.yml (matrix + standalone jobs)
- [ ] Updated README.md version references
- [ ] Removed any toolchain directives from go.mod
- [ ] Ran `make validate-versions` (passed)
- [ ] Ran `make build` (passed)
- [ ] Ran `make docker` (passed)
- [ ] Ran `go mod tidy` (no errors)
- [ ] Pre-commit hooks passed

## Additional Resources

- **Go Releases**: https://go.dev/dl/
- **Go Release History**: https://go.dev/doc/devel/release
- **Docker Golang Images**: https://hub.docker.com/_/golang/tags
- **Docker Alpine Images**: https://hub.docker.com/_/alpine/tags
- **Alpine Release Branches**: https://alpinelinux.org/releases/
- **GitHub Actions setup-go**: https://github.com/actions/setup-go

## Version History

| Date | Change | Reason |
|------|--------|--------|
| 2025-10-20 | Created version management guide | Prevent recurring version mismatch issues |
| 2025-10-20 | Added validation script & automation | Catch issues early in development |

---

**Remember**: Version consistency is critical for builds. When in doubt, run `make validate-versions` before committing!
