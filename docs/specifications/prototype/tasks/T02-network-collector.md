# T02 — Collect the network example

Status: complete (2026-10-24). Collector API, synthetic tests, and T04 fixtures implemented; official Linux v6.8 source verification and the enabled disposable Ubuntu comparison passed. See the [implementation handoff](../network-collector.md) for scope, bounds, full execution log, and tested userland/kernel limitations. Prerequisites: T00 baseline; T01 interfaces before integration. Read [collection scope](../collection.md).

## Outcome

Produce evidence-backed network review items from the current observable Linux namespace without external commands, network probes, or host configuration changes.

## Ownership

`internal/collect/network`, its tests, and network source fixtures under `test/fixtures/prototype/network/`. Coordinate any shared model changes with T01 rather than adding parallel types.

## Work steps

1. Verify the proposed OS sources and field interpretation against official documentation/source and the selected environment. Record versions and references. Use a disposable environment for live exploration.
2. Write a supported-source and namespace-scope note; specify exactly which TCP and UDP socket categories are included.
3. Implement injectable source readers and parsers for the chosen IPv4/IPv6 sources, with deterministic ordering.
4. Create typed endpoint findings using the shared identity rules. Preserve wildcard/loopback/family distinctions and aggregate duplicate endpoint evidence deliberately.
5. Attribute sockets to processes where observable, retaining UID and bounded executable information. Treat races and inaccessible attribution as explicit limitations.
6. Implement bounded enumeration, cancellation, diagnostics, and successful-empty behavior.
7. Supply two snapshots of the same fictional target to T04: one with only transient changes and one with a meaningful changed owner/context.
8. On the chosen disposable Ubuntu target, compare expected listeners with observations and record source/architecture assumptions. Do not open a test port on a real product or modify its firewall as part of ordinary validation.

## Exclusions

No reachability probes, firewall inference, executable-version lookup, CVE checks, full-container namespace traversal, service restart, process environment dump, or remote username lookup.

## Acceptance cases

- TCP listeners appear; established connections do not masquerade as listeners.
- IPv4/IPv6 wildcard and loopback samples decode correctly on the supported architecture.
- UDP wording and inclusion match the documented category, with no TCP-style guarantee implied.
- Disappearing or inaccessible process details do not erase a valid endpoint.
- Missing/malformed sources produce partial/failed coverage rather than a clean empty list.
- Multiple socket owners are represented without unstable PID-based finding identities.
- The collector invokes no external executable and performs no external network access.
- All ordinary parser/collector tests run unprivileged against synthetic sources.

## Handoff

Provide actual covered scope, format references, output fixtures, live validation environment (if tested), and limitations. T04 consumes fingerprints, T05 consumes evidence/coverage, and T06 invokes the collector. Lack of a live test must remain visible in the handoff.
