# Bounded local sudo collector

T03 implements configured-rule observations through `inventory.CollectorResult`, not effective permissions. T04/T05/T06 can consume its shared findings, evidence and coverage without a separate decision store. The CLI/report are not implemented by this task.

## Reference baseline and supported grammar

The selected reference is Ubuntu 24.04.5 LTS, sudo `1.9.15p5-3ubuntu5.24.04.4` (Linux/amd64). On 2026-10-24, the official Ubuntu [sudoers(5)](https://manpages.ubuntu.com/manpages/noble/man5/sudoers.5.html) was checked for include forms, relative paths, directory ordering/filtering, command arguments, tag inheritance and reserved built-ins. The initial local Debian sudo 1.9.13p3 manual was also checked. The subset below was validated with the selected Ubuntu syntax checker; it does not implement the full language.

Local identity formats were checked against Ubuntu [passwd(5)](https://manpages.ubuntu.com/manpages/noble/man5/passwd.5.html) and [group(5)](https://manpages.ubuntu.com/manpages/noble/man5/group.5.html): numeric account GIDs provide primary membership; group member lists provide supplementary membership. Authentication fields are not retained or evaluated.

| Construct | Supported interpretation |
| --- | --- |
| Explicit lowercase local user or `%group` | One principal per rule; local account/group files only |
| Host | Literal `ALL` only; no hostname matching |
| Run-as | Required `(user[,user][:group[,group]])`; lowercase local names or `ALL` |
| Commands | Comma-separated `ALL` or simple absolute file paths; preserve order |
| Arguments | No argument expressions; bare paths have no argument restriction, represented by `arguments: ""` |
| Tags | `PASSWD:` / `NOPASSWD:`; inherited within the command list, reset for each rule; absent tag remains absent |
| Includes | `@include`, `#include`, `@includedir`, `#includedir` with one unquoted path; relative to including file |
| Include directory | Lexical order; skip names containing `.` or ending in `~`; return to parent after traversal |
| Comments | Blank lines and standalone `#` comments; numeric `#uid` and include forms are not comments |
| Unsupported | Aliases, Defaults, negation, host matching, argument expressions, escapes, continuations, inline comments, quoted paths, `%h`, other tags and directives; `sudoedit` (including absolute-path spelling) |

Any unsupported policy line becomes `sudo.unparsed`. It also limits **all** configured-rule interpretations in the run. Missing sources, cycles, repeated sources, identities and limits similarly prevent complete interpretation. A continued physical line and every following continuation segment remain unsupported; no suffix is interpreted as an independent rule.

## Source and resource boundaries

The collector accepts an injected `Source` and clock. `LocalSource{Root: ...}` pins directory descriptors using Linux `openat` and `O_NOFOLLOW`, rejecting symlinks in every source path component even during replacement races, special files and noncanonical source paths. The caller-selected root itself is trusted as the configured boundary. An explicit live caller may select `/`; tests only use temporary fictional roots. It does not inspect NSS, LDAP, sudo.conf, file ownership/mode eligibility or policy backend selection. A configured file is not proof that the installed policy plugin uses it. Collection is not atomic; file replacement may affect observations, and no provenance or permission validation is claimed.

The entry point defaults to `/etc/sudoers`. Missing entry point is failed coverage; other incomplete sources are partial coverage. Diagnostics retain attempted paths and evidence locations but omit operating-system error text. Local passwd/group password fields are neither retained nor shown. The shared `policy_context` retains relevant original policy lines in traversal order, plus path/content digests and diagnostics. Local identity files contribute digests only. Included policy, ordering and membership changes affect all supported-rule fingerprints conservatively. Explicit user findings include the resolved user with primary membership context; group findings include both primary and supplementary membership. Source path/physical line/ordinal anchors use T01 normalization; source movement produces a new finding.

Default whole-run limits are 4 MiB of source content, 256 source operations, 4,096 directory entries, 16,384 physical policy/identity lines, 32 nested include levels, and 4,096 bytes per source path. The total serialized policy context budget is 16 MiB across rules. If full context would exceed that budget, every rule requests re-review (`context_limit`, incomplete interpretation) rather than emitting a misleading truncated interpretation. All fields of `Limits` are injectable for tests; negative values fail.

## Validation and fixtures

All default tests use temporary fictional policy roots and perform no root checks or machine-policy reads. Fixtures include accounts, groups, explicit and unsupported policy, plus [understood](../../../test/fixtures/prototype/sudo/snapshot-understood.json) and [unsupported](../../../test/fixtures/prototype/sudo/snapshot-unsupported.json) common-contract snapshots. Tests verify the checked-in snapshots; explicit regeneration uses `AVERA_UPDATE_SUDO_FIXTURES=1 go test ./internal/collect/sudo`.

`AVERA_SUDO_REFERENCE=1 go test -race -v ./internal/collect/sudo -run TestDisposableVisudoReference` passed with the installed Debian sudo 1.9.13p3 syntax checker. It checked only temporary fictional files, relative includes and both supported rules and valid-but-unsupported Defaults. The default suite skips this optional comparison. No effective-policy query was executed. This is not an Ubuntu execution claim.

After network approval, `vorbere run check`, `vorbere run test` (race detector), and `vorbere run build` all passed with Go 1.25.14 linux/amd64. The scan found zero reachable vulnerabilities, one vulnerability in imported packages and thirteen in required modules; this is not a claim that all dependencies are vulnerability-free. Build artifacts were removed after validation.

The first online scan flagged [GO-2026-6604](https://pkg.go.dev/vuln/GO-2026-6604), a Windows-specific `os.Root` issue, through this Linux-only collector's `os.Root` calls. The collector now uses the existing `golang.org/x/sys/unix` dependency to pin each directory with `openat`/`O_NOFOLLOW`. This also removes the earlier gap between checking a symlink and opening its replacement. The final scan succeeded without changing dependency versions or suppressing checks. Official sudo documentation also identified the special handling of `sudoedit`; the collector now preserves that construct as unsupported instead of interpreting an absolute path as an ordinary command.

### Disposable Ubuntu comparison

The optional `TestDisposableVisudoReference` ran without skipping and passed with exit status 0 on Ubuntu 24.04.5 LTS, sudo `1.9.15p5-3ubuntu5.24.04.4`. The base image was `ubuntu:24.04`, digest `sha256:68ee22004a9109b44973313b02ff7b1de250f1de0f5ecb628d73f579aaff33ac`. A disposable derived image installed sudo from the official Noble repositories. No package was installed on the host.

The test binary was compiled with `CGO_ENABLED=0 go test -c` and executed with a read-only repository/binary mount, read-only root filesystem, writable temporary filesystem, no network, all capabilities dropped, no new privileges, 64 PIDs, 256 MiB memory and one CPU. The reference checker and collector both read only temporary fictional policy roots. The container was removed automatically after the comparison. The optional host comparison also passed on Debian sudo 1.9.13p3; neither comparison evaluates real effective permissions.

To repeat the Ubuntu comparison, build a disposable image from the recorded Ubuntu base with `sudo` installed, compile the test binary, and run from this package's directory with the repository fixtures mounted read-only:

```sh
CGO_ENABLED=0 go test -c -o /tmp/avera-sudo-reference.test ./internal/collect/sudo
# Inside the disposable Ubuntu container, with the test binary mounted:
AVERA_SUDO_REFERENCE=1 /sudo-reference.test -test.v -test.run TestDisposableVisudoReference
```

Observed comparison output:

```text
sudo  1.9.15p5-3ubuntu5.24.04.4
=== RUN   TestDisposableVisudoReference
--- PASS: TestDisposableVisudoReference (0.01s)
PASS
```

## T05 handoff and next trial

For `alice ALL=(root) NOPASSWD: /usr/bin/id`, label the item **Configured sudo rule**, show its file/line, explicit run-as user, inherited tag and **arguments unrestricted**. State that effective permissions are not fully resolved.

For `Defaults authenticate`, show **Unsupported sudo configuration requiring review** with its source excerpt and `unsupported_construct` diagnostic. Any neighboring configured rule must show incomplete interpretation. Coverage and limitations belong in the report summary. Membership in a group named `sudo` alone produces no grant finding.

The disposable Ubuntu policy trial covered explicit user rules, a local-group rule and a Defaults line. The next useful trial is to feed the provided understood/unsupported snapshots into T05, verify that reviewers understand the configured/effective distinction, then expand the grammar only if that trial requires it. An installed syntax checker is optional for disposable fixtures and is never a runtime dependency.
