# Prototype collection and evidence

Status: proposed bounded collector design. Implementation agents must verify source formats against the selected Ubuntu/kernel/sudo environment and official source documentation. Record the references and observed versions in their task handoff. This document does not claim complete Linux or sudo coverage.

## Common collector behavior

Collectors return typed findings, structured evidence, and coverage diagnostics through the shared model. They do not print user reports or make acceptance decisions. Inject source access and clocks so fixture tests run unprivileged and without inspecting the developer's machine.

Read from the running environment; do not modify services, firewall rules, users, files under `/etc`, or kernel settings. Provide cancellation and finite limits for source size, entries, directory traversal, and process enumeration. Hitting a limit must identify affected coverage. Avoid repeated privileged reads and do not turn one disappearing process into a whole-run failure.

Preserve sufficient source context to explain an observation, with safe errors that do not dump secrets. Collection itself need not retain complete source files. The output artifact is sensitive working material; publication/redaction is a separate future decision.

## Network example

### Initial implementation proposal

Start by evaluating direct reads of the relevant `/proc` socket tables for IPv4/IPv6 TCP and UDP. Use a fixture-driven parser and a short source-format verification spike before committing the implementation. A kernel socket-diagnostic interface is an alternative if the spike shows a material correctness problem; record that change before rewriting the shared model. Avoid building two interchangeable backends merely for anticipated future needs.

Restrict the initial scan to the process's observable network namespace. Report this scope explicitly. Do not claim all container namespaces or every host socket was examined. Stable namespace comparison must be conservative across restarts; if comparable scope cannot be established, request re-review rather than assuming equivalence.

### Interpretation

- TCP: include listening sockets; do not list established outbound connections as listeners.
- UDP: describe the selected bound/unconnected endpoint set accurately. UDP has no TCP-style LISTEN guarantee; document how connected UDP sockets are treated and state any exclusion.
- Keep IPv4 and IPv6 distinctions, local wildcard versus loopback bindings, and address zones where represented by the source.
- Do not infer internet exposure, firewall allowance, application authentication, protocol security, or dual-stack behavior solely from a local endpoint.
- Sort deterministically and retain evidence of multiple sockets/owners for one logical endpoint.

### Process attribution

Where available, relate socket information to `/proc/<pid>` descriptors and record executable path/name and UID. Verify the actual source relationship in the target environment. Do not collect process environments or full command lines. A process may disappear or ownership may be unreadable; record attribution as incomplete, keep the endpoint, and expose the limitation to reuse and reporting.

Executable paths and UIDs help users understand a service; they do not verify the executable's integrity, identity across replacement, version, or vulnerabilities. Local username resolution must not trigger remote directory lookups. If a safe local mapping is unavailable, show the numeric UID.

### Required fixtures

IPv4/IPv6; wildcard/loopback; TCP listener versus established connection; documented UDP categories; duplicate endpoint sockets; multiple owners; missing process; unreadable descriptors; malformed row; missing source; empty-success; deterministic ordering; PID churn without meaningful change.

Any live comparison belongs in an explicitly selected disposable test environment. Default tests use synthetic sources. Validate that source decoding does not depend on an untested architecture assumption; claim only the architectures actually tested.

## Sudo example

### Scope

The initial collector inventories local sudo configuration and its visible context. It is not a full effective-privilege evaluator. Do not claim to enumerate all administrative privileges, all sudo policy backends, or all command execution possibilities.

Discover and read the local policy entry point and supported include directives directly. Treat the conventional `/etc/sudoers` path as the initial Ubuntu source assumption to verify, not a universal guarantee. Do not run `sudo -l`, invoke arbitrary commands, or require `visudo` at runtime.

### Bounded parser strategy

T03 begins with a source inventory and supported-grammar table. Select a small subset of explicit local user/group rules, run-as targets, command lists, and supported tags that can be parsed without guessing. Preserve all relevant unsupported constructs as evidence-backed review items or source diagnostics, and mark semantic coverage partial.

Aliases, `Defaults`, negation, command argument matching, escaping, continuations, host matching, ordering, and include behavior can alter meaning. Implement a construct only with explicit semantics and fixtures. A simple regular expression over lines is not enough to claim effective grants. Unsupported constructs that could affect interpreted rules must limit the applicability of those interpretations, not just produce an unrelated warning.

For the first trial, it is acceptable to show “a configured rule requiring review” with incomplete interpretation. It is not acceptable to present an unsupported rule as a confirmed unrestricted grant, or to omit it and suggest there are no concerns.

### Includes and identity context

- Verify accepted include forms, path resolution, ordering, and filename selection against the supported sudo version. Avoid assuming every file in a directory is active policy.
- Bound recursive traversal and total bytes; detect cycles, unreadable files, and repeated sources. Define and test symlink treatment without changing target files.
- Record source locations and minimal excerpts with diagnostic context.
- Read local account/group information where required, including primary-group relationships. A group named `sudo` is not a complete policy description.
- Do not invoke remote NSS/LDAP/SSSD lookups. If policy or identity resolution can depend on external services, report that interpretation as outside observed local coverage.
- Preserve rule/order/context changes for fingerprinting. A changed included file or local group membership may invalidate an old assessment even if the visible main rule text is unchanged.

### Required fixtures

Explicit user rule; group rule with primary and supplementary local membership; run-as and command restrictions in the supported subset; supported tags; include and include-directory behavior; missing include; recursive cycle; unsupported alias/Defaults/negation; malformed or multiline input; local identity missing; changed group membership; source movement; limits exceeded.

Before implementing a broad parser, show the owner a report for one understood rule and one unsupported construct. Use feedback to decide whether deeper interpretation is valuable for the next increment.

## Evidence and coverage review

Both collectors must let the user answer: what was observed, where it came from, what scope was inspected, what could not be read or understood, and what needs human explanation. Coverage diagnostics must reach the summary, not remain buried in raw evidence.

Removing an item from a later observation is not proof that it was remediated. Scope changes, stopped processes, inaccessible sources, and intermittent operation require the comparison rules in [data contracts](data-contracts.md).

## Local accounts and service runtime groups

The implemented [accounts/privileges extension](accounts-privileges.md) scopes
account judgments to local passwd/group settings, separately from configured sudo
rules. It adds process effective GID/supplementary groups only for observed socket
holders. External identities, actual SSH login access and other privilege paths
remain a textual exclusion, not additional findings or required follow-up.
