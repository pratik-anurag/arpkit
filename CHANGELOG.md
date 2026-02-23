# Changelog

## [Unreleased]

### Added

- GoReleaser configuration in `.goreleaser.yaml` for Linux/macOS builds (`amd64`, `arm64`) with `tar.gz` and `zip` archives plus `checksums.txt`.
- GitHub Actions release workflow in `.github/workflows/release.yml` triggered by `v*` tags.
- Root `main.go` entrypoint so `go install github.com/pratik-anurag/arpkit@latest` resolves to an installable command package.
- `make release-snapshot` target.

### Changed

- `--version` now prints version, commit, and build date.
- Build logic moved to `internal/cli` and reused by both root and `cmd/arpkit` entrypoints.
- README installation and release documentation updated for GoReleaser workflow and checksum verification.

## [0.2.0] - 2026-02-22

### Added

- Microarchitecture awareness:
  - microarchitecture naming from CPU vendor/family/model/model-name
  - ISA feature summary for x86 (`avx`, `avx2`, `avx512f`, `aes`, `bmi1`, `bmi2`, `fma`, `sha`)
  - ISA feature summary for arm64 (`sve`, `sve2`, `aes`, `sha1`, `sha2`, `crc32`)
- NUMA distance matrix parsing from `/sys/devices/system/node/node*/distance` and pretty/JSON rendering.
- SMT-oriented static hints with deterministic cpuset generation:
  - one thread per core
  - full SMT
  - spread across sockets
- LLC sharing map (`llc_groups`) with deterministic grouping and normalized cpuset output.
- Isolation object and summary (`isolcpus`, `nohz_full`, `rcu_nocbs`) with normalized CPU lists.
- Linux power snapshot (`governor`, `driver`, `turbo/boost`) from cpufreq and intel_pstate paths.
- Best-effort memory topology (`DIMMs populated/total`, `channels`) from EDAC sysfs when available.
- Lightweight PCIe NUMA affinity for NIC/NVMe devices from `/sys/bus/pci/devices`.
- Architecture posture score (`0.0`-`10.0`) with static check breakdown.
- New CLI flags:
  - `--microarch`
  - `--distance`
  - `--pcie`
  - `--posture`
  - `--color-theme`
  - `--unicode`
- New pretty renderer chip pill header:
  - `[ <OS> | <Kernel> | <Arch> | <Vendor> | <Microarch> ]`
  - deterministic pill token truncation on narrow terminals
  - color accents for OS/vendor/microarch with ANSI-safe resets
- New pretty output flags:
  - `--no-pill`
  - `--compact`

### Changed

- Pretty renderer default profile now includes microarch, ISA, power snapshot, and posture summary.
- Verbose profile includes distance matrix, LLC groups, isolation, PCIe affinity, and memory topology sections.
- Replaced ASCII logo/badge area with modern sectioned layout and top pill header.
- Pretty sections are now title/underline based with minimal borders (topology map remains boxed).
- Topology model extended for JSON output with:
  - `microarch`
  - `numa_distance`
  - `llc_groups`
  - `isolation`
  - `power`
  - `memory_topology`
  - `pcie_affinity`
  - `posture`
- Deterministic ordering and non-nil normalization expanded to new model fields.

### Tests

- Added unit tests for:
  - microarch parsing and ISA extraction
  - NUMA distance parsing
  - cmdline isolation parsing
  - LLC grouping logic
  - posture scoring determinism
  - PCIe affinity mapping from fixture sysfs tree
- Added JSON golden test for deterministic output ordering.
- Updated pretty-output goldens for widths `60`, `80`, and `140`.
- Added pill header layout goldens for:
  - Ubuntu amd64 Intel Skylake (widths 60/80/140)
  - Arch amd64 AMD Zen 3 (widths 60/80/140)
  - generic Linux arm64 ARM unknown (widths 60/80/140)
  - macOS arm64 Apple M2 (widths 60/80/140)
- Added pill truncation rule tests and ANSI visible-width tests.

## [0.1.0] - 2026-02-22

### Added

- Initial production-grade `arpkit` CLI implementation.
- Linux topology discovery via sysfs/procfs for sockets, cores, threads, NUMA, offline CPUs, and kernel-isolated CPUs.
- macOS collector with graceful NUMA degradation and sysctl-derived CPU/cache/memory data.
- Cache hierarchy extraction and summarization (L1i/L1d/L2/L3).
- ASCII topology diagram grouped by `NUMA -> Socket -> Core -> Threads` with truncation and `--wide` support.
- Memory and NUMA distribution reporting via `--mem` and verbose profile.
- Affinity and engineering hints via `--hints`.
- Output modes: `pretty`, `json`, and Graphviz `dot`.
- Deterministic sorting and stable output across renders.
- Tests:
  - CPU list parser unit tests
  - Topology normalization tests
  - Pretty output golden test
  - JSON determinism test
- Fixture scenarios:
  - single socket
  - dual socket NUMA
  - SMT off
  - offline CPUs
  - cache missing
- Makefile for build/test/fmt targets.
- GitHub Actions CI for Linux and macOS.
