package inventory

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

const fixtureDir = "../../test/fixtures/prototype/contracts"

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, e := os.ReadFile(filepath.Join(fixtureDir, name))
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func artifact(t *testing.T, name string) Artifact {
	t.Helper()
	a, e := Decode(fixture(t, name))
	if e != nil {
		t.Fatalf("%s: %v", name, e)
	}
	return a
}
func TestFixtureRoundTrips(t *testing.T) {
	paths, e := filepath.Glob(filepath.Join(fixtureDir, "*.json"))
	if e != nil {
		t.Fatal(e)
	}
	for _, p := range paths {
		name := filepath.Base(p)
		if name == "scenarios.json" || name == "normalization-vectors.json" {
			continue
		}
		t.Run(name, func(t *testing.T) {
			a := artifact(t, name)
			before, e := Canonical(a)
			if e != nil {
				t.Fatal(e)
			}
			data, e := Encode(a)
			if e != nil {
				t.Fatal(e)
			}
			again, e := Decode(data)
			if e != nil {
				t.Fatal(e)
			}
			encoded, e := Encode(again)
			if e != nil || !bytes.Equal(encoded, data) {
				t.Fatalf("unstable serialization: %v", e)
			}
			after, e := Canonical(a)
			if e != nil || !bytes.Equal(before, after) {
				t.Fatal("Encode changed caller data")
			}
			var original, roundtrip any
			if e = json.Unmarshal(fixture(t, name), &original); e != nil {
				t.Fatal(e)
			}
			if e = json.Unmarshal(data, &roundtrip); e != nil {
				t.Fatal(e)
			}
			if !reflect.DeepEqual(unorderedSets(original, ""), unorderedSets(roundtrip, "")) {
				t.Fatal("fixture fields/text/order lost in round trip")
			}
			if !bytes.HasSuffix(data, []byte("\n")) || !bytes.Contains(data, []byte("\n  \"")) {
				t.Fatal("wrong artifact formatting")
			}
		})
	}
}
func TestScenarioReferences(t *testing.T) {
	var manifest struct {
		Scenarios []struct {
			Name, Snapshot, Expected string
			Previous, Assessments    *string
		}
	}
	if e := json.Unmarshal(fixture(t, "scenarios.json"), &manifest); e != nil {
		t.Fatal(e)
	}
	for _, s := range manifest.Scenarios {
		t.Run(s.Name, func(t *testing.T) {
			current := artifact(t, s.Snapshot).(*Snapshot)
			var previous *Snapshot
			var assessments *Assessments
			if s.Previous != nil {
				previous = artifact(t, *s.Previous).(*Snapshot)
			}
			if s.Assessments != nil {
				assessments = artifact(t, *s.Assessments).(*Assessments)
			}
			review := artifact(t, s.Expected).(*Review)
			if e := ValidateInputs(current, previous, assessments, review); e != nil {
				t.Fatal(e)
			}
		})
	}
}
func TestFixedVectors(t *testing.T) {
	var vectors struct {
		Vectors []struct {
			Kind              Kind
			IdentityCanonical string `json:"identity_canonical"`
			FactsCanonical    string `json:"facts_canonical"`
			FindingID         string `json:"finding_id"`
			FactFingerprint   string `json:"fact_fingerprint"`
		}
		EncodingVector struct {
			Value             any
			Canonical, Digest string
		} `json:"encoding_vector"`
	}
	d := json.NewDecoder(bytes.NewReader(fixture(t, "normalization-vectors.json")))
	d.UseNumber()
	if e := d.Decode(&vectors); e != nil {
		t.Fatal(e)
	}
	snapshot := artifact(t, "snapshot-first.json").(*Snapshot)
	for i, f := range snapshot.Findings {
		v := vectors.Vectors[i]
		id, fp, e := Hashes(snapshot.TargetID, f.Collector, f.Kind, f.Identity, f.Facts)
		if e != nil || id != v.FindingID || fp != v.FactFingerprint {
			t.Fatalf("vector %d: %s %s %v", i, id, fp, e)
		}
		identity, _ := NormalizeIdentity(f.Kind, f.Identity)
		pre, e := Canonical([]any{IdentityVersion, snapshot.TargetID, f.Collector, f.Kind, identity})
		if e != nil || string(pre) != v.IdentityCanonical {
			t.Fatalf("identity preimage %d", i)
		}
		pre, e = Canonical([]any{FactsVersion, f.Kind, identity, normalizeFacts(f.Facts)})
		if e != nil || string(pre) != v.FactsCanonical {
			t.Fatalf("facts preimage %d", i)
		}
	}
	pre, e := Canonical(vectors.EncodingVector.Value)
	if e != nil || string(pre) != vectors.EncodingVector.Canonical {
		t.Fatal("encoding vector", e, string(pre))
	}
	hash, e := digest(vectors.EncodingVector.Value)
	if e != nil || hash != vectors.EncodingVector.Digest {
		t.Fatal("encoding digest", e)
	}
}
func changedFixture(t *testing.T, name string, change func(map[string]any)) []byte {
	t.Helper()
	var m map[string]any
	if e := json.Unmarshal(fixture(t, name), &m); e != nil {
		t.Fatal(e)
	}
	change(m)
	b, e := json.Marshal(m)
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func TestRejectInvalidArtifacts(t *testing.T) {
	tests := []struct {
		name, file, want string
		change           func(map[string]any)
	}{
		{"version", "snapshot-first.json", "unsupported", func(m map[string]any) { m["schema_version"] = "99" }},
		{"revision", "snapshot-first.json", "unsupported", func(m map[string]any) { m["contract_revision"] = "old" }},
		{"missing", "snapshot-first.json", "missing field", func(m map[string]any) { delete(m, "findings") }},
		{"null", "snapshot-first.json", "null", func(m map[string]any) { m["findings"] = nil }},
		{"unknown nested", "snapshot-first.json", "unknown field", func(m map[string]any) { m["environment"].(map[string]any)["extra"] = true }},
		{"timestamp", "snapshot-first.json", "timestamp", func(m map[string]any) { m["collected_at"] = "2026-02-30T00:00:00Z" }},
		{"timestamp offset", "snapshot-first.json", "timestamp", func(m map[string]any) { m["collected_at"] = "2026-10-01T00:00:00+00:00" }},
		{"finish", "snapshot-first.json", "precedes", func(m map[string]any) { m["finished_at"] = "2000-01-01T00:00:00Z" }},
		{"uid range", "snapshot-first.json", "invalid", func(m map[string]any) { m["environment"].(map[string]any)["effective_uid"] = 4294967296 }},
		{"duplicate finding", "snapshot-first.json", "duplicate finding", func(m map[string]any) { a := m["findings"].([]any); m["findings"] = append(a, a[0]) }},
		{"duplicate evidence", "snapshot-first.json", "duplicate evidence", func(m map[string]any) { a := m["evidence"].([]any); m["evidence"] = append(a, a[0]) }},
		{"evidence ref", "snapshot-first.json", "dangling evidence", func(m map[string]any) { m["findings"].([]any)[0].(map[string]any)["evidence_ids"] = []any{"absent"} }},
		{"hash", "snapshot-first.json", "fingerprint mismatch", func(m map[string]any) { m["findings"].([]any)[0].(map[string]any)["fact_fingerprint"] = "sha256:bad" }},
		{"coverage", "snapshot-first.json", "coverage status", func(m map[string]any) { m["coverage"].([]any)[0].(map[string]any)["status"] = "safe" }},
		{"enum", "assessments-reviewed.json", "decision", func(m map[string]any) { m["entries"].([]any)[0].(map[string]any)["decision"] = "safe" }},
		{"assessment duplicate", "assessments-reviewed.json", "duplicate assessment", func(m map[string]any) { a := m["entries"].([]any); m["entries"] = append(a, a[0]) }},
		{"actor", "assessments-reviewed.json", "requires actor", func(m map[string]any) { m["entries"].([]any)[0].(map[string]any)["actor"] = "  " }},
		{"basis discriminator", "assessments-reviewed.json", "unknown field", func(m map[string]any) { m["entries"].([]any)[0].(map[string]any)["origin"] = "advance" }},
		{"unknown expectation", "assessments-advance.json", "unknown field", func(m map[string]any) {
			b := m["entries"].([]any)[0].(map[string]any)["basis"].(map[string]any)
			b["expectation"].(map[string]any)["facts"].(map[string]any)["host"] = "ALL"
		}},
		{"expectation version", "assessments-advance.json", "normalization_version", func(m map[string]any) {
			b := m["entries"].([]any)[0].(map[string]any)["basis"].(map[string]any)
			b["expectation"].(map[string]any)["normalization_version"] = "future"
		}},
		{"summary", "review-first.json", "summary", func(m map[string]any) { m["summary"].(map[string]any)["accepted"] = 1 }},
		{"review dangling", "review-unchanged.json", "dangling", func(m map[string]any) { m["items"].([]any)[1].(map[string]any)["assessment_ids"] = []any{"missing"} }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, e := Decode(changedFixture(t, tt.file, tt.change))
			if e == nil || !strings.Contains(e.Error(), tt.want) {
				t.Fatalf("got %v, want %s", e, tt.want)
			}
		})
	}
}
func TestJSONAmbiguityAndLimits(t *testing.T) {
	for _, bad := range []string{`{"kind":"snapshot","kind":"snapshot"}`, `{"a":{"x":0,"x":1}}`, `"\ud800"`, `"\udc00"`, `"\ud800\u1234"`, `"\ud800x"`, `1.5`, `1e0`, `9007199254740992`, `-9007199254740992`, `{} {}`, "\xef\xbb\xbf{}", string([]byte{'"', 0xff, '"'})} {
		if _, e := parseJSON([]byte(bad)); e == nil {
			t.Errorf("accepted %q", bad)
		}
	}
	for _, good := range []string{`"\ud83d\ude00"`, `"\\ud800"`, `9007199254740991`, `-9007199254740991`, `0`, `-0`} {
		if _, e := parseJSON([]byte(good)); e != nil {
			t.Errorf("rejected %s: %v", good, e)
		}
	}
	tests := []struct {
		name      string
		good, bad []byte
	}{
		{"string", []byte(`"` + strings.Repeat("a", MaxStringBytes) + `"`), []byte(`"` + strings.Repeat("a", MaxStringBytes+1) + `"`)},
		{"depth", []byte(strings.Repeat("[", MaxDepth) + "0" + strings.Repeat("]", MaxDepth)), []byte(strings.Repeat("[", MaxDepth+1) + "0" + strings.Repeat("]", MaxDepth+1))},
		{"count", []byte("[" + strings.Repeat("0,", MaxEntries-1) + "0]"), []byte("[" + strings.Repeat("0,", MaxEntries) + "0]")},
		{"bytes", []byte("0" + strings.Repeat(" ", MaxArtifactBytes-1)), []byte("0" + strings.Repeat(" ", MaxArtifactBytes))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, e := parseJSON(tt.good); e != nil {
				t.Fatal("boundary rejected", e)
			}
			if _, e := parseJSON(tt.bad); e == nil {
				t.Fatal("overflow accepted")
			}
		})
	}
}
func TestNormalizationAndChanges(t *testing.T) {
	for _, tt := range []struct{ family, in, want string }{{"ipv4", "127.000.000.001", "127.0.0.1"}, {"ipv6", "2001:0DB8:0:0:1:0:0:1", "2001:db8::1:0:0:1"}, {"ipv6", "::ffff:192.0.2.1", "::ffff:c000:201"}, {"ipv6", "0:0:0:0:0:0:0:0", "::"}} {
		got, e := normalizeAddress(tt.family, tt.in)
		if e != nil || got != tt.want {
			t.Fatalf("%s: %q %v", tt.in, got, e)
		}
	}
	for _, path := range []string{"relative", "/etc/../sudoers", "/etc/./sudoers"} {
		if _, e := NormalizeIdentity(Rule, SourceIdentity{path, 1, 0}); e == nil {
			t.Fatal("bad source path accepted")
		}
	}
	s := artifact(t, "snapshot-first.json").(*Snapshot)
	repeat := artifact(t, "snapshot-repeat.json").(*Snapshot)
	for i, f := range s.Findings {
		if f.FindingID != repeat.Findings[i].FindingID || f.FactFingerprint != repeat.Findings[i].FactFingerprint {
			t.Fatal("incidental changes affected hashes")
		}
	}
	f := s.Findings[1]
	facts := f.Facts.(NetworkFacts)
	facts.Owners = append(append([]Owner{}, facts.Owners...), Owner{Executable: "/z", UID: 7}, facts.Owners[0])
	f.Facts = facts
	n, e := NormalizeFinding(s.TargetID, f)
	if e != nil {
		t.Fatal(e)
	}
	facts.Owners = slices.Clone(facts.Owners)
	slices.Reverse(facts.Owners)
	f.Facts = facts
	reordered, e := NormalizeFinding(s.TargetID, f)
	if e != nil || n.FactFingerprint != reordered.FactFingerprint {
		t.Fatal("owner set not normalized", e)
	}
	facts.AttributionComplete = false
	f.Facts = facts
	incomplete, e := NormalizeFinding(s.TargetID, f)
	if e != nil || incomplete.FactFingerprint == n.FactFingerprint {
		t.Fatal("completeness omitted", e)
	}
	merged, e := MergeEndpoints(s.TargetID, []Finding{n, incomplete})
	if e != nil || len(merged) != 1 || merged[0].Facts.(NetworkFacts).AttributionComplete {
		t.Fatal("bad merge", e)
	}
	id, fp, e := Hashes("other", n.Collector, n.Kind, n.Identity, n.Facts)
	if e != nil || id == n.FindingID || fp != n.FactFingerprint {
		t.Fatal("target scope", e)
	}
	changed := artifact(t, "snapshot-changed.json").(*Snapshot)
	if changed.Findings[1].FactFingerprint == s.Findings[1].FactFingerprint || changed.Findings[3].FactFingerprint == s.Findings[3].FactFingerprint {
		t.Fatal("meaningful changes omitted")
	}
}
func TestCrossInputRejection(t *testing.T) {
	s := artifact(t, "snapshot-first.json").(*Snapshot)
	a := artifact(t, "assessments-reviewed.json").(*Assessments)
	a.TargetID = "other"
	if e := ValidateInputs(s, nil, a, nil); e == nil || !strings.Contains(e.Error(), "target_id") {
		t.Fatal(e)
	}
	a.TargetID = s.TargetID
	b := a.Entries[0].Basis.(ReviewedBasis)
	b.FactFingerprint = "sha256:" + strings.Repeat("0", 64)
	a.Entries[0].Basis = b
	if e := ValidateInputs(s, nil, a, nil); e == nil || !strings.Contains(e.Error(), "basis fingerprint") {
		t.Fatal(e)
	}
	r := artifact(t, "review-first.json").(*Review)
	r.Items[0].FindingID = "sha256:" + strings.Repeat("0", 64)
	if e := ValidateInputs(s, nil, nil, r); e == nil || !strings.Contains(e.Error(), "dangling") {
		t.Fatal(e)
	}
}
func TestHumanTextPreserved(t *testing.T) {
	a := artifact(t, "assessments-reviewed.json").(*Assessments)
	text := "  日本語 <>&\u2028\n\t rationale  "
	a.Entries[0].Rationale = text
	b, e := Encode(a)
	if e != nil {
		t.Fatal(e)
	}
	decoded, e := Decode(b)
	if e != nil || decoded.(*Assessments).Entries[0].Rationale != text {
		t.Fatal("human text changed", e)
	}
	a.Entries[0].Rationale = string([]byte{0xff})
	if _, e := Encode(a); e == nil {
		t.Fatal("invalid UTF-8 accepted")
	}
}

