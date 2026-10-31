# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.1](https://github.com/VecTrade-io/vectrade-cli/compare/v0.2.0...v0.2.1) (2026-10-31)


### Bug Fixes

* CI golangci-lint-action v6 compat, windows build, pipe test, govulncheck non-fatal ([64ac06a](https://github.com/VecTrade-io/vectrade-cli/commit/64ac06a3cfcda089893e594320a3ab6dece48338))
* **ci:** pin GoReleaser v2.9.0 and Go 1.26.10; make govulncheck blocking; test updates ([bdb3508](https://github.com/VecTrade-io/vectrade-cli/commit/bdb3508d6ece750368c72e9bbc02ae5f5cd97bac))
* **ci:** resolve lint, test isolation, and goreleaser config issues ([b2a4e9f](https://github.com/VecTrade-io/vectrade-cli/commit/b2a4e9fc8fbb446e4a7d8cffd0efd46b6b5751d1))
* use github-native changelog to prevent email disclosure ([3dd683c](https://github.com/VecTrade-io/vectrade-cli/commit/3dd683c1164c0ffbe26df4b3afd3446c333c66ec))

## [Unreleased]

### Changed

- Minimum Go version is 1.25; CI tests Go 1.25 and 1.26 and lints with golangci-lint v2. Builds from Go 1.22 crashed on current macOS (`missing LC_UUID`).
- `vectrade webhook listen` now closes the event stream and in-flight forwards when it stops.

### Fixed

- On Windows, `auth login` opens the browser without passing the URL through `cmd.exe`, which could interpret `&` in the query string.

### Added

- Initial release

### Changed

- Pin GoReleaser v2.9.0 to preserve current Homebrew formula and Docker manifest behavior pending a planned distribution migration; build releases and run security scans on patched Go 1.26.10.
