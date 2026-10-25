package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pirakansa/avera/internal/collect/accounts"
	"github.com/pirakansa/avera/internal/collect/sudo"
	i "github.com/pirakansa/avera/internal/inventory"
	"github.com/pirakansa/avera/internal/reportfile"
)

func TestAccountAndSudoCollectEditAndRepeat(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "etc"), 0700); err != nil {
		t.Fatal(err)
	}
	for name, text := range map[string]string{
		"passwd":  "root:x:0:0::/root:/bin/sh\noperator:x:1000:1000::/home/operator:/bin/sh\n",
		"group":   "root:x:0:\noperator:x:1000:\nmaint:x:3000:operator\n",
		"sudoers": "%maint ALL=(root) NOPASSWD: /usr/bin/id\n",
	} {
		if err := os.WriteFile(filepath.Join(root, "etc", name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	source := sudo.LocalSource{Root: root}
	svc := Service{EffectiveUID: func() int { return 0 }, Collectors: []Registration{{Name: i.Accounts, Collector: accounts.Collector{Source: source}}, {Name: i.Sudo, Collector: sudo.Collector{Source: source, Now: func() time.Time { return time.Date(2026, 10, 25, 1, 2, 3, 456, time.UTC) }}}}}
	dir := t.TempDir()
	out := filepath.Join(dir, "first.json")
	code, err := svc.CollectReport(context.Background(), Options{TargetID: "fictional", Output: out})
	if err != nil || code != 0 {
		t.Fatal(code, err)
	}
	file, err := reportfile.Read(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(file.Current.Findings) != 3 {
		t.Fatal("account/sudo registration lost findings")
	}
	accountID := ""
	for _, finding := range file.Current.Findings {
		if identity, ok := finding.Identity.(i.AccountIdentity); ok && identity.Name == "operator" {
			accountID = finding.FindingID
		}
	}
	edit := reportfile.Edit{FindingID: accountID, Purpose: "Maintenance; fictional operations team", Rationale: "Configured account memberships match required duties", Actor: "fictional-reviewer", Decision: i.Accepted, Evidence: []i.EvidenceReference{}}
	file, err = file.Apply(edit, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	edited := filepath.Join(dir, "edited.json")
	if err = file.Write(context.Background(), edited); err != nil {
		t.Fatal(err)
	}
	second := filepath.Join(dir, "second.json")
	code, err = svc.CollectReport(context.Background(), Options{Input: edited, Output: second})
	if err != nil || code != 0 {
		t.Fatal(code, err)
	}
	repeated, err := reportfile.Read(second)
	if err != nil {
		t.Fatal(err)
	}
	review, err := repeated.Review()
	if err != nil {
		t.Fatal(err)
	}
	accepted := false
	for _, item := range review.Review.Items {
		if item.FindingID == accountID {
			accepted = item.CurrentDecision == i.Accepted
		}
	}
	if !accepted {
		t.Fatal("unchanged account judgment not reused")
	}
	if err = os.WriteFile(filepath.Join(root, "etc/group"), []byte("root:x:0:\noperator:x:1000:\nmaint:x:3000:\n"), 0600); err != nil {
		t.Fatal(err)
	}
	third := filepath.Join(dir, "third.json")
	_, err = svc.CollectReport(context.Background(), Options{Input: second, Output: third})
	if err != nil {
		t.Fatal(err)
	}
	changed, err := reportfile.Read(third)
	if err != nil {
		t.Fatal(err)
	}
	changedReview, err := changed.Review()
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range changedReview.Review.Items {
		if item.FindingID == accountID && item.Applicability != i.NeedsReview {
			t.Fatal("membership change reused account judgment", item)
		}
	}
}
