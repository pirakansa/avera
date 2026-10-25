# Local accounts, configured sudo and runtime groups

Status: implemented after owner agreement to review entry-related service users
and local server accounts. Management sockets, additional daemons and general
privilege-path enumeration remain outside this increment.

## Review boundary

The in-scope observations are current listening services, process effective UID/GID
and supplementary groups, local passwd/group account settings, and configured
sudo rules. External account directories, actual SSH authentication/login access,
management sockets, capabilities and other non-sudo privilege paths are outside
this review. A bundled English/Japanese paragraph states those exclusions in the
editor and report summary. It creates no findings, counters, required judgments
or follow-up tasks. An unavailable or malformed **in-scope source**, by contrast,
remains an actual collection diagnostic with partial/failed coverage.

No shell value, group name, UID, candidate service or configured sudo line is an
automatic safety/privilege judgment. Humans explain required duties and decide
whether observed settings are appropriate. Group membership alone is not a sudo
grant. The existing sudo grammar remains bounded; unsupported policy remains an
observed item requiring interpretation, not proof that no privilege exists.

## Live collection

The root-only CLI registers network, accounts and sudo collectors. Sudo reads
`/etc/sudoers` and supported includes through `sudo.LocalSource{Root: "/"}`; it does
not invoke sudo or an effective-policy query. Existing sudo grammar, symlink
rejection and diagnostics are unchanged. Evidence timestamps use the inventory
contract's second-precision UTC format, including when the clock has nanoseconds.

`internal/collect/accounts.Collector` reads only `/etc/passwd` and `/etc/group`
through an injected bounded source. The live source rejects symlinks and special
files. Password fields and GECOS are discarded; shadow, private keys,
authentication data, NSS and remote directories are not read. The account
collector has a 4 MiB total input budget, 16,384 non-comment data lines, 4,096
accounts/groups per source and 65,536 membership operations, including primary
membership expansion. Names are bounded to 256 bytes and account home/shell
fields to 4,096 bytes, avoiding expansion of oversized source strings across
memberships. Zero configurable limits use defaults; negative limits fail.

Each local account becomes `accounts` / `account.local`, identified by exact
account name within the logical target. Facts contain UID, configured primary
GID, home, shell, local group memberships and context completeness. Duplicate
UIDs remain separate named accounts; neither is selected as the definitive
identity. Group aliases with the same GID are preserved. Missing primary groups,
malformed/duplicate names, source failures and exhausted bounds retain usable
observations with incomplete coverage. Group member names absent from local
passwd remain observed directory entries; no external account is synthesized or
external-identity resolution task added. Groups with no explicit members remain
in the directory. The display separates primary membership from the explicit
member list, avoiding an empty-list inference about primary members.

Evidence records sanitized per-account facts with source line locations and a
structured group directory. It never retains raw passwd/group lines, password
fields or GECOS. Accounts use observed-snapshot judgments, not advance templates.
Unchanged complete account facts can reuse a prior judgment; changes in UID,
groups, home, shell or completeness require review. Sudo settings are reviewed
as separate findings; an account judgment is not a certification of its complete
privileges or of an associated sudo rule. Accounts are not added to shared endpoint
policies.

## Runtime groups and relationships

For a process already matched to an observed listening socket, the `Gid` effective
column and `Groups` list from bounded `/proc/<pid>/status` reads supply runtime
groups. An empty observed list is distinct from uncollected data. Numeric GIDs
remain visible; matching names from the local directory are shown when available.
These groups are separate from configured account memberships. Existing process
race checks clear runtime groups with the rest of unstable service metadata.
Malformed/missing group fields produce `service_groups_malformed`, preserving
known endpoints. PID remains technical evidence, not ordinary inventory display.

Service/account links use observed effective UID and matching local UID. All
matching aliases remain visible; the link is a local numeric match, not verified
login identity or proof of an external path. Account/sudo links use explicit
configured user names or locally observed membership in the configured subject
group. Relationships are derived read-only navigation, not new grants, matching
rules or judgments. HTML links stay within the corresponding current or historical
source; changed old relationships do not become current ones. Related navigation
is bounded to 64 links per item with a visible limit note when reached. The full
item inventory and judgment counts remain unchanged.

The editor and HTML/Markdown show readable account and sudo fields, including
run-as users/groups, commands, command-entry argument restrictions, password tags,
locally resolved subjects and interpretation completeness. Missing password tags
remain unspecified, never inferred as PASSWD/NOPASSWD. Human purpose fields invite
account function, necessity and responsible person/team; existing audience,
safeguards, rationale and actor fields remain available. Individual account
judgments persist through the portable report workflow.

## Contract compatibility

New live artifacts use `accounts-2026-10-25.1`; services, netfilter and T00 revisions
remain readable. Portable saves remain v3 with v1/v2 inputs supported. Existing
network/sudo identity and facts hash algorithms remain unchanged.

- Collector `accounts`, kind `account.local`, identity `{name}`.
- Account facts: `uid`, `primary_gid` (uint32), `home`, `shell` (strings), `groups`
  (array of `{name, gid, membership: primary|supplementary}`), `context_complete`
  (boolean). Group sets normalize by name, GID and membership; a complete account
  requires a resolved primary group matching its primary GID.
- Evidence `source_kind: local_identity` contains `{groups: [{name, gid, members}]}`.
  Members are explicit group-file members. Encode normalizes group names/member
  sets; primary memberships remain in account facts.
- Proc service evidence adds optional `effective_gid` (uint32) and
  `supplementary_groups` (uint32 array). Absence means uncollected, `[]` means
  observed empty. Duplicates are rejected; encoded arrays are sorted.
- Account payloads and runtime-group fields require the accounts revision. Older
  service evidence without these fields remains supported. No null migration,
  old snapshot rewrite or PID-based matching is introduced.

## Validation boundaries

Tests use fictional temporary passwd/group/sudo sources and synthetic service
observations. They cover data minimization, bounds/failures, empty and aliased
accounts, configured versus runtime groups, account save/repeat/change behavior,
known revision compatibility, derived relations, scope text and count invariants.
The browser regression `test/browser/accounts_sudo.py` exercises service → account
→ sudo navigation, account judgment saving, unchanged immutable observations,
standalone script-disabled HTML, escaping and narrow layouts. This does not
establish owner acceptance, actual SSH login access or a live sudo/systemd trial.
