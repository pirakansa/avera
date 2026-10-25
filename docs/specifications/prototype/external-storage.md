# Connected external storage inventory

Status: implemented from owner feedback. The review asks whether a connected
storage candidate has an understood purpose and should remain connected. A
purpose such as an application save destination plus a human reason and decision
can resolve it; mounting, unplugging or disabling is not a prerequisite.

## Collection and scope

The root-only live CLI registers `internal/collect/storage.Collector`. It reads
bounded metadata from `/sys/class/block`, resolved `/sys/devices` attributes and
`/proc/self/mountinfo`. It does not open `/dev` contents, invoke storage commands,
mount, unmount, scan files, evaluate file permissions or change device settings.
A local source with a private synthetic root supports ordinary tests without
root or real devices.

Candidates are whole block devices with USB ancestry in the resolved sysfs path,
or a removable flag. A USB connection/removable flag is a candidate criterion,
not proof of physical external placement. A non-root internal fixed disk is not
included merely because it is outside rootfs. USB devices without a block node,
other fixed external transports such as eSATA/PCIe/Thunderbolt without a removable
flag, and general USB peripherals are outside this backend. No extra judgment is
created for those exclusions.

Rootfs is located by the `/` mount's major:minor identifier. The collector follows
partition parents and block `slaves` through stacked mappings to their physical
backing disks. Every root backing disk and its other partitions are excluded,
including when root is on USB or a multi-device mapping. Other mounted mappings
are traced similarly, so a filesystem mounted through a crypt/LVM/RAID mapping
can appear on its physical storage partition rather than as falsely unmounted.
Only the current mount namespace is observed. The initial root-filesystem
backend supports ext2/ext3/ext4/xfs with a visible block graph. Filesystem-specific
multi-device roots such as btrfs, unsupported roots and unresolved virtual roots
are not guessed.

When root backing is unavailable, cyclic, ambiguous or an unresolved virtual
root (for example an overlay root), collection reports incomplete coverage and
suppresses candidate judgments. It does not guess that every non-root-looking
disk is external, or report successful absence. Zero candidates with complete
coverage is distinct from this failure and from an older file with no storage
collection record. Devices with no mount are still candidates when root backing
and their metadata can be resolved.

## Observations and identity

Facts contain connection classification, reported vendor/model/serial, byte
capacity, the disk read-only flag, and partition/mount observations. Mount paths,
filesystem type and per-mount read-only flags are retained, not raw mount sources
or superblock options that might contain credentials. Read/write flags do not
certify user access, content safety or absence of other access paths. Whole-device
unmounted rows are suppressed in ordinary display when partition rows exist.

A unique observed WWID is preferred for identity. Otherwise a USB vendor/product
and reported serial can supply an identifier. Missing or duplicate identifiers
among candidates use a fresh run-scoped identity, preventing automatic reuse by
`/dev/sdX` name alone. Identifiers are not authenticated and do not attest physical
device integrity. Transient device nodes, major:minor and disk sequence are kept
only in technical evidence. Disk sequence, device number and resolved sysfs path
are rechecked when available; reads remain non-atomic.

Live scope has a fresh nonce because namespace incarnation is not established.
The collector does not claim that an unmatched old identity disappeared across
equivalent hosts/namespaces. Stable identified devices with equal facts and
complete coverage can still reuse a human judgment. Changed capacity, mount
configuration or read-only state changes facts and requires review. No common
endpoint-policy behavior is extended to storage.

Default bounds: 4,096 block entries, 16,384 mount rows, 8 MiB total read bytes,
65,536 source operations, 4 KiB per small attribute/path and 4 MiB mountinfo.
Each candidate has at most 128 volume records and 256 mount observations. Reported
vendor/model/serial/WWID strings are bounded to 256 bytes. Missing optional identity
labels stay unknown; required unreadable/malformed data, races and bounds produce
partial/failed coverage with usable observations retained. Diagnostics use stable
`storage_*` codes without raw filesystem errors or source dumps. Cancellation is
checked between source operations; synchronous filesystem reads are not atomic.

## Judgment and presentation

Each `storage.external` finding is an ordinary main review item, mounted or not.
The browser form asks for purpose/necessity/responsible team, the reason for
continued connection, judgment and actor. It does not demand exposure, safeguards,
external evidence references or detailed file-permission analysis. Existing
optional human fields remain preserved if already present in JSON. Selecting
`accepted` displays **追加対応不要**, a human decision rather than a security test.
Purpose/reason completeness prompts do not introduce additional technical tasks.

The editor, HTML and both Markdown languages show product/capacity, serial,
connection classification, mount state and read-only observations. Unmounted
storage is explicitly identified as connected. Older files show uncollected
storage rather than an empty inventory. HTML preserves its three-page/offline
layout. Dynamic product names, labels and Markdown headings are escaped as text.

## Contract

New live artifacts use `storage-2026-10-25.1`. Accounts, services, netfilter and T00
contracts remain readable. Portable saves remain v3 and read v1/v2 inputs.

- Collector: `storage`; kind: `storage.external`.
- Identity: `{key: string}`; exact nonempty observed or run-scoped identifier.
- Facts: `connection` (`usb` or `removable`), `vendor`, `model`, `serial` (strings),
  `size_bytes` (nonnegative exact integer), `read_only` (boolean),
  `observation_complete` (boolean), `volumes` (array).
- Volume: `{partition: nonnegative integer, mounts: [...]}`. Zero denotes the
  whole device; other values are observed partition numbers. Duplicate partition
  numbers are invalid. Normalize volumes by partition number.
- Mount: `{path: absolute string, filesystem: nonempty string, read_only: boolean}`.
  Exact duplicates are rejected by the artifact contract; collection merges
  duplicate observations. Normalize by path, filesystem and read-only state.
- Evidence: existing `local_file`/`FileValues` with sanitized structured metadata,
  never raw device contents or complete mountinfo. Old contracts cannot contain
  storage kinds/coverage. Storage advance templates are not implemented.

Storage uses existing target-scoped identity/fact hashing, annotation provenance,
save/download/export and applicability rules. A complete observation and coverage
permit a saved matching judgment to apply without requiring a mounted filesystem.
No automatic accept/reject, rootfs modification or storage control API is added.

## Source verification and tests

The format assumptions were checked against official Linux references:

- [Linux proc mountinfo documentation](https://docs.kernel.org/filesystems/proc.html#proc-pid-mountinfo-information-about-mounts): major:minor, mount point,
  separator/optional fields and per-mount options.
- [Linux v6.8 block/genhd.c](https://github.com/torvalds/linux/blob/v6.8/block/genhd.c): block size/ro attributes, disk sequence and slave directory.
- [Linux v6.8 partition attributes](https://github.com/torvalds/linux/blob/v6.8/block/partitions/core.c): partition number and block-device linkage.

Temporary sysfs/proc tests cover mounted/unmounted USB, internal disk exclusion,
root mappings including USB/multi-device roots, encrypted external mappings,
missing/ambiguous root, malformed/bounded sources, fresh fallback identities,
transient evidence, complete empty results, cancellation, no `/dev` reads and
secret-option exclusion. Report tests cover purpose/reason-only acceptance,
roundtrip/reuse, changed-state review, legacy status and hostile product names.
The optional `test/browser/external_storage.py` covers the concise form, saving,
immutable observations, offline escaped HTML and narrow layouts. These tests do
not establish live USB hardware coverage or owner acceptance.
