package report

import (
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	i "github.com/pirakansa/avera/internal/inventory"
)

func TestPresentationScenarios(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(fixtures, "scenarios.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cases struct {
		Scenarios []struct {
			Name, Snapshot, Expected string
			Previous, Assessments    *string
		}
	}
	if err = json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, tc := range cases.Scenarios {
		t.Run(tc.Name, func(t *testing.T) {
			s := load(t, tc.Snapshot).(*i.Snapshot)
			var p *i.Snapshot
			var a *i.Assessments
			if tc.Previous != nil {
				p = load(t, *tc.Previous).(*i.Snapshot)
			}
			if tc.Assessments != nil {
				a = load(t, *tc.Assessments).(*i.Assessments)
			}
			before, _ := json.Marshal([]any{s, p, a})
			summary := load(t, tc.Expected).(*i.Review).Summary
			want := []int{summary.Current, summary.Accepted, summary.ActionRequired, summary.Unreviewed, summary.NeedsReview, summary.Conflict, summary.Historical, summary.UnmatchedAssessments}
			for _, lang := range []string{"ja", "en"} {
				d, err := makeDocument(s, p, a, lang)
				if err != nil {
					t.Fatal(err)
				}
				counts := []int{}
				for _, m := range d.Metrics {
					counts = append(counts, m.Count)
				}
				if !reflect.DeepEqual(counts, want) {
					t.Fatalf("%s counts %v != %v", lang, counts, want)
				}
				for _, format := range []string{"html", "markdown"} {
					o := Options{Format: format, Language: lang}
					out, err := RenderWithOptions(s, p, a, o)
					if err != nil {
						t.Fatal(err)
					}
					again, err := RenderWithOptions(s, p, a, o)
					if err != nil || out != again {
						t.Fatal("nondeterministic report", err)
					}
					if format == "html" {
						if !strings.Contains(out, `<html lang="`+lang+`">`) {
							t.Fatal("missing document language")
						}
						for _, m := range d.Metrics {
							if !strings.Contains(out, fmt.Sprintf("<strong>%d</strong><span>%s</span>", m.Count, m.Label)) {
								t.Fatal("missing metric")
							}
						}
						ids := map[string]bool{}
						for _, match := range regexp.MustCompile(` id="([^"]+)"`).FindAllStringSubmatch(out, -1) {
							if ids[match[1]] {
								t.Fatal("duplicate id")
							}
							ids[match[1]] = true
						}
						for _, match := range regexp.MustCompile(`href="#([^"]+)"`).FindAllStringSubmatch(out, -1) {
							if !ids[match[1]] {
								t.Fatalf("broken link %s", match[1])
							}
						}
					}
					if lang == "ja" && !strings.Contains(out, "確認レポート") {
						t.Fatal("missing Japanese title")
					}
					if lang == "ja" && tc.Name == "changed" && !strings.Contains(html.UnescapeString(out), "今回「追加対応不要」と判断したことを意味しません") {
						t.Fatal("stale acceptance not explained")
					}
				}
			}
			after, _ := json.Marshal([]any{s, p, a})
			if string(before) != string(after) {
				t.Fatal("inputs mutated")
			}
		})
	}
}
func TestHTMLTreatsUntrustedContentAsText(t *testing.T) {
	s := load(t, "snapshot-repeat.json").(*i.Snapshot)
	p := load(t, "snapshot-first.json").(*i.Snapshot)
	a := load(t, "assessments-reviewed.json").(*i.Assessments)
	payload := `</dd><script>alert(1)</script><img src="https://invalid.example/track" onerror="alert(1)">`
	a.Entries[0].Purpose = payload + "\u202e\n"
	a.Entries[0].Rationale = "翻訳しない原文 accepted"
	a.Entries[0].EvidenceRefs = []i.EvidenceReference{{Reference: "javascript:alert(1)", Provenance: "human_reported"}}
	for _, lang := range []string{"en", "ja"} {
		out, err := RenderWithOptions(s, p, a, Options{Format: "html", Language: lang})
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(out, "<script") || strings.Contains(out, "<img") || strings.Contains(out, `href="javascript:`) || strings.Contains(out, `href="http`) {
			t.Fatal("active untrusted markup")
		}
		if !strings.Contains(out, "&lt;script&gt;") || !strings.Contains(out, "翻訳しない原文 accepted") || !strings.Contains(out, `\u202E\u000A`) {
			t.Fatal("lost or hidden source content")
		}
		if !strings.Contains(out, "default-src &#39;none&#39;") && !strings.Contains(out, "default-src 'none'") {
			t.Fatal("missing restrictive CSP")
		}
	}
}
func TestPresentationOptionsAndCompatibility(t *testing.T) {
	s := load(t, "snapshot-first.json").(*i.Snapshot)
	expected, err := Render(s, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := RenderWithOptions(s, nil, nil, Options{})
	if err != nil || got != expected {
		t.Fatal("changed default Markdown")
	}
	for _, o := range []Options{{Format: "pdf"}, {Language: "fr"}, {Format: "HTML"}} {
		if _, err := RenderWithOptions(s, nil, nil, o); err == nil {
			t.Fatalf("accepted invalid options %v", o)
		}
	}
	for _, o := range []Options{{Format: "html", Language: "ja"}, {Format: "markdown", Language: "ja"}} {
		out, err := RenderWithOptions(s, nil, nil, o)
		if err != nil || !strings.Contains(html.UnescapeString(out), "前回のスナップショットが指定されていません") {
			t.Fatal("missing unavailable comparison", err)
		}
	}
}

func TestHTMLTemplateRequiresEscapingForEveryInterpolation(t *testing.T) {
	for _, action := range regexp.MustCompile(`\{\{([^}]+)\}\}`).FindAllStringSubmatch(htmlReport, -1) {
		expression := strings.TrimSpace(action[1])
		if strings.HasPrefix(expression, "range ") || strings.HasPrefix(expression, "if ") || expression == "else" || expression == "end" {
			continue
		}
		if !strings.HasSuffix(expression, "| escape") {
			t.Fatalf("unescaped template interpolation: %s", expression)
		}
	}
}

func TestOutstandingSummaryEmphasis(t *testing.T) {
	current := load(t, "snapshot-changed.json").(*i.Snapshot)
	previous := load(t, "snapshot-first.json").(*i.Snapshot)
	annotations := load(t, "assessments-reconfirmed.json").(*i.Assessments)
	for _, lang := range []string{"ja", "en"} {
		output, err := RenderWithOptions(current, previous, annotations, Options{Format: "html", Language: lang})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(output, `class="metric metric-action"`) || !strings.Contains(output, `class="metric metric-unreviewed"`) {
			t.Fatal("outstanding counts are not highlighted")
		}
		for _, label := range []string{localizer{lang == "ja"}.text("Prioritize action", "優先して対応"), localizer{lang == "ja"}.text("Review needed", "判断が必要")} {
			if !strings.Contains(output, label) {
				t.Fatal("missing text cue", label)
			}
		}
	}
	empty := load(t, "snapshot-empty.json").(*i.Snapshot)
	output, err := RenderWithOptions(empty, nil, nil, Options{Format: "html", Language: "ja"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output, `class="metric metric-action"`) || strings.Contains(output, `class="metric metric-unreviewed"`) || strings.Contains(output, `class="metric-notice"`) {
		t.Fatal("zero counts displayed as outstanding")
	}
}

func TestJapaneseAcceptedLabel(t *testing.T) {
	current := load(t, "snapshot-repeat.json").(*i.Snapshot)
	previous := load(t, "snapshot-first.json").(*i.Snapshot)
	annotations := load(t, "assessments-reviewed.json").(*i.Assessments)
	for _, format := range []string{"html", "markdown"} {
		output, err := RenderWithOptions(current, previous, annotations, Options{Format: format, Language: "ja"})
		if err != nil {
			t.Fatal(err)
		}
		visible := html.UnescapeString(output)
		if !strings.Contains(visible, "追加対応不要") || strings.Contains(visible, "受容済み") || strings.Contains(visible, "問題なし") {
			t.Fatal("inconsistent Japanese judgment label")
		}
	}
	if localizerState := (localizer{ja: true}).state("accepted"); localizerState != "追加対応不要" {
		t.Fatal("wrong item judgment label")
	}
}
