// Package viewer provides a local-only browser editor for portable reports.
package viewer

import (
	"bytes"
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/pirakansa/avera/internal/artifact"
	"github.com/pirakansa/avera/internal/assessment"
	"github.com/pirakansa/avera/internal/guidance"
	i "github.com/pirakansa/avera/internal/inventory"
	"github.com/pirakansa/avera/internal/report"
	"github.com/pirakansa/avera/internal/reportfile"
)

const (
	DefaultListen  = "127.0.0.1:0"
	MaxEditBytes   = 1 << 20
	headerTimeout  = 5 * time.Second
	requestTimeout = 30 * time.Second
	idleTimeout    = 60 * time.Second
	maxHeaders     = 16 << 10
)

//go:embed editor.html
var page []byte

//go:embed editor.js
var script []byte

//go:embed editor.css
var styles []byte

type Session struct {
	mu                                sync.Mutex
	file                              *reportfile.File
	revision                          int
	outputDir, host, origin, language string
	now                               func() time.Time
}

func New(f *reportfile.File, outputDir, host, language string) (*Session, error) {
	if err := report.ValidateOptions(report.Options{Format: "html", Language: language}); err != nil {
		return nil, err
	}
	data, err := f.Encode()
	if err != nil {
		return nil, err
	}
	copy, err := reportfile.Decode(data)
	if err != nil {
		return nil, err
	}
	if err = artifact.CheckDestination(filepath.Join(outputDir, "preflight-"+rand.Text())); err != nil {
		return nil, fmt.Errorf("output directory must exist and permit private file creation")
	}
	return &Session{file: copy, outputDir: outputDir, host: host, origin: "http://" + host, language: language, now: time.Now}, nil
}
func (s *Session) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self'; connect-src 'self'; frame-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'")
	if r.Host != s.host || (r.Header.Get("Origin") != "" && r.Header.Get("Origin") != s.origin) || r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		http.Error(w, "local origin required", http.StatusForbidden)
		return
	}
	if r.Method == http.MethodPost {
		if r.URL.Path != "/api/assess" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Origin") != s.origin || r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, "same-origin JSON request required", http.StatusForbidden)
			return
		}
		s.save(w, r)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	switch r.URL.Path {
	case "/":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(page)
	case "/editor.js":
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		_, _ = w.Write(script)
	case "/editor.css":
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		_, _ = w.Write(styles)
	case "/api/state":
		s.state(w)
	case "/download.json":
		s.download(w)
	case "/export.html", "/preview":
		s.html(w, r.URL.Path == "/export.html")
	default:
		http.NotFound(w, r)
	}
}

type item struct {
	Finding   i.Finding        `json:"finding"`
	Review    i.ReviewItem     `json:"review"`
	Guidance  guidance.Content `json:"guidance"`
	Judgments []i.Assessment   `json:"judgments"`
}
type state struct {
	Revision    int `json:"revision"`
	Target, Run string
	Summary     i.Summary
	Coverage    []i.Coverage
	Items       []item
	History     *i.Assessments
	Unmatched   *i.Assessments
	Language    string
}

