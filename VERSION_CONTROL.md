# Version Control Strategy

## Overview
This project uses **semantic versioning (SemVer)** with a **0.x.x pre-release** strategy for development and **automated release management** through GitHub Actions.

## Version Format
- **Pre-release (Development)**: `0.x.y` or `0.x.y-beta.z`
- **Production (Stable)**: `1.x.y` (when project reaches production readiness)

### Current Version Strategy
```
0.1.0          # Current baseline
0.1.1-beta.0   # Auto pre-release from main branch
0.1.1-beta.1   # Next beta iteration
0.2.0          # Minor feature release
1.0.0          # First production release
```

## Release Types

### 1. Manual Releases (workflow_dispatch)
Triggered manually through GitHub Actions:

#### Pre-release Options:
- **`alpha`**: `0.1.0` → `0.1.1-alpha.0` (earliest testing)
- **`beta`**: `0.1.0` → `0.1.1-beta.0` (feature complete)
- **`prerelease`**: General pre-release bump

#### Stable Options:
- **`patch`**: `0.1.0` → `0.1.1` (bug fixes)
- **`minor`**: `0.1.0` → `0.2.0` (new features)
- **`major`**: `0.x.x` → `1.0.0` (production ready / breaking changes)

### 2. Automated Pre-releases
- **Trigger**: Every push to `main` branch (except release commits)
- **Format**: `0.x.y-beta.z` (auto-incrementing beta versions)
- **Purpose**: Continuous integration testing and development builds
- **Marked**: As pre-release in GitHub

## Workflow Rules

### Automatic Triggers
1. **Push to main**: Creates auto pre-release (`beta` versions)
2. **Pull requests**: Runs tests and quality checks only
3. **Manual dispatch**: Creates controlled releases based on input

### Release Gates
All releases require:
- ✅ Tests passing (60% coverage minimum)
- ✅ Code quality checks (golangci-lint)
- ✅ Security scans (gosec)
- ✅ Multi-platform builds (Linux, Windows, macOS)

## Branch Strategy

### Main Branch (`main`)
- **Purpose**: Stable development line
- **Auto-release**: Beta pre-releases on every push
- **Manual release**: Available for all release types
- **Protection**: Pull request required

### Develop Branch (`develop`)
- **Purpose**: Feature integration and testing
- **Auto-release**: Docker images only
- **Manual release**: Not available
- **Protection**: Testing and quality checks

## Version Migration History

### Previous Approach (v1-v6)
```
v1, v2, v3, v4, v5, v6  # Ad-hoc versioning
```

### Current Approach (Semantic)
```
0.1.0          # Starting point
0.1.1-beta.0   # First beta
0.2.0          # Feature release
1.0.0          # Production ready
```

## Implementation Details

### Configuration Files
- **`VERSION`**: Current version (0.1.0)
- **`.versionrc.json`**: standard-version configuration
- **`CHANGELOG.md`**: Auto-generated release notes
- **`.github/workflows/release.yml`**: CI/CD pipeline

### Version Control Tools
- **standard-version**: Automated version bumping and changelog
- **conventional commits**: Semantic commit messages
- **GitHub Actions**: Automated release pipeline
- **semantic versioning**: Industry standard version format

## Best Practices

### Commit Messages
Use conventional commits format:
```bash
feat: add new authentication method
fix: resolve memory leak in health checker
docs: update API documentation
chore: update dependencies
```

### Release Process
1. **Development**: Work on feature branches
2. **Integration**: Merge to `develop` via PR
3. **Testing**: Merge to `main` → auto beta release
4. **Production**: Manual release when ready

### Version Progression
```
Development Cycle:
0.1.0 → 0.1.1-beta.0 → 0.1.1-beta.1 → 0.1.1 (stable)

Feature Release:
0.1.1 → 0.2.0-beta.0 → 0.2.0-beta.1 → 0.2.0 (stable)

Production Ready:
0.9.0 → 1.0.0-rc.0 → 1.0.0-rc.1 → 1.0.0 (production)
```

## Quality Gates

### Coverage Requirements
- **Minimum**: 60% test coverage
- **Target**: 70%+ for production releases
- **Tracking**: Codecov integration

### Security Standards
- **Static Analysis**: gosec security scanner
- **Dependency Check**: GitHub security advisories
- **SARIF Reports**: Uploaded to GitHub Security tab

### Build Validation
- **Platforms**: Linux, Windows, macOS
- **Architectures**: amd64, arm64
- **Format**: Compressed binaries and Docker images

## Troubleshooting

### Common Issues
1. **Version conflicts**: Check `VERSION` file matches git tags
2. **Release failures**: Verify all quality gates pass
3. **Permission errors**: Ensure `GITHUB_TOKEN` has sufficient permissions

### Recovery Procedures
1. **Failed release**: Fix issues and re-run workflow
2. **Wrong version**: Manually update `VERSION` file and re-tag
3. **Missing assets**: Re-run build job and attach to release

## Future Considerations

### Production Readiness Checklist
- [ ] Complete API documentation
- [ ] Performance benchmarks
- [ ] Security audit
- [ ] Production deployment guide
- [ ] Monitoring and alerting setup
- [ ] Backup and recovery procedures

When these items are complete, the project will be ready for `1.0.0` production release.