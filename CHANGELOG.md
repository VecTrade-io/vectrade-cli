# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