func (s *Session) state(w http.ResponseWriter) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result, err := assessment.Evaluate(s.file.Current, s.file.Previous, s.file.Assessments)
	if err != nil {
		http.Error(w, "cannot evaluate report", http.StatusInternalServerError)
		return
	}
	out := state{Revision: s.revision, Target: s.file.Current.TargetID, Run: s.file.Current.SnapshotID, Summary: result.Review.Summary, Coverage: s.file.Current.Coverage, Items: []item{}, History: s.file.History, Unmatched: result.Unmatched, Language: s.language}
	byID := map[string]i.Assessment{}
	for _, a := range s.file.Assessments.Entries {
		byID[a.AssessmentID] = a
	}
	byFinding := map[string]i.ReviewItem{}
	for _, v := range result.Review.Items {
		byFinding[v.FindingID] = v
	}
	for _, f := range s.file.Current.Findings {
		v := byFinding[f.FindingID]
		it := item{Finding: f, Review: v, Guidance: guidance.ForLanguage(f.Kind, s.language), Judgments: []i.Assessment{}}
		for _, id := range v.AssessmentIDs {
			it.Judgments = append(it.Judgments, byID[id])
		}
		out.Items = append(out.Items, it)
	}
	writeJSON(w, out)
}
func writeJSON(w http.ResponseWriter, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		http.Error(w, "cannot encode response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(data)
}

type editRequest struct {
	Revision int             `json:"revision"`
	Edit     reportfile.Edit `json:"edit"`
}

func (s *Session) save(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxEditBytes))
	if err != nil {
		http.Error(w, "edit exceeds size limit", http.StatusRequestEntityTooLarge)
		return
	}
	if err = i.ValidateJSON(data); err != nil {
		http.Error(w, "invalid edit JSON", http.StatusBadRequest)
		return
	}
	var req editRequest
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err = dec.Decode(&req); err != nil {
		http.Error(w, "invalid edit fields", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if req.Revision != s.revision {
		http.Error(w, "another save changed this report; reload before editing again", http.StatusConflict)
		return
	}
	updated, err := s.file.Apply(req.Edit, s.now())
	if err != nil {
		http.Error(w, "invalid judgment: provide decision, actor and rationale; only current items are editable", http.StatusBadRequest)
		return
	}
	name := "report-" + s.now().UTC().Format("20060102T150405Z") + "-" + rand.Text() + ".json"
	if err = updated.Write(r.Context(), filepath.Join(s.outputDir, name)); err != nil {
		http.Error(w, "save failed; no in-memory judgment was changed", http.StatusInternalServerError)
		return
	}
	s.file = updated
	s.revision++
	writeJSON(w, map[string]any{"revision": s.revision, "saved": name})
}
func (s *Session) download(w http.ResponseWriter) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.file.Encode()
	if err != nil {
		http.Error(w, "cannot encode report", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="avera-report.json"`)
	_, _ = w.Write(data)
}
func (s *Session) html(w http.ResponseWriter, attachment bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	text, err := report.RenderWithOptions(s.file.Current, s.file.Previous, s.file.Assessments, report.Options{Format: "html", Language: s.language})
	if err != nil {
		http.Error(w, "cannot render report", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; frame-ancestors 'self'; base-uri 'none'; form-action 'none'")
	if attachment {
		w.Header().Set("Content-Disposition", `attachment; filename="avera-report.html"`)
	}
	_, _ = io.WriteString(w, text)
}

// Serve starts only a literal loopback listener and never launches external commands.
func Serve(ctx context.Context, input, outputDir, listen, language string, diagnostics io.Writer) error {
	host, _, err := net.SplitHostPort(listen)
	if err != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() {
		return fmt.Errorf("--listen requires a literal loopback IP and port")
	}
	f, err := reportfile.Read(input)
	if err != nil {
		return err
	}
	if err = artifact.CheckDestination(filepath.Join(outputDir, "preflight-"+rand.Text())); err != nil {
		return fmt.Errorf("--output-dir must exist with trusted private write access")
	}
	listener, err := net.Listen("tcp", listen)
	if err != nil {
		return fmt.Errorf("cannot open loopback listener")
	}
	defer listener.Close()
	session, err := New(f, outputDir, listener.Addr().String(), language)
	if err != nil {
		return err
	}
	server := &http.Server{Handler: session, ReadHeaderTimeout: headerTimeout, ReadTimeout: requestTimeout, WriteTimeout: requestTimeout, IdleTimeout: idleTimeout, MaxHeaderBytes: maxHeaders}
	if _, err = fmt.Fprintf(diagnostics, "Open http://%s in your browser. Saves create new report JSON files in %q. Stop with Ctrl+C.\n", listener.Addr(), outputDir); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	select {
	case err = <-done:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), headerTimeout)
		defer cancel()
		err = server.Shutdown(shutdown)
		if err != nil {
			err = errors.Join(err, server.Close())
		}
		<-done
		return err
	}
}
