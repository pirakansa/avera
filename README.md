# AVERA

**Asset Visibility, Evidence & Risk Annotation**

AVERA helps teams make the security of their products explainable through observed facts, evidence, and documented human judgment.

Its guiding principle is simple: **make security explainable, rather than promise security.**

See [Product direction](docs/specifications/product-direction.md) for the intended users, scope, execution constraints, and open design decisions.

The [prototype plan](docs/specifications/prototype/README.md) breaks the proposed implementation into agent tasks and user trial cycles. See the [documentation index](docs/README.md) for specifications and guides.

AVERA is being designed for managers, developers, and operators who need to understand what deserves review in a running system, why it matters, and how the team has addressed or accepted it.

## Planned workflow

1. Collect configuration facts from a running Linux system, starting with listening network ports and sudo privileges.
2. Present items that need review, with plain-language explanations and questions.
3. Record each item's purpose, mitigations, evidence references, and human assessment separately from the collected facts.
4. Generate a report with a management summary, individual review items, and technical evidence.
5. Collect again after changes and compare results while preserving relevant annotations.

A listening port or an elevated privilege is a review item, not automatically a vulnerability. Reports will make collection failures, missing permissions, and inspection limits visible. Human statements will remain distinguishable from verified observations.

## Project status

AVERA is an early Linux prototype with network collection, saved-file review, assessment reuse, and English/Japanese Markdown and HTML reports with embedded guidance. The CLI exposes `collect`, `review`, and `report`; P1 live collection enables network only. Sudo collection integration and owner usability feedback remain pending. See the [CLI reference](docs/specifications/prototype/cli-workflow.md) and [trial guide](docs/user-guides/prototype-trials.md).

## Development quick start

With Go 1.25.14 or later and [Vorbere](https://github.com/pirakansa/Vorbere) installed, run:

```sh
vorbere run setup
vorbere run build
./bin/avera report \
  --snapshot test/fixtures/prototype/contracts/snapshot-repeat.json \
  --previous test/fixtures/prototype/contracts/snapshot-first.json \
  --assessments test/fixtures/prototype/contracts/assessments-reviewed.json \
  --output /tmp/avera-fictional-report.md
```

Open `/tmp/avera-fictional-report.md` to inspect fictional findings, guidance, and evidence. Choose a new output path if it already exists. This rehearsal needs no root or live collection. Add `--format html --lang ja` and choose an `.html` output path for a Japanese browser report; see the [presentation trial](docs/user-guides/prototype-trials.md#japanese-html-presentation-trial).

Run `vorbere run ci` for formatting checks, static analysis, vulnerability scanning, tests, and a build. Analysis tools are pinned in `go.mod`; setup and vulnerability scanning require network access.
