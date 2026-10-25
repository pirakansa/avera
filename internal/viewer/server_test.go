package viewer

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pirakansa/avera/internal/artifact"
	"github.com/pirakansa/avera/internal/assessment"
	i "github.com/pirakansa/avera/internal/inventory"
	"github.com/pirakansa/avera/internal/reportfile"
)

const testHost = "127.0.0.1:45678"

func session(t *testing.T) *Session {
	t.Helper()
	a, err := artifact.Read("../../test/fixtures/prototype/contracts/snapshot-first.json")
	if err != nil {
		t.Fatal(err)
	}
	f, err := reportfile.New(a.(*i.Snapshot), nil, nil, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	s, err := New(f, t.TempDir(), testHost, "ja")
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func request(s *Session, method, path string, body []byte, origin string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "http://"+testHost+path, bytes.NewReader(body))
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	if method == "POST" {
		r.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	return w
}
func editBody(t *testing.T, s *Session, revision int) []byte {
	t.Helper()
	id := ""
	for _, f := range s.file.Current.Findings {
		if f.Kind == i.Endpoint && f.Identity.(i.EndpointIdentity).LocalPort == 80 {
			id = f.FindingID
		}
	}
	data, err := json.Marshal(editRequest{Revision: revision, Edit: reportfile.Edit{FindingID: id, Decision: i.Accepted, Actor: "fictional-reviewer", Rationale: "Owner judgment", Purpose: `<script>doNotExecute()</script>`, Evidence: []i.EvidenceReference{}}})
	if err != nil {
		t.Fatal(err)
	}
	return data
}
func TestSaveDownloadExportAndRevision(t *testing.T) {
	s := session(t)
	initial := request(s, "GET", "/api/state", nil, "")
	if initial.Code != 200 {
		t.Fatal(initial.Body.String())
	}
	body := editBody(t, s, 0)
	w := request(s, "POST", "/api/assess", body, "http://"+testHost)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	files, err := filepath.Glob(filepath.Join(s.outputDir, "*.json"))
	if err != nil || len(files) != 1 {
		t.Fatal("save missing", err)
	}
	saved, err := reportfile.Read(files[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range saved.Assessments.Entries {
		if a.Actor == "fictional-reviewer" && a.Purpose != `<script>doNotExecute()</script>` {
			t.Fatal("human data changed")
		}
	}
	download := request(s, "GET", "/download.json", nil, "")
	loaded, err := reportfile.Decode(download.Body.Bytes())
	if err != nil || loaded.Current.SnapshotID != saved.Current.SnapshotID {
		t.Fatal("download failed", err)
	}
	for _, path := range []string{"/export.html", "/preview"} {
		w = request(s, "GET", path, nil, "")
		if w.Code != 200 || strings.Contains(w.Body.String(), "<script>") || !strings.Contains(w.Body.String(), "&lt;script&gt;") {
			t.Fatal("unsafe HTML export")
		}
	}
	w = request(s, "POST", "/api/assess", body, "http://"+testHost)
	if w.Code != 409 || s.revision != 1 {
		t.Fatal("stale save accepted")
	}
	w = request(s, "POST", "/api/assess", editBody(t, s, 1), "http://"+testHost)
	if w.Code != 200 || len(s.file.History.Entries) != 1 {
		t.Fatal("history not retained", w.Body.String())
	}
}
func TestRequestBoundariesAndFailedSave(t *testing.T) {
	s := session(t)
	body := editBody(t, s, 0)
	for _, origin := range []string{"", "http://evil.invalid", "null"} {
		if w := request(s, "POST", "/api/assess", body, origin); w.Code != 403 {
			t.Fatal("origin accepted", origin)
		}
	}
	r := httptest.NewRequest("GET", "http://evil.invalid/api/state", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("host rebinding accepted")
	}
	if w = request(s, "POST", "/api/assess", []byte(`{"revision":0,"revision":0}`), "http://"+testHost); w.Code != 400 {
		t.Fatal("duplicate fields accepted")
	}
	bad := bytes.Replace(body, []byte(`"edit":{`), []byte(`"edit":{"basis":{},`), 1)
	if w = request(s, "POST", "/api/assess", bad, "http://"+testHost); w.Code != 400 {
		t.Fatal("immutable fields accepted")
	}
	if w = request(s, "POST", "/api/assess", bytes.Repeat([]byte("x"), MaxEditBytes+1), "http://"+testHost); w.Code != 413 {
		t.Fatal("oversized body accepted")
	}
	before, err := s.file.Encode()
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Remove(s.outputDir); err != nil {
		t.Fatal(err)
	}
	w = request(s, "POST", "/api/assess", body, "http://"+testHost)
	if w.Code != 500 {
		t.Fatal("failed save succeeded")
	}
	after, _ := s.file.Encode()
	if !bytes.Equal(before, after) || s.revision != 0 {
		t.Fatal("failed publication mutated session")
	}
}
func TestPageAndServerScope(t *testing.T) {
	s := session(t)
	for _, path := range []string{"/", "/editor.js", "/editor.css"} {
		w := request(s, "GET", path, nil, "")
		if w.Code != 200 || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("missing bundled asset", path)
		}
	}
	if strings.Contains(string(script), "innerHTML") || strings.Contains(string(script), "eval(") {
		t.Fatal("unsafe DOM sink")
	}
	if err := Serve(context.Background(), "unused", t.TempDir(), "0.0.0.0:0", "ja", &bytes.Buffer{}); err == nil {
		t.Fatal("public listener accepted")
	}
}

func TestReferenceScopeAndHumanLinks(t *testing.T) {
	s := session(t)
	w := request(s, "GET", "/api/state", nil, "")
	var initial struct {
		ReferenceCount int
		Summary        i.Summary
	}
	if err := json.Unmarshal(w.Body.Bytes(), &initial); err != nil {
		t.Fatal(err)
	}
	if initial.ReferenceCount != 1 || initial.Summary.Current != 3 || initial.Summary.Unreviewed != 3 {
		t.Fatal("reference included in main decisions")
	}
	var req editRequest
	if err := json.Unmarshal(editBody(t, s, 0), &req); err != nil {
		t.Fatal(err)
	}
	for _, f := range s.file.Current.Findings {
		if assessment.IsReference(f) {
			req.Edit.RelatedIDs = []string{f.FindingID}
		}
	}
	req.Edit.RouteExplanation = `Proxy route <script>mustRemainText()</script>`
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	w = request(s, "POST", "/api/assess", data, "http://"+testHost)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if len(s.file.Routes) != 1 {
		t.Fatal("relationship not saved")
	}
	exported := request(s, "GET", "/export.html", nil, "")
	if strings.Contains(exported.Body.String(), "<script>") || !strings.Contains(exported.Body.String(), "&lt;script&gt;") || !strings.Contains(exported.Body.String(), "local-reference") {
		t.Fatal("unsafe or incomplete reference export")
	}
	req.Revision = 1
	req.Edit.FindingID = req.Edit.RelatedIDs[0]
	data, err = json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	if w = request(s, "POST", "/api/assess", data, "http://"+testHost); w.Code != 400 || s.revision != 1 {
		t.Fatal("reference-only judgment accepted")
	}
}
