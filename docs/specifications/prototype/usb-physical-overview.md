# USB HID inventory and physical overview

New live artifacts use `usb-input-2026-10-25.1`. Earlier supported revisions, including
`usb-2026-10-25.1` and `storage-2026-10-25.1`, remain readable. The portable container remains v3.

## Observation and review boundary

The Linux USB collector reads bounded current device and active interface sysfs
metadata from `/sys/bus/usb/devices` and its resolved `/sys/devices` targets.
Device/interface class `03` identifies HID candidates. A physical USB device has
one `usb/usb.device` finding across its HID interfaces. Keyboard and mouse labels
use declared boot-HID subclass/protocol values; they do not authenticate a device.
Root hubs, hubs and non-HID devices are topology references, not extra judgment
items. Storage-only devices retain the external-storage judgment workflow.

The collector never opens input/device nodes, reads keystrokes, HID reports,
report descriptors, storage contents or binary USB descriptors. It never writes
USB authorization/configuration, disables devices or enforces permitted devices.
Empty ports, future insertions, physical desk location, Bluetooth and other
non-USB peripheral transports are outside this increment.

Human purpose, intended use, responsible team, permitted handling, operating
manuals and physical safeguards belong in the explanation. References are inert
human statements. The tool does not fetch manuals or verify implementation or
compliance. Acceptance requires the ordinary explicit actor, reason and decision;
missing explanations remain guidance, not automated adequacy judgments.

## Contract

Collector `usb` supplies coverage and `usb_metadata` evidence with typed
`USBValues.nodes`. Each node has `key`, `parent_key`, `port_path`, `sysfs_path`,
`instance`, `role` (`root_hub`, `hub`, `device`) and `facts`. Empty parent means a
root hub or unavailable attachment; the presentation distinguishes these cases.
Keys are unique; unknown nonempty parents, cycles and more than 1,024 nodes are
rejected. Physical connections come from Linux USB port paths, not device names.

`USBIdentity.key` uses VID/PID/serial when unique in the collection. Missing or
duplicate serial identifiers use a fresh collection-scoped key. Attachment paths,
bus/device numbers and active configuration are observation evidence, not stable
judgment identity. Identifier strings are not authenticated device identities.

`USBFacts` contains `vendor_id`, `product_id`, `manufacturer`, `product`, `serial`,
`device_class`, `interfaces`, `storage_fingerprints`, `observation_complete`, and
optional `input` as defined below.
An interface contains hexadecimal `number`, `class`, `subclass`, `protocol`.
Interfaces normalize by number; storage fingerprints normalize in sorted order.
USB findings require matching HID topology evidence. Incomplete USB coverage or
facts prevent acceptance reuse.

Optional `FileValues.usb` on storage evidence contains `sysfs_path`, `instance`,
`vendor_id`, `product_id`, `serial`. A storage-to-USB association requires all of
these current observations to agree. Product names alone never establish a link.
When an associated USB device has an HID finding, that USB item owns the judgment
for the whole composite device. Associated storage stays visible as reference,
with its old judgments retained, and is excluded from focused counts and separate
editing. Its fingerprint is included in the USB basis: storage changes require
fresh review, and incomplete storage coverage prevents reuse. Storage acceptance
never automatically becomes HID acceptance. Missing association evidence keeps
the original independent storage workflow.

## Limits and failure behavior

Reads are non-atomic. Device path, bus/device/configuration and VID/PID are
rechecked; changed attachments are dropped with partial coverage. Active
interface counts are checked. Malformed or unreadable metadata yields explicit
partial/failed coverage, never a successful absence statement. A successful empty
USB directory is distinct from uncollected or failed inputs.

Limits: 8,192 directory entries, 1,024 devices, 256 interfaces per device,
4 KiB per attribute, 256 bytes per displayed metadata label and 4 MiB total
attribute data. Cancellation stops collection. Symlink resolution stays inside a
trusted source root; tests use synthetic trees and never host hardware.

## Physical configuration overview

The editor and standalone HTML configuration page show the target, root hubs,
hubs and attached devices as a tree, with clickable HID and storage judgments.
Current human purpose, safeguards and derived decision state accompany HID
cards. The operating boundary above the host invites manual/physical safeguards,
as the network overview invites upstream controls. Neither is verified by AVERA.
Missing parents are explicit; old storage inputs without attachment evidence do
not receive invented edges. The diagram shows up to 64 USB nodes/eight levels;
remaining collected details stay in JSON. HTML is self-contained, escaped,
script-free and printable. The editor builds metadata as text nodes.

