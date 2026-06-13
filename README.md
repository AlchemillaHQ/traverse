# traverse

A fast and easy to use STUN and TURN server probe with structured output for humans and machines.

---

## Quick start

```bash
# install (requires Go 1.21+)
go install github.com/AlchemillaHQ/traverse@latest

# probe a STUN server
traverse -s stun.l.google.com:19302

# omit the port - defaults to 3478 (RFC 5389 / 5766)
traverse -s stun.example.com

# JSON for scripting
traverse -s stun.example.com -o json | jq .

# test a TURN server
traverse -t turn.example.com -u alice -p s3cret

# batch report (both STUN + TURN, XML output)
traverse -s stun.example.com -t turn.example.com -u alice -p s3cret -o xml -b
```

## Install

### Binary downloads (no Go required)

Grab a prebuilt binary from the [releases page](https://github.com/AlchemillaHQ/traverse/releases).

| OS | Architectures |
|---|---|
| **Linux** | amd64, 386, arm64, arm (v6, v7), ppc64le, s390x, mips64le, riscv64 |
| **macOS** | amd64, arm64 (Apple Silicon) |
| **Windows** | amd64, 386, arm64 |
| **FreeBSD** | amd64, 386, arm64 |
| **OpenBSD** | amd64, arm64 |
| **NetBSD** | amd64, 386, arm64 |

```bash
# Linux amd64 example
curl -L -o traverse.tar.gz https://github.com/AlchemillaHQ/traverse/releases/download/v1.0.0/traverse_1.0.0_linux_amd64.tar.gz
tar xzf traverse.tar.gz
./traverse -h
```

Every tag `v*` pushed to this repo triggers a [GoReleaser](https://goreleaser.com) workflow that
cross-compiles the binary and attaches tarballs, .zip files, and a checksums.txt to the release automatically.

### `go install` (recommended for Go users)

```bash
go install github.com/AlchemillaHQ/traverse@latest
```

The binary lands in `$GOPATH/bin` (or `$GOBIN`). Make sure that directory is on your `$PATH`:

```bash
export PATH="$HOME/go/bin:$PATH"
```

### Build from source

```bash
git clone https://github.com/AlchemillaHQ/traverse.git
cd traverse
go build -o traverse .
./traverse -h
```

## Usage

```
traverse [options]
```

### Options

| Flag | Alias | Description | Default |
|---|---|---|---|
| `--stun-server` | `-s` | STUN server address (`host[:port]`) | - |
| `--turn-server` | `-t` | TURN server address (`host[:port]`) | - |
| `--username` | `-u` | TURN username (**required** for TURN) | - |
| `--password` | `-p` | TURN password (**required** for TURN) | - |
| `--protocol` | `-P` | TURN transport (`udp` or `tcp`) | `udp` |
| `--output` | `-o` | Output format (`text`, `json`, `xml`) | `text` |
| `--batch` | `-b` | Combined JSON/XML batch report | - |
| `--timeout` | `-T` | Request timeout in seconds | `5` |
| `--local-ip` | `-l` | Bind to a specific local IP | - |
| `--version` | `-v` | Print version and exit | - |
| `--help` | `-h` | Show help | - |

If no port is given, STUN defaults to `3478` and TURN defaults to `3478`.

### Examples

```bash
# basic STUN
traverse -s stun.l.google.com:19302

# STUN with JSON output
traverse -s stun.l.google.com:19302 -o json

# STUN with XML output
traverse -s stun.l.google.com:19302 -o xml

# TURN with explicit port
traverse -t turn.example.com:3478 -u alice -p s3cret

# TURN over TCP
traverse -t turn.example.com:3478 -u alice -p s3cret -P tcp

# custom timeout
traverse -s stun.example.com -T 10

# bind to a specific local interface
traverse -s stun.example.com -l 10.0.0.5

# batch: test both STUN and TURN, output as JSON
traverse -s stun.example.com -t turn.example.com -u alice -p s3cret -o json -b

# batch: XML report
traverse -s stun.example.com -t turn.example.com -u alice -p s3cret -o xml -b
```

## Output formats

### Text (default)

```
=== STUN Test Result ===
Timestamp:  2026-06-13T18:07:10+05:30
Server:     stun.l.google.com:19302
Duration:   5.5ms
Status:     SUCCESS
Public IP:  203.0.113.42
Public Port:49152
Local IP:   10.0.0.5
NAT Type:   symmetric
------------------------
```

### JSON

```json
{
  "success": true,
  "server": "stun.l.google.com:19302",
  "duration": "5.5ms",
  "public_ip": "203.0.113.42",
  "public_port": 49152,
  "nat_type": "symmetric",
  "local_ip": "10.0.0.5",
  "timestamp": "2026-06-13T18:07:10+05:30"
}
```

### XML

```xml
<STUNResult>
  <success>true</success>
  <server>stun.l.google.com:19302</server>
  <duration>5.5ms</duration>
  <PublicIP>203.0.113.42</PublicIP>
  <PublicPort>49152</PublicPort>
  <NATType>symmetric</NATType>
  <LocalIP>10.0.0.5</LocalIP>
  <timestamp>2026-06-13T18:07:10+05:30</timestamp>
</STUNResult>
```

### Batch report (JSON)

```json
{
  "generated_at": "2026-06-13T18:10:00+05:30",
  "stun_results": [
    { "success": true, "server": "...", "public_ip": "203.0.113.42", "nat_type": "symmetric" }
  ],
  "turn_results": [
    { "success": true, "server": "...", "relayed_ip": "198.51.100.10", "protocol": "udp" }
  ],
  "summary": { "total_tests": 2, "successful": 2, "failed": 0 }
}
```

## Exit codes

| Code | Meaning |
|---|---|
| `0` | All tests passed |
| `1` | One or more tests failed, or invalid arguments |

## What it tests

### STUN (RFC 5389)

- Sends a **Binding Request** to the server
- Parses the **XOR-MAPPED-ADDRESS** from the response
- Reports your public IP and port as seen by the server
- Identifies NAT type (`open_internet`, `symmetric`, etc.)

### TURN (RFC 5766)

- Allocates a **relay address** on the TURN server using long-term credentials
- Reports the allocated relay IP, port, and lifetime
- Supports both UDP and TCP transports

## Public STUN servers

| Provider | Address |
|---|---|
| Google | `stun.l.google.com:19302` |
| Cloudflare | `stun.cloudflare.com:3478` |
| Freeswitch | `stun.freeswitch.org` |
| Twilio | `stun.twilio.com:3478` |

## License

MIT
