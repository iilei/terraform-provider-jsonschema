# Release Process

This project uses automated releases via [GoReleaser](https://goreleaser.com/) triggered by Git tags.

## Creating a Release

1. **Ensure all changes are committed and pushed**
   ```bash
   git add .
   git commit -m "feat!: your breaking change description"
   git push origin main
   ```

2. **Create and push a tag** (following [semantic versioning](https://semver.org/))
   ```bash
   # For a new minor version with breaking changes (0.x development)
   git tag -a v0.5.0 -m "v0.5.0"
   
   # Or use a more descriptive message
   git tag -a v0.5.0 -m "Release v0.5.0 - Template variable renames"
   
   # Push the tag
   git push origin v0.5.0
   ```

3. **GitHub Actions automatically:**
   - Builds binaries for all platforms
   - Creates checksums and signs artifacts
   - Generates changelog from conventional commits
   - Creates a GitHub Release with all assets

## Conventional Commit Types

The changelog groups commits by type:

- **Breaking Changes**: Commits with `!` (e.g., `feat!:`, `fix!:`)
- **Features**: `feat:` or `feat(scope):`
- **Bug Fixes**: `fix:` or `fix(scope):`
- **Performance**: `perf:` or `perf(scope):`
- **Refactoring**: `refactor:` or `refactor(scope):`
- **Documentation**: `docs:` or `docs(scope):`

Excluded from changelog:
- `docs:` (unless you want to include them)
- `test:`
- `chore:`
- `ci:`
- Merge commits

## Example Commit Messages

```bash
# Breaking change (appears first in changelog)
git commit -m "feat!: rename template variables for clarity

BREAKING CHANGE: {{.Schema}} → {{.SchemaFile}}, {{.Path}} → {{.DocumentPath}}"

# Feature
git commit -m "feat: add schema traversal examples"

# Bug fix
git commit -m "fix: populate Value field in error details"

# Documentation (excluded from changelog by default)
git commit -m "docs: update migration guide"
```

## Versioning Guidelines (0.x Development)

Since this provider is in **0.x development**:

- **0.y.0** - Minor version bump for any changes (features, fixes, or breaking changes)
- **0.y.z** - Patch version for urgent fixes only
- Breaking changes are allowed at any time during 0.x

Once stable (1.0.0+), follow standard semver:
- **x.0.0** - Breaking changes
- **0.y.0** - New features (backward compatible)
- **0.0.z** - Bug fixes (backward compatible)

## Testing Release Process Locally

```bash
# Dry run (without publishing)
goreleaser release --snapshot --clean

# Check what would be included in the changelog
goreleaser release --skip=publish --skip=validate
```

## Required Secrets

Ensure these GitHub secrets are configured:
- `GPG_PRIVATE_KEY` - Your GPG private key for signing
- `PASSPHRASE` - Passphrase for the GPG key
- `GITHUB_TOKEN` - Automatically provided by GitHub Actions
