# Network collector implementation handoff

T02 implements `internal/collect/network` against the T01 inventory contract.
The API and synthetic tests are implemented, and Linux v6.8 official-source
verification is complete. The enabled disposable Ubuntu comparison passed on
2026-10-24 with complete coverage. The CLI still has no collection command.

## API and sources

`Collector.Collect(ctx, targetID)` returns an `inventory.CollectorResult`.
Inject `Source` and `Now` for tests. A nil source selects `ProcSource`, which reads
the current process's `/proc/net/{tcp,tcp6,udp,udp6}` and enumerates `/proc/*/fd`.
Matching `socket:[inode]` links provide process attribution through `/proc/*/exe`.
No executable, username resolver, environment, command line, or network API is
invoked. Root enforcement is the future application boundary's responsibility;
the collector and all ordinary tests can run without root.

The backend accepts Linux/amd64 only. It reverses bytes within each printed
32-bit address word, including the four IPv6 words. IPv4 and IPv6 stay distinct.
Fixtures cover wildcard, loopback, and a non-loopback IPv6 address. These are
synthetic format tests, not architecture execution evidence on Ubuntu.

TCP includes state `0A` only. UDP includes state `07` only, with a nonzero local
port and an unspecified remote address with port zero. Connected UDP and
unbound port-zero rows are excluded. This UDP category does not imply TCP LISTEN
semantics, reachability, successful delivery, or firewall allowance. IPv6 zones
are unavailable from the chosen tables and remain empty; no dual-stack behavior
is inferred.

Each socket row supplies its observed socket UID, mapped into the proc reader's
user namespace, including when no process can be attributed. It is initially
associated with the creating user's filesystem UID but may change with socket
ownership changes. This is not a claim about an attributed process's current
effective UID or an immutable creator identity. A matched executable is an observed path, including any
kernel deletion suffix; neither integrity nor version is established. Multiple
processes may hold the same socket, and duplicate endpoints merge owners and
evidence using `inventory.MergeEndpoints`. PID, inode, row location, timestamp,
and enumeration order do not enter stable finding identities or facts hashes.
Evidence retains socket row locations and matched PID/descriptor locations.

## Scope, coverage, and bounds

Only the current observable network namespace is scanned. Proc namespace inode
numbers do not by themselves prove a stable incarnation across observations.
By default each run receives a cryptographically random `unverified-netns:` scope,
so comparisons must require re-review. `Collector.ScopeID` accepts a caller's
explicit assertion of comparable namespace incarnation; the fictional tests use
this injection. Integration must not supply a descriptive constant for live runs.

Default limits are 4 MiB per socket table, 16,384 data rows per table, 4,096 proc
directory entries, 65,536 descriptors across all processes, 4,096 bytes per link
target, and 65,536 evidence entries across the run. Zero fields select defaults;
negative fields are rejected. Proc entries other than PIDs count against the proc
enumeration bound conservatively. Limits do not silently establish completeness.
The total evidence bound also prevents duplicate rows with many owners from
expanding output without limit.

Missing or malformed tables, read/close errors, enumeration limits, inaccessible
or disappearing processes/descriptors, invalid or oversized executable targets,
and missing owners produce explicit diagnostics. Endpoints already observed are
retained. Any incomplete attribution enumeration conservatively limits all
observed endpoint attributions, including otherwise matched owners. All four
valid header-only tables produce complete, successful-empty coverage without
process enumeration. If no source supplies a usable header or endpoint, coverage
is failed. Source failures otherwise produce partial coverage. Cancellation,
invalid API configuration, or internal normalization failures return errors.
When enumerating its own descriptors, `ProcSource` excludes exactly the directory
descriptor it opened for enumeration. Closing that descriptor does not create an
artificial disappearance diagnostic. Other disappearing descriptors still limit
coverage, and the enumeration bound accounts for the omitted internal descriptor.

Cancellation is checked between source operations, rows, processes, descriptors,
and output observations. Individual filesystem syscalls are synchronous; an
injected source must return promptly. This is a non-atomic observation, and proc
races can remain undetectable. Completeness means completion of this bounded
scope, not proof that every host or container socket/owner was visible.

Diagnostic codes are `source_unreadable`, `source_malformed`,
`source_read_failed`, `source_limit`, `process_limit`, `process_unreadable`,
`descriptor_limit`, `descriptors_incomplete`, `descriptor_unreadable`,
`executable_unreadable`, `attribution_incomplete`, and `evidence_limit`.
Diagnostics do not include raw filesystem error strings or source dumps.

## Tests and T04 artifacts

