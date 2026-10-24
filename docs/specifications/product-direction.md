# Product direction

Status: agreed product direction from the requirements discussion; detailed behavior remains a draft. This document describes intended behavior, not implemented capabilities.

## Purpose and users

AVERA (Asset Visibility, Evidence & Risk Annotation) helps people inventory security concerns in products, servers, and PCs and explain how each concern is addressed or accepted.

Managers and developers new to security may receive a request to improve security from executives, customers, or compliance stakeholders without knowing what to inspect or how to report the result. They may turn to source code analysis because it produces visible findings. That work can improve application security, but it leaves other aspects of the deployed system unexplained.

The missing capability is a practical framework for identifying concerns and recording their treatment. Users need help understanding what constitutes a useful explanation, not just a text field asking them to justify a finding. The audience includes managers and beginners as well as experienced developers and operators; execution and review roles are not fixed.

## Core responsibilities

- Enumerate observable conditions that deserve security review, including communication entry points and privileges.
- Explain why an item matters and help users identify relevant purposes, exposure, safeguards, evidence, and remaining risks.
- Keep observations, human statements, evidence references, and human decisions distinguishable. A statement with an attached reference is not automatically verified.
- Keep unexplained items visible. Retain explained or accepted items and their rationale in the inventory instead of silently removing them.
- Make collection scope, failures, and permission limitations visible. Missing observations do not establish the absence of a concern.
- Produce a report that lets recipients trace what was inspected, what was found, what was done, and why each decision was made.

AVERA supports an explanation of security decisions. It does not automatically establish overall safety or regulatory compliance. A required product feature may still need safeguards; its necessity alone does not settle its risk.

This inventory complements source code analysis, penetration testing, and product threat analysis. Scores and dashboards are not prerequisites for the prototype.

## Example: an HTTP service

An observed listener on port 80 becomes a review item. The user needs help describing its context, rather than merely entering “required by the product.” Possible explanatory dimensions include:

| Dimension | Example information |
|---|---|
| Purpose and functionality | Public static content; no login or data submission functionality |
| Intended audience | Public internet, internal users, or selected devices |
| Access restrictions | An upstream firewall restricts access to specified networks |
| Safeguards | Authentication where applicable, transport protection, restricted privileges, or multiple layers of protection |
| Supporting evidence | Service design, network diagrams, or configuration references |
| Decision | Remediation needed or acceptance with recorded rationale and remaining risk |

These are explanatory dimensions, not a mandatory checklist or automatic acceptance rules. Static content describes the service's functionality; it does not by itself prove safety. Authentication may be irrelevant to an intentionally public service.

Relevant protections may exist outside the inspected host. AVERA should allow those protections to be documented while making clear whether they were observed, reported by a person, or independently verified. A local listener alone does not establish external reachability.

The mechanism for guiding users toward sufficient explanations remains to be designed: questions, contextual prompts, structured answers, and review rules are options, not agreed features.

## Execution constraints

- Ubuntu is the initial validation target. The release and supported host, container, or embedded scenarios remain undecided.
- Aim for a single distributable binary with as few target-side dependencies as possible. Do not require installing an SSH server, opening a port, or adding a resident service to inspect the environment.
- Prefer local, read-only collection using information already available from the operating system. Avoid mandatory external utilities and runtimes where practical.
- Collection requires root privileges to inspect the target machine. An unprivileged collection mode is not required for the prototype. Root privileges do not change the read-only inspection approach or guarantee that every information source is accessible; collection limits and failures must still be reported.
- The design should support collection without internet access or external APIs. The exact output transfer and report-generation workflow remains open.
- Development tools are separate from dependencies required to execute the distributed binary. One binary per supported OS/architecture is compatible with this direction.

## Scope boundaries

Listening TCP/UDP ports and sudo-related privileges were chosen as easy-to-understand examples for an exploratory prototype. They do not define the intended inventory coverage or the scope of a first release. Their exact coverage and interpretation are not yet specified.

Collection considerations retained for prototype design:

- Ports: consider IPv4/IPv6, local addresses and ports, and associated processes and execution users where available.
- Sudo: consider sudoers, included configuration, and group-derived permissions. Membership in a group named `sudo` alone does not establish all effective privileges. Explain which rules and sources were actually inspected.
- Future candidates include services and systemd units, users and authentication paths, container privileges and mounts, file permissions, and network or firewall configuration. These are possibilities, not a committed roadmap.

## Prototype approach

The next implementation milestone is an unreleased prototype using those two examples. The project owner will act as the initial test user, try the workflow, and use that experience to explore usability and needed features. Implementing the two examples is not a release milestone or evidence that the inventory is complete.

Use a small collection-to-explanation-to-report workflow to make design choices tangible. Guidance, input methods, report presentation, and further collection items can evolve through trial and feedback. Define release scope and readiness separately after learning from the prototype.

## Exclusions

SBOM generation or integration and software-version-based CVE matching are out of scope. For example, AVERA is not intended to identify an outdated Nginx version and match it to vulnerabilities. Distribution packaging and backported fixes introduce a separate maintenance burden, and dependency vulnerability assessment is treated as a separate workflow.

This product scope does not prohibit vulnerability scanning of AVERA's own dependencies during development.

## Reusing prior assessments

AVERA must support supplying previous inventory information and assessments made in advance when running collection on the target machine. The resulting inventory should use this context to help the user identify and assess items that still need a decision. Users should not have to rebuild their explanations from scratch on every run.

Prior observations and supplied assessments must remain distinguishable from current observations. Reusing an assessment does not mean hiding its item or treating a human statement as newly verified evidence. How to match items, decide whether a previous assessment still applies, and request reassessment after changes remains open.

Collecting on the target machine and reviewing on a separate workstation is an acceptable working assumption, not a fixed workflow. Input formats, editing methods, and whether the reviewed output itself becomes the report or is combined with other data to generate one will be explored through prototype use.

## Report direction and open decisions

The intended report has three levels: a summary, individual items with explanations and decisions, and technical evidence. It should support communication to requesting organizations, executives, customers, auditors, and relevant authorities. Markdown and HTML are candidate formats; presentation can evolve.

These levels serve different needs. Technical evidence gives inexperienced users material on which to base their inspection and judgment, not just an audit trail after the decision. Individual items connect that evidence to explanations and decisions. The summary makes the results and remaining issues understandable to recipients unfamiliar with operational details. These roles are agreed; their concrete presentation will be explored through the prototype.

Traceability should cover the inspected target and OS, collection time, tool version, information sources and methods, and who made a judgment and when. Summary candidates include assessment counts, unresolved items, and changes since the previous inventory. Exact fields and handling of sensitive evidence remain design work.

The following remain open:

- The input and editing mechanisms for prior inventories, advance assessments, and decisions made after collection.
- How guidance helps inexperienced users produce adequate explanations, and who judges adequacy.
- Assessment states, evidence verification, reviewers, and re-review rules.
- Exact collection limits for each information source and permissions needed for later annotation or report-only operations.
- Storage formats, stable item identifiers, assessment carry-forward rules, and comparison between collections.
- Whether reviewed collection output is itself the report or an input to a separate report-generation step.
- Handling sensitive output and supporting explanations about protections outside the host.
- Which capabilities belong in the first usable release.

For each prototype increment, make the behavior under trial concrete enough to implement and verify; the complete product specification need not be settled first. This document supersedes the original handoff memo and records the clarified direction. Only explicitly agreed decisions are requirements; candidate features are not commitments. If this document is later removed, preserve its agreed decisions in the specifications that replace it.