## Linux reference

The attribute formats and active-configuration semantics follow the Linux
[USB sysfs implementation](https://github.com/torvalds/linux/blob/v6.8/drivers/usb/core/sysfs.c)
and [USB sysfs ABI](https://github.com/torvalds/linux/blob/v6.8/Documentation/ABI/stable/sysfs-bus-usb).

## Input route extension

`usb-input-2026-10-25.1` adds optional `USBFacts.input` and `USBValues.input`.
Older USB snapshots omit these fields and explicitly display input routes as
uncollected. Existing USB findings and storage associations remain supported.
Live collection enables input metadata alongside the USB collector.

The association algorithm is:

1. Resolve `/sys/class/input/eventN` and `inputN` into the sysfs device tree.
   Match the longest USB HID device ancestor by path components. Read each event
   node's `dev` major/minor. Device names and numbers remain evidence only.
2. Read the sanitized `S: Sysfs` and `H: Handlers` projection from
   `/proc/bus/input/devices`. Match exact input-device sysfs paths. Normalize
   numbered `eventN` handlers to `evdev`; numbered mouse/joystick aliases do not
   enter the stable judgment basis. Kernel-only input devices can carry `kbd`
   even when no evdev node exists.
3. Enumerate visible `/proc/<PID>/fd` metadata. `stat` the existing descriptor
   reference without opening it. Match character devices by `st_rdev`
   major/minor, not by path spelling or a process-name guess. Regular files and
   sockets do not match. Multiple holders remain observed consumers.
4. For matching holders, project process name, executable-link text, effective
   UID, effective GID and supplementary groups from bounded proc metadata.
   Recheck process start time, executable, credentials and descriptors. Recheck
   event sysfs paths/numbers and USB attachment instance/VID/PID afterward.
   Unstable observations are discarded with incomplete diagnostics.

`USBInputFacts` contains `status` (`complete`, `partial`, `failed`), `handlers`,
`consumers`, `diagnostic_codes`. Each `USBInputConsumer` contains `process_name`,
`executable`, `effective_uid`, `effective_gid`, `supplementary_groups`. Consumers
and group memberships normalize as sets; duplicates and order do not change the
basis. PID, start time, FD numbers and input-node numbers are never persisted in
this stable basis. PIDs, start times, whole proc files, command lines and process
environments are not retained by input collection. Changed consumer identity,
executable, credentials or handlers changes the USB fingerprint and requires
fresh review. Partial/failed input scope prevents reuse of acceptance.

`USBInputValues` contains aggregate `status`, `diagnostic_codes`, and `devices`.
Each device contains `usb_key`, `sysfs_path`, `node`, `device_number`, `handlers`,
`consumers`. Kernel-only devices use empty `node`/`device_number` and have no
userspace consumers in this backend. Event nodes require matching event names
and valid character-device numbers. Evidence summaries must agree with USB node
facts and the USB finding. New input fields are rejected under old revisions.

Observed consumer effective UIDs link to all matching collected local accounts,
including UID zero and low-UID accounts. These accounts become individual review
items even without a network listener. Runtime groups stay distinct from account
configuration. A missing account does not cause external identity resolution.
The diagram branches kernel handlers and evdev interfaces from their Linux input
device; holder cards can link to a uniquely matched local account.

The relation means **descriptor held at observation time**, not delivered events,
active reads, a focused application, successful login or effective privilege
certification. `kbd` identifies a kernel virtual-terminal handler, never a
particular tty, shell or active console. Input events are never read. `hidraw`,
aggregate `/dev/input/mice`, brokers, virtual-device forwarding, inaccessible
process namespaces and downstream application routing are outside this backend.
No consumer found is distinct from incomplete enumeration and never proves that
USB input cannot operate. UID/GID values are in the reader's namespace.

Input limits: 4,096 input directory entries, 32,768 proc entries, 4,096 descriptors
per process, 262,144 metadata operations, 16 MiB total read data, 256 KiB per
process metadata file, 1 MiB handler metadata, 4 KiB executable/path text,
4,096 runtime groups, and at most 256 distinct consumers/64 handler names per USB
judgment. Limits, denied reads and races produce explicit incomplete status.
Source roots are trusted; proc descriptor targets are inspected only with
metadata `stat`, never opened or read. Tests use private sysfs/proc fixtures and
fake descriptor metadata, not host input devices.
