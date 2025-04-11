# Release Process

This project uses semantic-release to automate version management and release publishing.

## Automated Release with semantic-release

The release process is fully automated using semantic-release. It will:

1. Analyze commit messages to determine the version bump
2. Update version references in the codebase
3. Generate a changelog
4. Create a Git tag
5. Push a GitHub release

### How it works

1. When commits are pushed to the `main` branch, the semantic-release workflow is triggered
2. The version is automatically determined based on commit messages:
   - `fix:` or `fix(scope):` → patch release
   - `feat:` or `feat(scope):` → minor release
   - `BREAKING CHANGE:` in the commit body → major release
3. The workflow will update all necessary files with the new version and create a tagged release

### Commit Message Format

To properly trigger version updates, follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

Examples:

- `fix: resolve null pointer in validation function` (triggers patch version)
- `feat: add support for environment variables` (triggers minor version)
- `feat!: remove deprecated API endpoints` (triggers major version)
- ```
  feat: add new parameter to API

  BREAKING CHANGE: this changes the behavior of existing integrations
  ```

  (triggers major version)

## Manual Release Process (Legacy)

If you need to release manually:

1. Update the version in:
   - `plugin.yaml`
   - `scripts/install_version.sh`
2. Commit and push these changes
3. Create and push a tag:
   ```
   git tag -a v1.2.3 -m "Release v1.2.3"
   git push origin v1.2.3
   ```

When a tag is pushed, the `release.yml` workflow will trigger, which:

1. Verifies the version in the files matches the tag
2. Builds and publishes the release artifacts

## Version Numbering

This project follows Semantic Versioning (SemVer):

- **MAJOR** version when making incompatible API changes
- **MINOR** version when adding functionality in a backward compatible manner
- **PATCH** version when making backward compatible bug fixes
