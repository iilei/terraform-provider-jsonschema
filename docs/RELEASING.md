# Release Process

## Table of Contents

- [Pre-release Strategy](#pre-release-strategy)
  - [Pre-release Tags](#pre-release-tags)
  - [Terraform Registry Behavior](#terraform-registry-behavior)
  - [Testing Pre-releases](#testing-pre-releases-with-terraform)
  - [Version Constraints](#version-constraints)
  - [CLI Tool Pre-releases](#cli-tool-pre-releases)
- [Release Workflow](#release-workflow)
- [Technical Details](#technical-details)
  - [Goreleaser Configuration](#goreleaser-configuration)
  - [GitHub Actions Workflow](#github-actions-workflow)
- [Release Guidelines](#release-guidelines)
  - [Version Numbering](#version-numbering-guidelines)
  - [Pre-Release Checklist](#checklist-before-tagging)
  - [Local Testing](#local-testing-before-release)
  - [Post-Release Verification](#post-release-verification)
- [Troubleshooting](#troubleshooting)
  - [Rollback Strategy](#rollback-strategy)
- [Communication](#communication)

## Pre-release Strategy

This project uses semantic versioning with pre-release identifiers for safe rollout of new features.

### Pre-release Tags

Pre-release versions are automatically detected and marked as pre-releases in GitHub:

```bash
# Alpha releases (early testing, unstable)
git tag v0.6.0-alpha.1
git push origin v0.6.0-alpha.1

# Beta releases (feature complete, testing)
git tag v0.6.0-beta.1
git push origin v0.6.0-beta.1

# Release candidates (stable, final testing)
git tag v0.6.0-rc.1
git push origin v0.6.0-rc.1

# Stable release
git tag v0.6.0
git push origin v0.6.0
```

### Terraform Registry Behavior

**Important:** The Terraform Registry **only indexes stable releases** (no pre-release suffix).

- ✅ `v0.6.0` → Published to registry, users can install with `version = "0.6.0"`
- ❌ `v0.6.0-beta.1` → Available on GitHub Releases only, **NOT** in Terraform Registry
- ❌ `v0.6.0-rc.1` → Available on GitHub Releases only, **NOT** in Terraform Registry

### Testing Pre-releases with Terraform

Users can test pre-release versions by installing directly from GitHub:

**Option 1: Manual download from GitHub Releases**

```bash
# Download pre-release binary from GitHub Releases
# Replace OS/ARCH as needed: linux_amd64, darwin_amd64, darwin_arm64, windows_amd64, etc.
wget https://github.com/iilei/terraform-provider-jsonschema/releases/download/v0.6.0-beta.1/terraform-provider-jsonschema_0.6.0-beta.1_linux_amd64.tar.gz

# Extract and install to Terraform plugin directory
tar -xzf terraform-provider-jsonschema_0.6.0-beta.1_linux_amd64.tar.gz
mkdir -p ~/.terraform.d/plugins/registry.terraform.io/iilei/jsonschema/0.6.0-beta.1/linux_amd64/
mv terraform-provider-jsonschema_v0.6.0-beta.1 ~/.terraform.d/plugins/registry.terraform.io/iilei/jsonschema/0.6.0-beta.1/linux_amd64/

# Use in Terraform (note: pre-release version must be exact)
# terraform {
#   required_providers {
#     jsonschema = {
#       source  = "iilei/jsonschema"
#       version = "0.6.0-beta.1"  # Exact pre-release version
#     }
#   }
# }
```

**Option 2: Local development override**

Create or update `~/.terraformrc` (or `%APPDATA%/terraform.rc` on Windows):

```hcl
provider_installation {
  dev_overrides {
    "iilei/jsonschema" = "/path/to/your/local/build"
  }
  direct {}
}
```

Then build locally:

```bash
go build -o terraform-provider-jsonschema
```

**Important:** Dev overrides bypass version checking. Terraform will use your local binary regardless of version constraints.

### Version Constraints

Users can pin to specific versions using Terraform version constraints:

```terraform
terraform {
  required_providers {
    jsonschema = {
      source  = "iilei/jsonschema"
      # Stable releases only
      version = "~> 0.5.0"     # 0.5.x
      version = ">= 0.5.0, < 0.6.0"
      version = "0.5.0"        # Exact version
    }
  }
}
```

**Pre-releases are NOT resolved by version constraints:**
- `version = "~> 0.6.0"` will NOT match `v0.6.0-beta.1`
- Users must manually install pre-releases for testing

### CLI Tool Pre-releases

The CLI tool (`jsonschema-validator`) follows the same versioning:

```bash
# Install stable version
go install github.com/iilei/terraform-provider-jsonschema/cmd/jsonschema-validator@latest

# Install specific pre-release
go install github.com/iilei/terraform-provider-jsonschema/cmd/jsonschema-validator@v0.6.0-beta.1

# Install specific stable version
go install github.com/iilei/terraform-provider-jsonschema/cmd/jsonschema-validator@v0.5.0
```

### Release Workflow

1. **Development Phase**
   - Work on feature branch
   - Merge to `main` via PR with tests passing

2. **Alpha Release** (optional, for early feedback)
   ```bash
   git tag v0.6.0-alpha.1
   git push origin v0.6.0-alpha.1
   ```
   - GitHub Release created automatically (marked as pre-release)
   - NOT published to Terraform Registry
   - Manual testing by early adopters

3. **Beta Release** (feature complete, testing)
   ```bash
   git tag v0.6.0-beta.1
   git push origin v0.6.0-beta.1
   ```
   - GitHub Release created automatically (marked as pre-release)
   - NOT published to Terraform Registry
   - Broader testing with real-world scenarios

4. **Release Candidate** (final testing)
   ```bash
   git tag v0.6.0-rc.1
   git push origin v0.6.0-rc.1
   ```
   - GitHub Release created automatically (marked as pre-release)
   - NOT published to Terraform Registry
   - Final validation before stable release

5. **Stable Release**
   ```bash
   git tag v0.6.0
   git push origin v0.6.0
   ```
   - GitHub Release created automatically (stable release)
   - **Published to Terraform Registry** (automatic webhook)
   - Available to all users via version constraints

## Technical Details

### Goreleaser Configuration

The `.goreleaser.yml` automatically handles pre-releases:

```yaml
release:
  prerelease: auto  # Detects -alpha, -beta, -rc suffixes
  make_latest: false  # Don't mark pre-releases as latest
```

Pre-release detection regex: `v[0-9]+\.[0-9]+\.[0-9]+-(alpha|beta|rc)`

### GitHub Actions Workflow

The release workflow (`.github/workflows/release.yml`):

1. Triggered on any `v*` tag push
2. Detects if tag is pre-release via regex
3. Runs goreleaser with appropriate flags
4. Creates GitHub Release (marked as pre-release if applicable)
5. Terraform Registry webhook picks up **stable releases only**

### Rollback Strategy

If a stable release has critical issues:

1. **Immediately tag a patch release** (recommended):
   ```bash
   # Fix the issue in main
   git tag v0.6.1
   git push origin v0.6.1
   ```

2. **Delete the broken release** (last resort):
   ```bash
   # Delete tag locally and remotely
   git tag -d v0.6.0
   git push origin :refs/tags/v0.6.0

   # Delete GitHub Release manually (not via Terraform Registry)
   ```

   **Warning:** Deleting a published Terraform Registry version is not supported. Always prefer patching forward.

### Troubleshooting

**Problem: Goreleaser fails with "dirty git state"**

```bash
# Check git status
git status

# Ensure all changes committed
git add .
git commit -m "prepare release"

# Ensure working on correct branch
git checkout main
```

**Problem: Release created but not in Terraform Registry**

- **Pre-release versions**: Registry only indexes stable releases (no `-alpha`, `-beta`, `-rc`)
- **Wait time**: Can take up to 15 minutes for registry webhook to process
- **Check logs**: View GitHub Actions logs for goreleaser run
- **Verify signing**: Ensure GPG signature created (required by registry)

**Problem: CLI tool not installable via `go install`**

```bash
# Verify tag exists
git ls-remote --tags origin | grep v0.6.0

# Verify module path correct
go install github.com/iilei/terraform-provider-jsonschema/cmd/jsonschema-validator@v0.6.0

# Clear Go module cache if stale
go clean -modcache
```

**Problem: Pre-commit hook fails to install**

```bash
# Verify Go installed
go version

# Verify repository accessible
git clone https://github.com/iilei/terraform-provider-jsonschema

# Clean pre-commit cache
pre-commit clean
pre-commit uninstall

# Reinstall
pre-commit install
pre-commit run jsonschema-validator --all-files
```

**Problem: Checksum mismatch or signature verification failed**

- **Cause**: Binary built without `-trimpath` flag
- **Fix**: Already configured in `.goreleaser.yml` with `flags: [-trimpath]`
- **Verify**: Check goreleaser config has trimpath for both builds
- **Test locally**: `goreleaser build --snapshot --clean` and verify binaries

**Problem: Archive naming mismatch**

- **Provider**: Should be `terraform-provider-jsonschema_VERSION_OS_ARCH.tar.gz`
- **CLI**: Should be `jsonschema-validator_VERSION_OS_ARCH.tar.gz`
- **Fix**: Check `.goreleaser.yml` archives section uses correct `name_template`
- **Verify**: Check `{{ .Env.CLI_NAME }}` environment variable set correctly

### Communication

When releasing pre-releases:

1. **GitHub Release Notes**: Clearly mark as pre-release, include testing instructions
2. **Changelog**: Document breaking changes and migration paths
3. **Discussions/Issues**: Announce pre-release, request feedback
4. **README**: Update with pre-release installation instructions

### Example Timeline

```
Week 1: v0.6.0-alpha.1  → Internal testing
Week 2: v0.6.0-beta.1   → Community testing
Week 3: v0.6.0-rc.1     → Final validation
Week 4: v0.6.0          → Stable release → Terraform Registry
```

## Release Guidelines

### Version Numbering Guidelines

**Stable releases (0.x.x):**
- `0.x.0` - Minor version bump (new features, may break compatibility in 0.x)
- `0.x.y` - Patch version bump (bug fixes, no breaking changes)
- `1.0.0` - First stable release (semver guarantees begin)

**Pre-releases:**
- `-alpha.N` - Early development, unstable, frequent changes
- `-beta.N` - Feature complete, testing phase, API may change
- `-rc.N` - Release candidate, stable, only critical fixes

**Note:** During 0.x development, breaking changes may occur in ANY release (per semver spec).
Pin exact versions in production: `version = "0.5.0"` (not `~> 0.5.0`).

### Checklist Before Tagging

**Before creating any release tag:**

1. ✅ All tests passing (`go test ./...`)
2. ✅ Acceptance tests passing (`TF_ACC=1 go test ./internal/provider/ -v`)
3. ✅ Code coverage maintained or improved
4. ✅ CHANGELOG.md updated with changes
5. ✅ README.rst updated if new features added
6. ✅ Examples updated/added if behavior changed
7. ✅ Documentation updated (docs/*.md)
8. ✅ Version constraints tested (`version = "~> 0.x.0"`)
9. ✅ Breaking changes clearly documented
10. ✅ Migration guide provided (if breaking changes)

**Before stable release (no pre-release suffix):**

11. ✅ Pre-release testing completed (alpha/beta/rc as needed)
12. ✅ No known critical bugs
13. ✅ Community feedback addressed
14. ✅ Release notes drafted in GitHub

### Local Testing Before Release

Test the release process locally before pushing tags:

```bash
# Test goreleaser configuration
goreleaser check

# Build snapshot (doesn't require tag)
goreleaser build --snapshot --clean

# Test provider binary
./dist/terraform-provider-jsonschema_linux_amd64_v1/terraform-provider-jsonschema_v0.6.0

# Test CLI binary
./dist/jsonschema-validator_linux_amd64_v1/jsonschema-validator --version

# Full release dry-run (requires clean git state)
goreleaser release --snapshot --clean
```

### Post-Release Verification

After pushing a tag and release completes:

1. **GitHub Release**: Verify release created with correct assets
   - Provider archives (`.tar.gz`, `.zip`)
   - CLI archives (`.tar.gz`, `.zip`)
   - SHA256SUMS files (2 separate: provider + CLI)
   - GPG signatures (`.sig` files)

2. **Terraform Registry**: Verify stable releases appear (within ~15 minutes)
   - Check https://registry.terraform.io/providers/iilei/jsonschema/latest
   - Verify version number correct
   - Test installation: `terraform init` in example project

3. **CLI Installation**: Verify `go install` works
   ```bash
   go install github.com/iilei/terraform-provider-jsonschema/cmd/jsonschema-validator@v0.6.0
   jsonschema-validator --version
   ```

4. **Pre-commit Hook**: Verify golang language auto-install works
   ```bash
   # In test repo with .pre-commit-config.yaml
   pre-commit clean
   pre-commit run jsonschema-validator --all-files
   ```
