# Changelog

All notable changes to this project will be documented in this file.


## [0.2.0-beta.4](https://github.com/Last-emo-boy/infra-core/compare/v0.2.0-beta.3...v0.2.0-beta.4) (2025-09-27)

## [0.2.0-beta.3](https://github.com/Last-emo-boy/infra-core/compare/v0.2.0-beta.1...v0.2.0-beta.3) (2025-09-27)

## [0.2.0-beta.2](https://github.com/Last-emo-boy/infra-core/compare/v0.2.0-beta.1...v0.2.0-beta.2) (2025-09-27)

## [0.2.0-beta.1](https://github.com/Last-emo-boy/infra-core/compare/v0.2.0-beta.0...v0.2.0-beta.1) (2025-09-25)

## [0.2.0-beta.0](https://github.com/Last-emo-boy/infra-core/compare/v0.1.1-beta.1...v0.2.0-beta.0) (2025-09-25)


### Features

* Add interactive deployment config and enhance setup (no release) ([c3544d8](https://github.com/Last-emo-boy/infra-core/commit/c3544d80c4b73ff5586c7b8380a088c5088056e2))
* integrate multiple mirror sources for reliable Docker builds ([8efd614](https://github.com/Last-emo-boy/infra-core/commit/8efd614896b38694d45966955682efaaa020d158))


### Bug Fixes

* improve Docker build reliability and Alpine package installation ([08048a1](https://github.com/Last-emo-boy/infra-core/commit/08048a1ce0e4d56403292b1999ab204ecd1cf333))
* resolve Docker build and compose configuration issues ([89340d0](https://github.com/Last-emo-boy/infra-core/commit/89340d04ea519789d3629607d20b74a14474e60b))

### [0.1.1-beta.2](https://github.com/Last-emo-boy/infra-core/compare/v0.1.1-beta.1...v0.1.1-beta.2) (2025-09-25)


### Features

* integrate multiple mirror sources for reliable Docker builds ([8efd614](https://github.com/Last-emo-boy/infra-core/commit/8efd614896b38694d45966955682efaaa020d158))


### Bug Fixes

* improve Docker build reliability and Alpine package installation ([08048a1](https://github.com/Last-emo-boy/infra-core/commit/08048a1ce0e4d56403292b1999ab204ecd1cf333))
* resolve Docker build and compose configuration issues ([89340d0](https://github.com/Last-emo-boy/infra-core/commit/89340d04ea519789d3629607d20b74a14474e60b))

### [0.1.1-beta.1](https://github.com/Last-emo-boy/infra-core/compare/v0.1.1-beta.0...v0.1.1-beta.1) (2025-09-24)


### Bug Fixes

* ensure log directory exists before first log output ([bd05944](https://github.com/Last-emo-boy/infra-core/commit/bd05944bab2474ce71ebf47b4ae5317e7ef613ee))

### [0.1.1-beta.0](https://github.com/Last-emo-boy/infra-core/compare/v0.1.0...v0.1.1-beta.0) (2025-09-24)


### Features

* 添加智能发布功能 ([e8f4756](https://github.com/Last-emo-boy/infra-core/commit/e8f4756bd843214097e39826df26b8baf7f88cec))


### Bug Fixes

* resolve lint issues and SARIF format errors in CI/CD ([ae56fcf](https://github.com/Last-emo-boy/infra-core/commit/ae56fcf392315492fc1a0200eda956ce4f42c555))
* update deprecated actions and ensure SARIF files exist ([79efc84](https://github.com/Last-emo-boy/infra-core/commit/79efc84c8ef42102be7c717257e5fd467e3519af))

## [0.1.0] - 2024-01-XX

### Added
- Initial semantic versioning implementation
- Base project structure with multi-cmd architecture
- Core packages: auth, router, services, config, database
- GitHub Actions CI/CD pipeline
- Test infrastructure foundation
- Version control documentation

### Infrastructure
- **Authentication Module**: Core authentication functionality
- **Router Module**: HTTP routing and middleware
- **Services Module**: Health checking and service management
- **Config Module**: Configuration management system
- **Database Module**: Database connection and operations

### Development Workflow
- Conventional commit standards
- Automated changelog generation
- Multi-platform binary builds
- Docker container support
- Security scanning integration
- Code quality enforcement

## Version Control Migration

### Previous Versions (Pre-Semantic)
- `v1` through `v6`: Manual tagging approach (deprecated)

### New Approach (Semantic)
- **0.x.x**: Pre-release development versions
- **0.x.x-beta.y**: Automated beta releases from main branch
- **0.x.x-alpha.y**: Early testing releases
- **1.0.0**: First production-ready release (future)

---

## Release Types

### Manual Releases
- **patch**: Bug fixes and minor improvements
- **minor**: New features (backward compatible)
- **major**: Breaking changes or production readiness
- **alpha**: Early development testing
- **beta**: Feature-complete testing
- **prerelease**: General pre-release increment

### Automated Releases
- **Auto Pre-release**: Generated on every main branch push
- **Format**: `0.x.y-beta.z` (incremental beta versions)
- **Purpose**: Continuous integration and testing builds

## Quality Standards

### Test Coverage
- **Current**: 73.3% average across core packages
- **Minimum**: 60% for releases
- **Target**: 70%+ for production

### Package Coverage Detail
- **auth**: 87.5% (7/8 statements)
- **router**: 95.1% (39/41 statements)  
- **services/health_checker**: 93.4% (28/30 statements)
- **config**: 62.2% (28/45 statements)
- **database**: 27.5% (11/40 statements)

### Security
- Static analysis with gosec
- Dependency vulnerability scanning
- SARIF report generation
- GitHub Security integration

---

*This changelog is automatically updated during releases using standard-version and conventional commits.*