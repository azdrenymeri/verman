# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2025-01-25

### Added

- **Core Version Management**
  - `verman install <lang> <version>` - Install specific versions of languages/tools
  - `verman use <lang> <version>` - Switch to a specific installed version
  - `verman list <lang>` - List installed versions
  - `verman list <lang> --all` - List all available versions from remote
  - `verman current [lang]` - Show currently active versions
  - `verman uninstall <lang> <version>` - Remove an installed version
  - `verman which <lang>` - Show path to active version

- **Supported Languages/Tools**
  - Java (Eclipse Adoptium Temurin)
  - Node.js
  - Go
  - Scala (2.x and 3.x)
  - Kotlin
  - Maven
  - Gradle
  - SBT
  - Mill

- **Version Detection**
  - `verman detect` - Auto-detect versions from project files
  - `verman detect --apply` - Detect and switch to detected versions
  - Supports: `.java-version`, `.nvmrc`, `.node-version`, `.go-version`, `go.mod`, `.mill-version`

- **Shell Integration**
  - `verman init` - Display shell integration commands
  - `verman init --install` - Install shell integration to PowerShell profile
  - `verman env` - Output environment variable commands for current shell
  - PowerShell wrapper script for session-local environment changes

- **System Setup**
  - `verman setup` - Initialize verman directory structure
  - `verman doctor` - Diagnose common issues with verman setup
  - Windows junction points for version switching (no admin required)
  - Global environment variable persistence via Windows Registry

- **Download Features**
  - Progress indicator with percentage, speed, and ETA
  - Support for both zip archives and single-file downloads
  - Automatic extraction with pattern matching

- **Extensible Architecture**
  - JSON-based source definitions for languages
  - User-customizable sources in `~/.verman/sources/`
  - Post-install command support

### Technical Details

- Written in Go with Cobra CLI framework
- Windows-focused design using native APIs
- No administrator privileges required for normal operation
- GitHub Actions CI/CD pipeline
- Scoop package manager support

### Known Limitations

- Windows only (by design)
- Checksum verification not yet implemented
- No automatic updates

[0.1.0]: https://github.com/azdren/verman/releases/tag/v0.1.0
