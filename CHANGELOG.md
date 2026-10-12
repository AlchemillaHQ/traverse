# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Types of changes: `Added`, `Changed`, `Deprecated`, `Removed`, `Fixed`, `Security`.

---

## [Unreleased]

## [1.0.1] - 2026-10-12

### Fixed

- Use an unconnected UDP socket and start inbound processing before TURN allocation.
- Use an actual TCP connection for TCP TURN transport.
- Honor probe timeouts and context cancellation during DNS, connection setup,
  and pending STUN/TURN operations.
- Reject STUN responses with mismatched transaction IDs.
- Return failure exit status for malformed flags and unexpected arguments.
- Reject invalid local IP addresses instead of silently ignoring them.
- Validate permissions and responses in the relayed connectivity probe.

### Changed

- Report NAT type as `unknown` instead of inferring unsupported NAT behavior
  from a single binding response.
- Report TURN allocation lifetime as `unknown` instead of inventing a default.
- Document the actual Go 1.26.2 minimum and current IPv4 TURN limitation.

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

[Unreleased]: https://github.com/AlchemillaHQ/traverse/compare/v1.0.1...HEAD
[1.0.1]: https://github.com/AlchemillaHQ/traverse/releases/tag/v1.0.1
[1.0.0]: https://github.com/AlchemillaHQ/traverse/releases/tag/v1.0.0