Ordinary collector tests inject synthetic proc content; the `ProcSource` tests use
`t.TempDir()` exclusively. No ordinary test opens a socket, reads live host proc
tables, runs an external executable, or needs root. Tests cover TCP/UDP selection,
IPv4/IPv6 decoding, duplicate endpoints, multiple owners, disappearing or
inaccessible process details, malformed/missing sources, read and close failures,
empty success, bounds, cancellation, deterministic directory ordering, and fresh
scope IDs. A socket UID change preserves endpoint identity, changes its facts
fingerprint, and retains each duplicate socket's own UID. Shared `inventory.Encode`
validates the checked-in snapshots.
Temporary-directory tests also check that omission of the enumeration descriptor
preserves directory bounds and does not hide additional entries.

An opt-in `networklive` build tag adds `TestLiveUbuntuEndpoints`. It requires an
explicit acknowledgement and root PID 1 on Ubuntu with only a loopback interface
before creating any socket or changing loopback configuration. See the
[disposable validation procedure](../../user-guides/network-collector-validation.md).
The dedicated binary compiles with `CGO_ENABLED=0`; its unacknowledged run skips
the test. The enabled container comparison passed; see the recorded execution below.

Fixtures under `test/fixtures/prototype/network/` include the four source tables
and three full snapshots for the fictional target `fictional-web-01`:

- `snapshot-first.json`: the baseline observation.
- `snapshot-transient.json`: only PID/descriptor evidence changes; finding IDs
  and facts fingerprints match the baseline.
- `snapshot-changed-owner.json`: an executable path changes; finding IDs remain
  stable while affected facts fingerprints change.

To deliberately regenerate these fictional artifacts, run
`AVERA_UPDATE_NETWORK_FIXTURES=1 go test ./internal/collect/network`.
Ordinary tests compare generated output with the checked-in artifacts.

Validation performed: `vorbere run setup`, `vorbere run check`,
`vorbere run test` (race detector), and `vorbere run build` passed with Go
`1.25.14 linux/amd64`. The vulnerability scan reported no reachable vulnerabilities,
with one vulnerability in imported packages and thirteen in required modules;
this is not a clean dependency vulnerability report. An attempted offline setting
used `GOVULNDB`, which this govulncheck version does not support: the scan used
its normal database configuration, so it must not be described as offline.

## Official-source verification

On 2026-10-24, the following upstream Linux **v6.8** source functions were
inspected directly. These versioned references establish the implemented format
assumptions; they do not establish execution support for other kernels.

