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

AVERA is an early Linux prototype with network collection, saved-file review, assessment reuse, and English/Japanese Markdown and HTML reports with embedded guidance. The CLI collects one portable report JSON and provides a local browser editor with JSON/HTML export through `serve`, including reusable service judgments across equivalent servers. Live network collection includes socket observations and read-only iptables/nftables forwarding candidates. Owner feedback on the browser/report workflow is in progress; final acceptance and sudo collection integration remain pending. See the [CLI reference](docs/specifications/prototype/cli-workflow.md) and [trial guide](docs/user-guides/prototype-trials.md).

## Development quick start

With Go 1.26.10 or later and [Vorbere](https://github.com/pirakansa/Vorbere) installed, run:

```sh
vorbere run setup
vorbere run build
trial_dir=$(mktemp -d /tmp/avera-review.XXXXXX)
./bin/avera serve \
  --input test/fixtures/prototype/reports/fictional-report.json \
  --output-dir "$trial_dir"
```

Open the printed local URL to edit judgments in Japanese, save a new report JSON, and export HTML. This fictional rehearsal needs no root. The [single-report workflow](docs/user-guides/report-workflow.md) covers live collection, carrying one file between target and host, and repeat use. With an older development Go installation, `GOTOOLCHAIN=auto` enables downloading the required toolchain.

Run `vorbere run ci` for formatting checks, static analysis, vulnerability scanning, tests, and a build. Analysis tools are pinned in `go.mod`; setup and vulnerability scanning require network access.
