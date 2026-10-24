# AVERA

**Asset Visibility, Evidence & Risk Annotation**

AVERA helps teams make the security of their products explainable through observed facts, evidence, and documented human judgment.

Its guiding principle is simple: **make security explainable, rather than promise security.**

AVERA is being designed for managers, developers, and operators who need to understand what deserves review in a running system, why it matters, and how the team has addressed or accepted it.

## Planned workflow

1. Collect configuration facts from a running Linux system, starting with listening network ports and sudo privileges.
2. Present items that need review, with plain-language explanations and questions.
3. Record each item's purpose, mitigations, evidence references, and human assessment separately from the collected facts.
4. Generate a report with a management summary, individual review items, and technical evidence.
5. Collect again after changes and compare results while preserving relevant annotations.

A listening port or an elevated privilege is a review item, not automatically a vulnerability. Reports will make collection failures, missing permissions, and inspection limits visible. Human statements will remain distinguishable from verified observations.

## Project status

AVERA is in the design stage. The `avera` CLI currently prints `Hello, world!`; security inventory features are not implemented yet.

Linux is the initial target. Windows and macOS support are future directions; their scope and schedule are undecided.

The first milestone is to define the data model and sample report before implementing collection. AVERA is intended to complement source code analysis and other security assessment activities. Its reports do not establish overall product safety or regulatory compliance.

## Development quick start

With Go 1.25.14 or later and [Vorbere](https://github.com/pirakansa/Vorbere) installed, run:

```sh
vorbere run setup
vorbere run build
./bin/avera
```

Expected output: `Hello, world!`.

Run `vorbere run ci` for formatting checks, static analysis, vulnerability scanning, tests, and a build. Analysis tools are pinned in `go.mod`; setup and vulnerability scanning require network access.
