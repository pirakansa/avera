# Port-based listening service inventory

The live CLI enriches the existing TCP listener / bound-unconnected UDP inventory
with information about the processes holding those sockets. It does not enumerate
all daemons, installed packages, systemd units, users or privileges. Existing UDP
selection and loopback-reference rules remain unchanged.

## Sources and limits

`network.Collector.IncludeServices` enables enrichment. The live CLI enables it;
the original socket-only collector and T00 fixtures retain their original behavior.
After matching a descriptor to an observed socket inode, read that PID's
`/proc/<pid>/status`, `/proc/<pid>/cgroup` and `/proc/<pid>/stat`. No process without
a matched socket is inspected for service metadata. Status/cgroup are read once
per matched PID, even if it holds several observed sockets. No commands, systemd
API, NSS/username lookup, environments or command lines are used.

- `Name` supplies the observed process name, capped at 256 bytes. Names can be
  changed by the process; they do not attest a product or service identity.
- The second of the four `Uid` values supplies the process effective UID. It is
  displayed numerically, independently of the existing **socket UID**. Neither
  UID is a complete privilege inventory or stable human account identity.
- Unified cgroup entries (`0::`) and v1 `name=systemd` entries supply the cgroup
  path. The deepest `.service` component is a **systemd service candidate**.
  No matching unit, executable, intended socket-activation target or effective
  systemd configuration is verified. Non-systemd/container paths may have no
  service candidate; display an explicit unknown without inventing a name.
- Read stat start time before and after metadata collection, recheck the
  executable, and recheck each socket descriptor link before attaching metadata.
  If the process cannot be checked or changes, discard its service metadata and
  retain the endpoint with an explicit diagnostic. Collection remains non-atomic;
  these checks cannot eliminate every race or namespace visibility limitation.

Each metadata file read is limited to the smaller of the configured source-byte
limit and 64 KiB. Retained cgroup paths use the configured path-byte limit. Existing
process/descriptor/evidence bounds also apply. Raw stat/status/cgroup files are
not persisted. Unrelated fields read from status are discarded.

Missing, malformed, oversized or unstable sources preserve known endpoints and
make network coverage partial. Diagnostics are `service_status_unreadable`,
`service_status_malformed`, `service_cgroup_unreadable`,
`service_cgroup_malformed` and `service_process_changed`. A valid cgroup without
a `.service` candidate is an explicit unknown, not a collection failure.

## Storage and compatibility

New live artifacts use `services-2026-10-25.1`. Both `netfilter-2026-10-25.1` and
`t00-2026-10-24.1` remain readable. Portable reports remain v3, with v1/v2 inputs
still readable. The revision adds optional `service` to proc `ProcessValues`
evidence associated with a positive PID:

| Field | Type / meaning |
| --- | --- |
| `process_name` | String; empty when unknown |
| `effective_uid` | Optional uint32; absent when unknown, zero is valid |
| `unit_candidate` | String; empty or a `.service` component without `/` |
| `cgroup_path` | String; empty or an absolute observed path |
| `diagnostics` | Required array of unique nonempty diagnostic codes |

Service evidence is rejected under older revisions. Missing optional `service`
means metadata was not collected; explicit JSON nulls remain forbidden.
Evidence remains linked through endpoint `evidence_ids`. Service metadata does
not enter endpoint identities/fact fingerprints or shared judgment matching.
Existing executable/socket-UID fact semantics are unchanged. Process/service
changes therefore do not introduce new service-specific judgment invalidation;
coverage diagnostics remain independent of shared endpoint decisions.

## Presentation and human explanation

The editor and HTML/Markdown item details show process name, executable,
effective UID, socket UID, cgroup path, service candidate and diagnostics. Multiple
socket holders remain separate. PID is transient collection evidence, not a
service identifier, and is omitted from the ordinary service inventory display.
Existing technical evidence and JSON retain PID/source locations. Legacy reports show existing executable/socket
UID information with **unknown / not collected** for missing service metadata.
The configuration diagram adds abbreviated observed process/service information;
full details remain linked from the card. Loopback services remain reference-only.

For endpoint judgments, the purpose field invites the service's function,
necessity and responsible person/team. These remain human explanations, separate
from observed process data. No new mandatory annotation field or automatic
service judgment is added. Report summary counts and the three-page HTML layout
are unchanged.

## Verification

Synthetic collector tests cover targeted reads, multiple holders/sockets,
effective UID distinct from socket UID, PID replacement, missing/malformed/oversized
sources, v1/v2 cgroup forms, non-systemd paths, empty observations, bounded
retention and new/old contract rules. Report/viewer tests cover linked observations,
legacy unknowns, escaping, save/download preservation and HTML/Markdown output.
`test/browser/listening_services.py` checks the editor, human explanation save,
immutable observations, configuration diagram, offline script-disabled export
and narrow layouts. These checks do not establish live systemd or owner acceptance.
