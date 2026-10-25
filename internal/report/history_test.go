package report

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/pirakansa/avera/internal/assessment"
	i "github.com/pirakansa/avera/internal/inventory"
	"github.com/pirakansa/avera/internal/reportfile"
)

func TestReplacedHistoryHiddenWithoutChangingPortableData(t *testing.T) {
	snapshot := load(t, "snapshot-first.json").(*i.Snapshot)
	f, err := reportfile.New(snapshot, nil, nil, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	var entrance, local i.Finding
	for _, v := range snapshot.Findings {
		if assessment.IsReference(v) {
			local = v
		}
		if v.Kind == i.Endpoint && v.Identity.(i.EndpointIdentity).LocalPort == 80 {
			entrance = v
		}
	}
	edit := reportfile.Edit{FindingID: entrance.FindingID, Decision: i.Accepted, Actor: "fictional-reviewer", Rationale: "ReplacedJudgmentMarker", RelatedIDs: []string{local.FindingID}, RouteExplanation: "ReplacedRouteMarker"}
	f, err = f.Apply(edit, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	edit.Rationale = "CurrentAcceptedMarker"
	edit.RouteExplanation = "CurrentRouteMarker"
	f, err = f.Apply(edit, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	before, err := f.Encode()
	if err != nil {
		t.Fatal(err)
	}
	if len(f.History.Entries) != 1 || len(f.RouteHistory) != 1 {
		t.Fatal("setup did not retain history")
	}
	for _, language := range []string{"ja", "en"} {
		for _, format := range []string{"html", "markdown"} {
			rendered, err := RenderFile(f, Options{Format: format, Language: language})
			if err != nil {
				t.Fatal(err)
			}
			for _, hidden := range []string{"ReplacedJudgmentMarker", "ReplacedRouteMarker", "置き換え前", "Superseded"} {
				if strings.Contains(rendered, hidden) {
					t.Fatalf("%s/%s exposes %s", language, format, hidden)
				}
			}
			for _, current := range []string{"CurrentAcceptedMarker", "CurrentRouteMarker"} {
				if !strings.Contains(rendered, current) {
					t.Fatalf("current context lost: %s", current)
				}
			}
		}
	}
	after, err := f.Encode()
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("render changed stored history", err)
	}
}