| Assumption | Official reference and verification |
|---|---|
| TCP local/remote endpoints, state, UID and inode fields | [TCP/IPv4 `get_tcp4_sock`](https://github.com/torvalds/linux/blob/v6.8/net/ipv4/tcp_ipv4.c#L2855-L2918) and [TCP/IPv6 `get_tcp6_sock`](https://github.com/torvalds/linux/blob/v6.8/net/ipv6/tcp_ipv6.c#L2167-L2238) emit addresses as hexadecimal native words, ports after `ntohs`, and state as two hexadecimal digits. UID and inode occupy whitespace fields 7 and 9 (zero-based). IPv6 emits four words. |
| TCP LISTEN and UDP state categories | [State enum](https://github.com/torvalds/linux/blob/v6.8/include/net/tcp_states.h#L12-L29) defines LISTEN as 10 and CLOSE as 7. [IPv4 datagram connect](https://github.com/torvalds/linux/blob/v6.8/net/ipv4/datagram.c#L25-L79) changes connected sockets to ESTABLISHED (1). The collector's UDP subset additionally requires an unspecified remote endpoint and a nonzero local port. |
| UDP row layout and address byte order | [IPv4 `udp4_format_sock`](https://github.com/torvalds/linux/blob/v6.8/net/ipv4/udp.c#L2890-L2923), [IPv6 `udp6_seq_show`](https://github.com/torvalds/linux/blob/v6.8/net/ipv6/udp.c#L1713-L1726), and [IPv6 common datagram formatter](https://github.com/torvalds/linux/blob/v6.8/net/ipv6/datagram.c#L1048-L1072) confirm the same field positions, native-word addresses, host-order hexadecimal ports, socket state and inode relationship. |
| Socket UID semantics | The formatters use `from_kuid_munged` in the reader's user namespace. [`sock_i_uid`](https://github.com/torvalds/linux/blob/v6.8/net/core/sock.c#L2591-L2600) reads the socket inode UID; [`sock_alloc` and `sockfs_setattr`](https://github.com/torvalds/linux/blob/v6.8/net/socket.c#L599-L646) explain initialization and ownership updates. This corrected the earlier overstatement that it is always the creator UID. |
| Inode-to-descriptor and executable attribution | [`sockfs_dname`](https://github.com/torvalds/linux/blob/v6.8/net/socket.c#L358-L364) produces socket inode links, and [`proc_fd_link`](https://github.com/torvalds/linux/blob/v6.8/fs/proc/fd.c#L171-L192) resolves a process descriptor. [Kernel proc documentation](https://docs.kernel.org/filesystems/proc.html) documents executable links, access restrictions, and PID races. Path reads remain non-atomic. |
| Namespace scope | [`proc_net_init`](https://github.com/torvalds/linux/blob/v6.8/fs/proc/proc_net.c#L408-L415) makes `/proc/net` refer to `self/net`. The scope is the calling process's observable network namespace. |

The [kernel TCP proc documentation](https://docs.kernel.org/networking/proc_net_tcp.html)
also confirms that the tables contain established connections as well as
listeners, and notes that the interface is deprecated in favor of `tcp_diag`.
The prototype deliberately retains its bounded proc backend; this verification
does not justify adding a second backend.

## Disposable Ubuntu execution verification

On 2026-10-24, `TestLiveUbuntuEndpoints` ran (without skipping) and passed
with exit status **0** and complete coverage. All eight IPv4/IPv6 wildcard and
loopback TCP/UDP endpoints matched, including executable `/network-live.test`
and socket UID 0 attribution. Established TCP connections, connected UDP, and
unbound UDP added no findings. No production collector code changes were needed.

- Userland: Ubuntu 24.04.5 LTS (Noble Numbat), official `ubuntu:24.04`.
- Image ID: `sha256:68ee22004a9109b44973313b02ff7b1de250f1de0f5ecb628d73f579aaff33ac`.
- Repository digest: `ubuntu@sha256:68ee22004a9109b44973313b02ff7b1de250f1de0f5ecb628d73f579aaff33ac`.
- Actual shared kernel: `7.0.0-38-generic`; architecture: Linux/amd64
  (Docker host reports x86_64); compiler: Go 1.25.14, `CGO_ENABLED=0`.
- Docker client/server: 29.9.2-2, local Unix socket, default context.
- Isolation: dedicated PID/network namespaces, `--network none`, no published
  ports, read-only filesystem and binary mount, all capabilities dropped except
  `NET_ADMIN`, no new privileges, 64 PIDs, 256 MiB memory and one CPU.

The [validation procedure](../../user-guides/network-collector-validation.md)
was executed as documented. Docker removed the disposable container on exit;
the downloaded image remains cached. The test configured only its own loopback
namespace and created temporary sockets there. No host service or firewall
configuration was changed.

Ubuntu userland runs against the Docker host's kernel. This result establishes
the tested combination only; it does not independently verify the provenance of
that kernel as Ubuntu-distributed, other kernels/architectures, reachability,
firewall behavior, or traversal of other container namespaces.

Post-comparison verification also passed: `vorbere run check`,
`vorbere run test` (race detector), `vorbere run build`,
`go vet -tags networklive ./internal/collect/network`, and
`go tool staticcheck -tags networklive ./internal/collect/network`.
`vorbere run clean` removed generated build artifacts. The vulnerability scan
again reported zero reachable vulnerabilities, one in imported packages and
thirteen in required modules.

### Full enabled comparison log

```text
=== RUN   TestLiveUbuntuEndpoints
    live_linux_test.go:58: OS: PRETTY_NAME="Ubuntu 24.04.5 LTS"
        NAME="Ubuntu"
        VERSION_ID="24.04"
        VERSION="24.04.5 LTS (Noble Numbat)"
        VERSION_CODENAME=noble
        ID=ubuntu
        ID_LIKE=debian
        HOME_URL="https://www.ubuntu.com/"
        SUPPORT_URL="https://help.ubuntu.com/"
        BUG_REPORT_URL="https://bugs.launchpad.net/ubuntu/"
        PRIVACY_POLICY_URL="https://www.ubuntu.com/legal/terms-and-policies/privacy-policy"
        UBUNTU_CODENAME=noble
        LOGO=ubuntu-logo; kernel: 7.0.0-38-generic; architecture: amd64; Go: go1.25.14
    live_linux_test.go:157: matched all 8 wildcard/loopback TCP/UDP endpoints and their owners; established TCP, connected UDP and unbound UDP did not add findings
--- PASS: TestLiveUbuntuEndpoints (0.01s)
PASS
```

## Live CLI forwarding extension

The socket-only collector described above is unchanged. The live CLI now wraps it
with read-only [forwarding collection](forwarding-collector.md), sharing namespace
scope and conservatively combining coverage. Socket-only fixtures retain the T00
contract revision.
