package report

import (
	"html"
	"reflect"
	"strings"
	"testing"

	i "github.com/pirakansa/avera/internal/inventory"
)

func TestServiceInventoryProjectionAndEscaping(t *testing.T) {
	snapshot := load(t, "snapshot-first.json").(*i.Snapshot)
	snapshot.ContractRevision = i.ContractRevision
	uid := uint32(0)
	payload := `worker <script>window.injected=1</script>`
	var findingID string
	evidence := snapshot.Evidence[0].EvidenceID
	for _, f := range snapshot.Findings {
		for _, id := range f.EvidenceIDs {
			if id == evidence {
				findingID = f.FindingID
			}
		}
	}
	values := snapshot.Evidence[0].Values.(i.ProcessValues)
	values.Service = &i.ServiceObservation{ProcessName: payload, EffectiveUID: &uid, UnitCandidate: "fictional.service", CgroupPath: "/system.slice/fictional.service", Diagnostics: []string{}}
	snapshot.Evidence[0].Values = values
	for _, language := range []string{"en", "ja"} {
		views := EndpointServices(snapshot, language)
		view := views[findingID]
		if !strings.Contains(view.Summary, "fictional.service") || !strings.Contains(view.Summary, "UID 0") {
			t.Fatal("missing observed service", view)
		}
		for id, other := range views {
			if id != findingID && strings.Contains(other.Summary, "fictional.service") {
				t.Fatal("service leaked to another endpoint")
			}
		}
		for _, format := range []string{"html", "markdown"} {
			text, err := RenderWithOptions(snapshot, nil, nil, Options{Format: format, Language: language})
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(text, "<script>") || !strings.Contains(html.UnescapeString(text), payload) || !strings.Contains(html.UnescapeString(text), "fictional.service") {
				t.Fatal("missing or unescaped service metadata")
			}
		}
	}
}
func TestLegacyServiceInventoryIsExplicitlyUnknown(t *testing.T) {
	snapshot := load(t, "snapshot-first.json").(*i.Snapshot)
	views := EndpointServices(snapshot, "ja")
	for _, view := range views {
		if view.Summary != "" {
			t.Fatal("invented service identification")
		}
		unknown := false
		executable := false
		for _, f := range view.Fields {
			if f.Value == "不明・未収集" {
				unknown = true
			}
			if strings.Contains(f.Label, "実行ファイル") {
				executable = true
			}
		}
		if !unknown || !executable {
			t.Fatal("lost legacy attribution or unknowns", view)
		}
	}
}

func TestServiceInventoryOmitsTransientPID(t *testing.T) {
	snapshot := load(t, "snapshot-first.json").(*i.Snapshot)
	before := EndpointServices(snapshot, "ja")
	for n, entry := range snapshot.Evidence {
		if values, ok := entry.Values.(i.ProcessValues); ok && values.PID != nil {
			pid := *values.PID + 50000
			values.PID = &pid
			snapshot.Evidence[n].Values = values
		}
	}
	after := EndpointServices(snapshot, "ja")
	if !reflect.DeepEqual(before, after) {
		t.Fatal("PID changes affected the displayed inventory")
	}
	for _, language := range []string{"ja", "en"} {
		for _, view := range EndpointServices(snapshot, language) {
			for _, field := range view.Fields {
				if strings.Contains(field.Label, "PID") {
					t.Fatal("PID exposed in service inventory")
				}
			}
		}
	}
}