func unorderedSets(v any, key string) any {
	switch x := v.(type) {
	case map[string]any:
		for k, v := range x {
			x[k] = unorderedSets(v, k)
		}
	case []any:
		for i, v := range x {
			x[i] = unorderedSets(v, "")
		}
		if oneOf(key, "findings", "evidence", "coverage", "entries", "items", "owners", "local_members", "run_as_users", "run_as_groups", "tags", "assessment_ids", "evidence_ids", "reason_codes", "unmatched_assessment_ids", "attempted_sources", "diagnostic_codes", "limitations") {
			slices.SortFunc(x, func(a, b any) int { aa, _ := Canonical(a); bb, _ := Canonical(b); return bytes.Compare(aa, bb) })
		}
	}
	return v
}

func TestNodeLimit(t *testing.T) {
	for _, tt := range []struct {
		start     int
		wantError bool
	}{{MaxNodes - 1, false}, {MaxNodes, true}} {
		d := json.NewDecoder(strings.NewReader("0"))
		d.UseNumber()
		nodes := tt.start
		_, e := readValue(d, 0, &nodes)
		if (e != nil) != tt.wantError {
			t.Fatalf("start %d: %v", tt.start, e)
		}
	}
}
func TestSudoNormalization(t *testing.T) {
	s := artifact(t, "snapshot-first.json").(*Snapshot)
	f := s.Findings[3]
	facts := f.Facts.(SudoFacts)
	facts.RunAsUsers = []string{"root", "daemon", "root"}
	facts.Commands = append([]Command{}, facts.Commands...)
	facts.Commands[0].Tags = []string{"PASSWD", "PASSWD"}
	facts.LocalMembers = append(append([]Member{}, facts.LocalMembers...), Member{Name: "other", UID: 1, Membership: "primary"})
	f.Facts = facts
	n, e := NormalizeFinding(s.TargetID, f)
	if e != nil {
		t.Fatal(e)
	}
	slices.Reverse(facts.RunAsUsers)
	slices.Reverse(facts.LocalMembers)
	f.Facts = facts
	again, e := NormalizeFinding(s.TargetID, f)
	if e != nil || again.FactFingerprint != n.FactFingerprint {
		t.Fatal("sudo sets changed fingerprint", e)
	}
	facts.Commands[0].Arguments = "  --literal 'space text'  "
	f.Facts = facts
	changed, e := NormalizeFinding(s.TargetID, f)
	if e != nil || changed.FactFingerprint == n.FactFingerprint {
		t.Fatal("arguments omitted", e)
	}
	facts = changed.Facts.(SudoFacts)
	facts.PolicyContext = append([]string{}, facts.PolicyContext...)
	slices.Reverse(facts.PolicyContext)
	f.Facts = facts
	reordered, e := NormalizeFinding(s.TargetID, f)
	if e != nil || reordered.FactFingerprint == changed.FactFingerprint {
		t.Fatal("policy order omitted", e)
	}
}
