# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Types of changes: `Added`, `Changed`, `Deprecated`, `Removed`, `Fixed`, `Security`.

---

## [1.0.0] - 2026-06-13

### Added

- Initial release
- STUN server probe with NAT type detection (RFC 5389)
- TURN server probe with relay allocation (RFC 5766)
- JSON, XML, and plain-text output formats
- Batch mode for combined STUN + TURN reports
- Default port auto-detection (3478 for STUN/TURN)
- Cross-platform prebuilt binaries (Linux, macOS, Windows, FreeBSD, OpenBSD, NetBSD)
- GitHub Actions CI (build, test, release via GoReleaser)

[1.0.0]: https://github.com/AlchemillaHQ/traverse/releases/tag/v1.0.0
