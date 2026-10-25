package report

import (
	"strings"
	"testing"

	"github.com/pirakansa/avera/internal/assessment"
	i "github.com/pirakansa/avera/internal/inventory"
)

func TestRecipientSummaryMatchesReviewState(t *testing.T) {
	for _, tc := range []struct{ file, conclusion string }{
		{"snapshot-first.json", "判断が完了していません"},
		{"snapshot-empty.json", "主な確認対象は0件"},
	} {
		t.Run(tc.file, func(t *testing.T) {
			s := load(t, tc.file).(*i.Snapshot)
			d, err := makeDocument(s, nil, nil, "ja")
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(d.Pages.Conclusion, tc.conclusion) {
				t.Fatal(d.Pages.Conclusion)
			}
			if d.Pages.Context[0].Value != s.TargetID || len(d.Pages.Coverage) < len(s.Coverage) {
				t.Fatal("summary lost target or coverage")
			}
			if len(d.Pages.FollowUp) != d.Metrics[3].Count {
				t.Fatal("follow-up differs from unreviewed main scope")
			}
			for _, a := range d.Pages.FollowUp {
				found := false
				for _, b := range d.Blocks {
					if a.ID == b.ID {
						found = true
					}
				}
				if !found {
					t.Fatal("missing follow-up destination")
				}
			}
		})
	}
}

func TestRecipientSummaryActionAndAcceptedConclusions(t *testing.T) {
	current := load(t, "snapshot-changed.json").(*i.Snapshot)
	previous := load(t, "snapshot-first.json").(*i.Snapshot)
	annotations := load(t, "assessments-reconfirmed.json").(*i.Assessments)
	d, err := makeDocument(current, previous, annotations, "ja")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(d.Pages.Conclusion, "対応が必要な項目があります") {
		t.Fatal(d.Pages.Conclusion)
	}
	result, err := assessment.Evaluate(current, previous, annotations)
	if err != nil {
		t.Fatal(err)
	}
	for n := range result.Review.Items {
		if result.Review.Items[n].Presence == i.Current {
			result.Review.Items[n].CurrentDecision = i.Accepted
			result.Review.Items[n].Applicability = i.Applicable
		}
	}
	d, err = documentFromReview(current, previous, annotations, "ja", result)
	if err != nil {
		t.Fatal(err)
	}
	if d.Metrics[3].Count != 0 {
		t.Fatal("fixture has unfinished main judgments")
	}
	if len(d.Pages.FollowUp) != 0 || !strings.Contains(d.Pages.Conclusion, "追加対応が必要と判断された項目はありません") {
		t.Fatal(d.Pages.Conclusion)
	}
}

func TestHTMLSeparatesRecipientPages(t *testing.T) {
	current := load(t, "snapshot-first.json").(*i.Snapshot)
	for _, lang := range []string{"en", "ja"} {
		out, err := RenderWithOptions(current, nil, nil, Options{Format: "html", Language: lang})
		if err != nil {
			t.Fatal(err)
		}
		summary := strings.Index(out, `<div id="report-summary"`)
		configuration := strings.Index(out, `<div id="report-configuration"`)
		details := strings.Index(out, `<div id="report-details"`)
		if summary < 0 || configuration <= summary || details <= configuration {
			t.Fatal("missing ordered pages")
		}
		if !strings.Contains(out[summary:configuration], `class="metrics"`) || strings.Contains(out[summary:configuration], `id="item-`) {
			t.Fatal("summary includes item details or lacks counts")
		}
		if !strings.Contains(out[configuration:details], `id="network-map"`) {
			t.Fatal("configuration missing diagram")
		}
		if !strings.Contains(out[details:], `id="scope"`) || !strings.Contains(out[details:], `id="item-`) {
			t.Fatal("details missing scope or judgments")
		}
	}
}
