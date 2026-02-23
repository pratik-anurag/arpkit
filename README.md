# arpkit

`arpkit` means **Architecture Profiling Kit**.

`arpkit` is a zero-daemon CLI for machine architecture profiling:

- topology (`NUMA -> socket -> core -> thread`)
- cache hierarchy and LLC sharing
- microarchitecture + ISA capability summary
- isolation posture (`isolcpus`, `nohz_full`, `rcu_nocbs`)
- power scaling snapshot (governor/driver/boost)
- memory and NUMA distribution
- PCIe NUMA affinity for NIC/NVMe devices
- static architecture posture score for performance engineering

## Why arpkit

- Fast and deterministic output for local performance engineering.
- Single binary, no background services.
- Linux-first topology depth using sysfs/procfs.
- macOS support with graceful degradation (no NUMA topology expected).
- No external command execution.
- Safe to run as non-root.

## Installation

Go 1.18+ is required.

### Go install

```bash
go install github.com/pratik-anurag/arpkit@latest
```

### Binary releases

Download archives from the GitHub Releases page:

<https://github.com/pratik-anurag/arpkit/releases>

Each release includes `tar.gz`, `zip`, and `checksums.txt` artifacts per supported OS/ARCH target.

### Verify checksum

```bash
sha256sum -c checksums.txt
```

## Build from source

```bash
go build -o bin/arpkit ./cmd/arpkit
```

Build with explicit version metadata:

```bash
go build -ldflags "-X main.version=v0.2.0 -X main.commit=$(git rev-parse --short HEAD) -X main.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o bin/arpkit ./cmd/arpkit
```

## Usage

```bash
./bin/arpkit [flags]
```

### Core flags

- `--format=pretty|json|dot`
- `--json` (alias for `--format=json`)
- `--color=auto|always|never`
- `--color-theme=auto|distro|mono`
- `--profile=min|default|verbose`
- `--only=summary,topology,cache,freq,microarch,power,posture,distance,llc,isolation,pcie,mem,memtop,notes`
- `--no-pill`
- `--compact`
- `--no-diagram`
- `--wide`
- `--unicode`
- `--mem`
- `--microarch`
- `--distance`
- `--pcie`
- `--posture`
- `--debug`
- `--version`
- `--help`

Version output:

```text
arpkit v0.2.0
commit: abc1234
built: 2026-02-23T02:30:00Z
```

Default profile includes a top pill header and these sections:
- Summary
- Topology
- Cache
- Power
- Architecture Posture

Verbose profile additionally includes:
- NUMA distance matrix
- Memory
- Isolation
- PCIe NUMA affinity
- full posture checks

## Output examples

Default pretty output (`--color=never` sample, Linux):

```text
[ Ubuntu 24.04 LTS | 6.8.0-52-generic | amd64 | Intel | Ice Lake ]

Summary
------------
CPU:       Intel(R) Xeon(R) Silver 4314 CPU @ 2.40GHz
Topology:  Sockets: 2  NUMA: 2  Cores: 32  Threads: 64  SMT: on
Freq:      cur=2394 MHz  min=800 MHz  max=3400 MHz
uArch:     Ice Lake
ISA:       avx,avx2,aes,bmi1,bmi2,fma,sha

Topology
------------
┌─ NUMA 0 (mem 126 GiB | cpus 0-31) ──────────────────────────────────────────┐
│┌─ S0 ───────────────────────────────────────────────────────────────────────┐│
││C0 [0-1]  C1 [2-3]  C2 [4-5]  C3 [6-7]  +12 more                           ││
│└────────────────────────────────────────────────────────────────────────────┘│
└──────────────────────────────────────────────────────────────────────────────┘

Cache
------------
L1i:       32 KiB per-core
L1d:       48 KiB per-core
L2:        1.2 MiB per-core
L3:        24 MiB shared by 32 threads (2 groups)
LLC Groups: Group 0: CPUs 0-31

Power
------------
Governor:  performance
Driver:    intel_pstate
Turbo/Boost: enabled
```

macOS example (`--color=never`):

```text
[ macOS 14.3 | 14.3 | arm64 | Apple | M2 ]

Summary
------------
CPU:       Apple M2
Topology:  Sockets: 1  NUMA: N/A  Cores: 8  Threads: 8  SMT: off
Freq:      cur=n/a  min=n/a  max=n/a
uArch:     Apple M2
ISA:       aes,sha1,sha2
```

Verbose profile sample command:

```bash
./bin/arpkit --profile=verbose --color=never
```

Sections you will additionally see in verbose:

- `Memory`
- `Isolation`
- `PCIe NUMA Affinity`
- `Architecture Posture` check list

JSON output:

```bash
./bin/arpkit --format=json
```

JSON schema (abridged):

```text
{
  "metadata": {...},
  "cpu": {...},
  "microarch": {...},
  "nodes": [...],
  "numa_distance": {"node_ids":[...], "matrix":[...]},
  "sockets": [...],
  "cores": [...],
  "threads": [...],
  "caches": [...],
  "llc_groups": [...],
  "memory_distribution": {...},
  "memory_topology": {...},
  "isolation": {...},
  "power": {...},
  "pcie_affinity": [...],
  "posture": {...},
  "flags": {...}
}
```

Graphviz DOT:

```bash
./bin/arpkit --format=dot > topology.dot
```

Feature-focused examples:

```bash
./bin/arpkit --microarch --posture
./bin/arpkit --distance --pcie --profile=verbose
./bin/arpkit --mem --profile=verbose
```

Width handling:

- `>=120` columns: side-by-side socket boxes when possible.
- `80-119` columns: stacked socket boxes.
- `<80` columns: compact collapsed core groups.
- `--wide`: disables topology truncation.
- `--color=auto` enables ANSI on TTY, otherwise monochrome.
- Pill header token truncation order is deterministic:
  1. Drop kernel token
  2. Shorten OS token
  3. Drop microarch token
  4. Fallback to `[ OS | Arch | Vendor ]`
- `--no-pill` disables the pill header.
- `--compact` removes blank lines between sections and tightens key/value spacing.

## Exit codes

- `0`: success
- `2`: unsupported platform or partial capability profile
- `1`: generic error

## Platform notes

### Linux

Primary implementation reads from:

- `/sys/devices/system/cpu`
- `/sys/devices/system/node`
- `/proc/cpuinfo`
- `/proc/cmdline`
- `/sys/bus/pci/devices`
- `/sys/devices/system/cpu/cpufreq`
- `/sys/devices/system/edac/mc` (memory topology when available)

### macOS

Uses `sysctl`-backed APIs through Go `syscall` bindings for:

- CPU brand/vendor
- Physical/logical core counts
- Cache sizes (when available)
- Memory size

NUMA is reported as unavailable (`N/A`) on macOS.

## Development

```bash
make fmt
make test
make build
make release-snapshot
```

## Release process

- Push a semantic version tag prefixed with `v` (for example `v0.1.0` or `v1.0.0`).
- Tag pushes matching `v*` trigger `.github/workflows/release.yml`.
- GitHub Releases are published by GoReleaser with per-OS/ARCH archives and `checksums.txt`.

Local validation:

```bash
goreleaser check
goreleaser release --snapshot --clean
```

## Project layout

```text
cmd/arpkit/main.go
internal/
  platform/
  topology/
  parser/
  render/
  util/
testdata/fixtures/
```
