package report

import (
	"bytes"
	"fmt"
	"html"
	"strings"
	"text/template"
	"unicode"

	"github.com/pirakansa/avera/internal/assessment"
	"github.com/pirakansa/avera/internal/guidance"
	i "github.com/pirakansa/avera/internal/inventory"
	"github.com/pirakansa/avera/internal/reportfile"
)

// Options selects presentation only; source values and assessment rules are unchanged.
type Options struct{ Format, Language string }

func (o Options) normalized() Options {
	if o.Format == "" {
		o.Format = "markdown"
	}
	if o.Language == "" {
		o.Language = "en"
	}
	return o
}

// ValidateOptions rejects unsupported output choices before publication.
func ValidateOptions(o Options) error {
	o = o.normalized()
	if o.Format != "markdown" && o.Format != "html" {
		return fmt.Errorf("--format: choose markdown or html")
	}
	if o.Language != "en" && o.Language != "ja" {
		return fmt.Errorf("--lang: choose en or ja")
	}
	return nil
}

type field struct{ Label, Value string }
type link struct{ ID, Label string }
type block struct {
	Technical       bool
	ID, Title, Note string
	FindingID       string
	SourceIndex     int
	Fields          []field
	Questions       []string
	Links           []link
}
type metric struct {
	Label        string
	Count        int
	Tone, Notice string
}
type reportPages struct {
	Summary, Configuration, Details, ReadingGuide, Conclusion string
	Context, Coverage                                         []field
	FollowUp                                                  []link
}
type document struct {
	Boundary                                             string
	Pages                                                reportPages
	Network                                              *NetworkMap
	Physical                                             *PhysicalMap
	Language, Title, Warning, Summary, Contents, Details string
	Metrics                                              []metric
	Blocks                                               []block
}
type localizer struct{ ja bool }

func (l localizer) text(en, ja string) string {
	if l.ja {
		return ja
	}
	return en
}
func (l localizer) state(s string) string {
	if !l.ja {
		return s
	}
	names := map[string]string{"accepted": "追加対応不要", "action_required": "対応が必要", "unreviewed": "未確認", "applicable": "適用可能", "needs_review": "再確認が必要", "conflict": "判断の競合", "none": "判断なし", "new": "新規", "unchanged": "変更なし", "changed": "変更あり", "unknown": "不明", "not_observed": "今回は未観測", "not_compared": "比較なし", "complete": "収集完了", "partial": "一部収集", "failed": "収集失敗", "not_collected": "未収集", "advance": "事前の判断", "reviewed_snapshot": "観測後の判断", "purpose": "目的", "exposure": "利用者・公開範囲", "safeguards": "保護策", "rationale": "判断理由", "supporting evidence references": "裏付ける証拠の参照"}
	if v, ok := names[s]; ok {
		return v
	}
	return s
}
func (l localizer) reasons(codes []string) string {
	if !l.ja {
		return reasonText(codes)
	}
	names := map[string]string{reportfile.SharedJudgment: "同じサービス構成に対する共通設定の判断を適用しています（このPC固有の安全性の検証ではありません）", reportfile.RelatedContextChanged: "関連するローカルサービスが変化したか未観測です。入口側の判断と経路の説明を再確認してください", assessment.FactsChanged: "観測内容が変わりました。以前の説明を判断者と再確認してください", assessment.CoverageIncomplete: "収集や帰属の特定が不完全です。判断の再利用前に不足する情報を確認してください", assessment.MultipleAssessments: "複数の判断が競合しています。どの判断を適用するか担当者と確認してください", assessment.ExpectationMismatch: "事前の想定が不完全、または観測内容と異なります。作成者と確認してください", assessment.NotObserved: "比較可能な今回の収集では未観測です。対処済みとは限りません"}
	out := []string{}
	for _, c := range codes {
		v, ok := names[c]
		if !ok {
			v = "この診断の意味を担当者に確認してください"
		}
		out = append(out, v+" ("+c+")")
	}
	return strings.Join(out, "; ")
}
func (l localizer) annotation(b *block, a i.Assessment, context string) {
	b.Fields = append(b.Fields, field{l.text("Assessment context", "判断の位置付け"), context}, field{l.text("Assessment ID", "判断 ID"), a.AssessmentID}, field{l.text("Origin", "判断の由来"), l.state(string(a.Origin))}, field{l.text("Human decision", "人による判断"), l.state(string(a.Decision))}, field{l.text("Actor / date", "判断者・日時"), a.Actor + " / " + a.AssessedAt}, field{l.text("Purpose", "目的"), a.Purpose})
	if a.Selector.Kind != i.ExternalStorage || a.Exposure != "" {
		b.Fields = append(b.Fields, field{l.text("Audience / exposure", "利用者・公開範囲"), a.Exposure})
	}
	if a.Selector.Kind != i.ExternalStorage || a.Safeguards != "" {
		b.Fields = append(b.Fields, field{l.text("Safeguards (human statement, not verified)", "保護策（人の申告・ツール未検証）"), a.Safeguards})
	}
	b.Fields = append(b.Fields, field{l.text("Rationale", "判断理由"), a.Rationale})
	for _, ref := range a.EvidenceRefs {
		b.Fields = append(b.Fields, field{l.text("Evidence reference (not fetched)", "証拠の参照（取得しません）"), ref.Reference + " / " + ref.Provenance})
	}
	missing := guidance.Missing(a)
	for n, v := range missing {
		missing[n] = l.state(v)
	}
	if len(missing) > 0 {
		b.Fields = append(b.Fields, field{l.text("Missing explanations (completeness only, not a verified defect)", "不足する説明（記入状況の確認であり、欠陥の検証ではありません）"), strings.Join(missing, ", ")})
	}
}
func makeDocument(current, previous *i.Snapshot, annotations *i.Assessments, language string) (document, error) {
	result, err := assessment.Evaluate(current, previous, annotations)
	if err != nil {
		return document{}, err
	}
	return documentFromReview(current, previous, annotations, language, result)
}
func documentFromReview(current, previous *i.Snapshot, annotations *i.Assessments, language string, result *assessment.Result) (document, error) {
	l := localizer{language == "ja"}
	r := result.Review
	s, referenceCount := assessment.ReviewSummary(current, r)
	d := document{Language: language, Title: l.text("AVERA review report", "AVERA 確認レポート"), Summary: l.text("Review overview", "確認状況の概要"), Contents: l.text("Contents", "目次"), Details: l.text("Technical details", "技術情報を開く"), Warning: l.text("Counts describe review state, not overall safety. Missing observations do not prove absence or remediation. Human explanations are not tool-verified safeguards.", "件数は確認状況を表し、全体の安全性を示すものではありません。「追加対応不要」は人による判断です。未観測は不存在や対処済みを意味しません。人による説明や保護策はツールで検証した事実ではありません。")}
	d.Boundary = AccountBoundary(language)
	d.Pages = reportPages{
		Summary: l.text("Report summary", "報告サマリー"), Configuration: l.text("Configuration overview", "構成の概要"), Details: l.text("Judgments and evidence", "判断・根拠の詳細"),
		ReadingGuide: l.text("Start here for the conclusion and required follow-up. Configuration and individual evidence are available on the other pages when needed.", "報告の結論と必要な対応は、このページで確認できます。構成や個別の判断根拠は、必要に応じて他のページで確認してください。"),
		Context:      []field{{l.text("Target", "対象"), current.TargetID}, {l.text("Collection start / finish", "収集開始・終了"), current.CollectedAt + " / " + current.FinishedAt}},
	}
	switch {
	case s.ActionRequired > 0:
		d.Pages.Conclusion = l.text("Human judgments identify items requiring action. Review the follow-up items below.", "人による判断で、対応が必要な項目があります。下の確認・対応事項を確認してください。")
	case s.Unreviewed > 0 || s.NeedsReview > 0 || s.Conflict > 0:
		d.Pages.Conclusion = l.text("Judgments remain unfinished. Review or reconfirm the items below before concluding the report.", "判断が完了していません。報告の結論を確定する前に、下の項目の判断・再確認が必要です。")
	case s.Current == 0:
		d.Pages.Conclusion = l.text("There are no items in the main review scope. Check collection coverage before drawing a conclusion.", "主な確認対象は0件です。結論を出す前に、収集範囲と確認できていない事項を確認してください。")
	default:
		d.Pages.Conclusion = l.text("Within the reviewed scope, no current items are judged to require additional action. Check collection coverage separately.", "確認した範囲では、追加対応が必要と判断された項目はありません。収集範囲と確認できていない事項も確認してください。")
	}
	for _, c := range current.Coverage {
		d.Pages.Coverage = append(d.Pages.Coverage, field{string(c.Collector), l.state(string(c.Status)) + " / " + c.Scope})
		if len(c.Limitations) > 0 {
			d.Pages.Coverage = append(d.Pages.Coverage, field{l.text("Limitations (source text)", "制限事項（情報源の原文）"), strings.Join(c.Limitations, "; ")})
		}
		if len(c.DiagnosticCodes) > 0 {
			d.Pages.Coverage = append(d.Pages.Coverage, field{l.text("Diagnostics", "診断コード"), strings.Join(c.DiagnosticCodes, ", ")})
		}
	}
	storageCollected := false
	for _, coverage := range current.Coverage {
		if coverage.Collector == i.Storage {
			storageCollected = true
		}
	}
	if !storageCollected {
		d.Pages.Coverage = append(d.Pages.Coverage, field{l.text("External storage", "外付けストレージ"), StorageStatus(current, language)})
	}
	d.Metrics = []metric{
		{Label: l.text("Current items", "主な確認対象"), Count: s.Current},
		{Label: l.text("Accepted", "追加対応不要"), Count: s.Accepted},
		{Label: l.text("Action required", "対応が必要"), Count: s.ActionRequired, Tone: "metric-action", Notice: l.text("Prioritize action", "優先して対応")},
		{Label: l.text("Unreviewed", "未確認"), Count: s.Unreviewed, Tone: "metric-unreviewed", Notice: l.text("Review needed", "判断が必要")},
		{Label: l.text("Needs re-review (subset)", "再確認が必要（内数）"), Count: s.NeedsReview},
		{Label: l.text("Conflicts (subset)", "判断の競合（内数）"), Count: s.Conflict},
		{Label: l.text("Historical items", "過去の項目"), Count: s.Historical},
		{Label: l.text("Unmatched assessments", "対応する観測がない判断"), Count: s.UnmatchedAssessments},
	}
	d.Metrics = append(d.Metrics, metric{Label: l.text("Local-only references (outside review counts)", "ローカル限定の参考情報（集計外）"), Count: referenceCount})
	for n := range d.Metrics {
		if d.Metrics[n].Count == 0 {
			d.Metrics[n].Tone = ""
			d.Metrics[n].Notice = ""
		}
	}
	env, err := technical(current.Environment)
	if err != nil {
		return d, err
	}
	meta := block{ID: "run", Title: l.text("Target and run", "対象と実行情報"), Fields: []field{{l.text("Target", "対象"), current.TargetID}, {l.text("Run", "実行 ID"), current.SnapshotID}, {l.text("Collection start / finish", "収集開始・終了"), current.CollectedAt + " / " + current.FinishedAt}, {l.text("Tool version", "ツールのバージョン"), current.ToolVersion}, {l.text("Environment / execution scope", "環境・実行範囲"), env}}}
	if previous != nil {
		meta.Fields = append(meta.Fields, field{l.text("Previous run", "前回の実行 ID"), previous.SnapshotID})
	} else {
		meta.Fields = append(meta.Fields, field{l.text("Comparison", "前回との比較"), l.text("Unavailable; no previous snapshot supplied", "比較できません。前回のスナップショットが指定されていません")})
	}
	d.Blocks = append(d.Blocks, meta)
	scope := block{ID: "scope", Title: l.text("Collection scope and limitations", "収集範囲と制限")}
	for _, c := range current.Coverage {
		scope.Fields = append(scope.Fields, field{l.text("Collector / status / scope", "収集機能・状態・範囲"), string(c.Collector) + " / " + l.state(string(c.Status)) + " / " + c.Scope}, field{l.text("Collector version", "収集機能のバージョン"), c.CollectorVersion}, field{l.text("Attempted sources", "収集を試みた情報源"), strings.Join(c.AttemptedSources, ", ")}, field{l.text("Diagnostics", "診断コード"), strings.Join(c.DiagnosticCodes, ", ")}, field{l.text("Limitations (source text)", "制限事項（情報源の原文）"), strings.Join(c.Limitations, "; ")})
	}
	d.Blocks = append(d.Blocks, scope)
	d.Blocks = append(d.Blocks, accountBlocks(current, language)...)
	d.Blocks = append(d.Blocks, block{ID: "storage-status", Title: l.text("External storage collection", "外付けストレージの収集"), Note: StorageStatus(current, language)})
	sources := []*i.Snapshot{current}
	if previous != nil {
		sources = append(sources, previous)
	}
	evidenceIDs := map[string]string{}
	for si, source := range sources {
		for ei, e := range source.Evidence {
			evidenceIDs[fmt.Sprintf("%d/%s", si, e.EvidenceID)] = fmt.Sprintf("evidence-%d-%d", si, ei)
		}
	}
	services := map[*i.Snapshot]map[string]ServiceView{}
	for _, source := range sources {
		services[source] = EndpointServices(source, language)
	}
	byID := map[string]i.Assessment{}
	if annotations != nil {
		for _, a := range annotations.Entries {
			byID[a.AssessmentID] = a
		}
	}
	accountReferences := assessment.AccountReferences(current)
	usbReferences := i.USBStorageOwners(current)
	for _, category := range []string{"current", "historical", "reference", "account-reference", "usb-storage-reference"} {
		presence := i.Current
		if category == "historical" {
			presence = i.Historical
		}
		group := block{ID: "current", Title: l.text("Current items", "今回の確認項目")}
		sourceIndex := 0
		if presence == i.Historical {
			group = block{ID: "historical", Title: l.text("Historical items", "過去の項目"), Note: l.text("Prior observations only; excluded from current counts.", "過去の観測です。今回の件数には含めません。")}
			sourceIndex = 1
		}
		if category == "reference" {
			group = block{ID: "local-reference", Title: l.text("Local-only reference services", "ローカル限定の参考情報"), Note: l.text("Outside the main review scope, not automatically accepted. Explain paths and safeguards at the entrance; collected translation candidates do not verify effective paths.", "通常は個別の判断を求めず、未確認件数にも含めません。安全と自動判定したものではありません。転送先として利用する場合は入口側で経路と保護策を説明してください。ツールは転送設定を検証していません。")}
		}
		if category == "account-reference" {
			group = block{ID: "account-reference", Title: l.text("System accounts (reference)", "システムアカウント（参考情報）"), Note: l.text("UID below 1000 and not identified as the process user of an in-scope listening service. No individual judgment is requested; excluded from current counts. UID 1000+ is only a user-account candidate heuristic, not proof of SSH login access.", "UID 1000未満で、主な待受サービスの実行ユーザとして対応付いていないアカウントです。個別判断を求めず、今回の集計に含めません。UID 1000以上もユーザ候補を選ぶ目安で、SSHログイン可否の判定ではありません。")}
		}
		if category == "usb-storage-reference" {
			group = block{ID: "usb-storage-reference", Title: l.text("Storage grouped with USB HID devices (reference)", "USBのHID機器にまとめたストレージ（参考情報）"), Note: l.text("Judge the whole composite device in its USB item. Stored storage judgments remain context only.", "複合機器全体の用途・保護策はUSB機器の項目で判断します。ストレージの以前の判断は文脈として保持します。")}
		}
		d.Blocks = append(d.Blocks, group)
		if presence == i.Historical && previous == nil {
			continue
		}
		findings := map[string]i.Finding{}
		for _, f := range sources[sourceIndex].Findings {
			findings[f.FindingID] = f
		}
		for n, item := range r.Items {
			if item.Presence != presence {
				continue
			}
			f := findings[item.FindingID]
			if presence == i.Current {
				if assessment.IsReference(f) != (category == "reference") || accountReferences[f.FindingID] != (category == "account-reference") || (usbReferences[f.FindingID] != "") != (category == "usb-storage-reference") {
					continue
				}
			}
			isAccountReference := presence == i.Current && accountReferences[f.FindingID]
			isReference := assessment.IsReference(f) || isAccountReference || (presence == i.Current && usbReferences[f.FindingID] != "")
			g := guidance.ForLanguage(f.Kind, language)
			title := fmt.Sprintf(l.text("Item %d", "項目 %d"), n+1)
			if id, ok := f.Identity.(i.EndpointIdentity); ok {
				title += " · " + strings.ToUpper(id.Transport) + " " + id.LocalAddress + fmt.Sprintf(":%d", id.LocalPort)
			}
			if account, ok := f.Identity.(i.AccountIdentity); ok {
				title += l.text(" · Local account ", " · ローカルアカウント ") + account.Name
			}
			if facts, ok := f.Facts.(i.USBFacts); ok {
				title += l.text(" · USB HID ", " · USB HID機器 ") + facts.Product
			}
			if facts, ok := f.Facts.(i.StorageFacts); ok {
				model := facts.Model
				if model == "" {
					model = l.text("Model unknown", "製品名不明")
				}
				title += l.text(" · External storage", " · 外付けストレージ") + " · " + model
			}
			if f.Kind == i.Forwarding {
				title += l.text(" · Forwarding configuration", " · 転送設定")
			}
			b := block{FindingID: f.FindingID, SourceIndex: sourceIndex, ID: fmt.Sprintf("item-%d", n), Title: title, Fields: []field{{l.text("Finding ID", "観測項目 ID"), item.FindingID}, {l.text("Current decision", "今回の判断"), l.state(string(item.CurrentDecision))}, {l.text("Change", "前回からの変化"), l.state(string(item.Change))}, {l.text("Applicability", "判断の適用状態"), l.state(string(item.Applicability))}, {l.text("Review reason", "再確認の理由"), l.reasons(item.ReasonCodes)}, {l.text("Observed fact", "観測した事実"), g.Observed}, {l.text("Why review", "確認する理由"), g.Why}}, Questions: g.Questions}
			if presence == i.Current && !isReference && (item.CurrentDecision != i.Accepted || item.Applicability == i.NeedsReview || item.Applicability == i.Conflict) {
				d.Pages.FollowUp = append(d.Pages.FollowUp, link{b.ID, title + " · " + l.state(string(item.CurrentDecision)) + " / " + l.state(string(item.Applicability))})
			}
			if facts, ok := f.Facts.(i.ForwardingFacts); ok {
				b.Fields = append(b.Fields, field{l.text("Observed forwarding rule", "観測した転送設定"), facts.Rule})
			}
			if assessment.IsReference(f) {
				b.Fields = []field{{l.text("Finding ID", "観測項目 ID"), f.FindingID}, {l.text("Observed fact", "観測した事実"), l.text("Loopback-only endpoint; retained as reference, not a verified vulnerability or safety judgment.", "ループバック限定の接続口です。参考情報として保持し、脆弱性や安全性を判定したものではありません。")}, {l.text("Change", "前回からの変化"), l.state(string(item.Change))}}
				b.Questions = nil
			}
			if isAccountReference {
				b.Note = l.text("Reference only; not an individual judgment task.", "参考情報です。個別の判断・対応は求めません。")
				b.Fields = []field{{l.text("Finding ID", "観測項目 ID"), f.FindingID}, {l.text("Change", "前回からの変化"), l.state(string(item.Change))}}
				b.Questions = nil
			}
			if view, ok := services[sources[sourceIndex]][f.FindingID]; ok {
				b.Fields = append(b.Fields, field{l.text("Listening service inventory", "待受サービスの棚卸し"), view.Note})
				for _, value := range view.Fields {
					b.Fields = append(b.Fields, field(value))
				}
			}
			if owner := usbReferences[f.FindingID]; presence == i.Current && owner != "" {
				b.Questions = nil
				b.Note = l.text("Review with the composite USB device; no separate storage judgment requested.", "複合USB機器の項目でまとめて判断します。ストレージ単独の判断は求めません。")
				for k, v := range r.Items {
					if v.FindingID == owner {
						b.Links = append(b.Links, link{fmt.Sprintf("item-%d", k), l.text("Review USB device", "USB機器の判断へ")})
					}
				}
			}
			for _, value := range append(StorageFields(f, language), USBFields(f, language)...) {
				b.Fields = append(b.Fields, field(value))
			}
			for _, value := range PrivilegeFields(f, language) {
				b.Fields = append(b.Fields, field(value))
			}
			if !isReference && len(item.AssessmentIDs) == 0 {
				b.Note = l.text("Next: explain purpose, audience or privilege users, safeguards, rationale and supporting evidence.", "次に、目的、利用者・公開範囲、保護策、判断理由、裏付ける証拠を記入してください。")
			}
			for _, id := range item.AssessmentIDs {
				if isAccountReference {
					continue
				}
				context := l.text("Prior context only; does not establish current acceptance", "過去の説明です。今回「追加対応不要」と判断したことを意味しません")
				if isReference {
					context = l.text("Stored judgment outside the main review scope", "集計外の参考情報に対する保存済みの判断")
				}
				if !isReference && presence == i.Current && item.Applicability == i.Applicable {
					context = l.text("Applicable human assessment; not a safety certification", "適用可能な人の判断です。安全性の認証ではありません")
				}
				for _, code := range item.ReasonCodes {
					if code == reportfile.SharedJudgment {
						context = l.text("Shared human service policy; not a fresh host inspection judgment", "同じ構成に対する共通設定の判断です。このPCで新たに判断したことを意味しません")
					}
				}
				l.annotation(&b, byID[id], context)
			}
			values, e := technical(struct {
				Identity i.Identity
				Facts    i.Facts
			}{f.Identity, f.Facts})
			if e != nil {
				return d, e
			}
			b.Fields = append(b.Fields, field{l.text("Observed values (bounded)", "観測値（表示量に上限あり）"), values})
			for _, id := range f.EvidenceIDs {
				b.Links = append(b.Links, link{evidenceIDs[fmt.Sprintf("%d/%s", sourceIndex, id)], l.text("Underlying evidence", "根拠となる証拠")})
			}
			b.Links = append(b.Links, link{"scope", l.text("Collection limitations", "収集上の制限")})
			d.Blocks = append(d.Blocks, b)
		}
	}
	unmatched := block{ID: "unmatched", Title: l.text("Unmatched assessments", "対応する観測がない判断"), Note: l.text("Historical context only; excluded from current decisions.", "過去の文脈として保持します。今回の判断には含めません。")}
	for _, id := range r.UnmatchedAssessmentIDs {
		l.annotation(&unmatched, byID[id], unmatched.Note)
	}
	d.Blocks = append(d.Blocks, unmatched)
	d.Blocks = append(d.Blocks, block{ID: "evidence", Title: l.text("Evidence", "技術的な証拠"), Note: l.text("Technical values are limited to 2000 Unicode characters per value. Truncated values require the original artifact. Source text and human statements are preserved, not translated. External references are never fetched.", "技術値の表示は各値 2000 文字までです。省略された部分は元のファイルで確認してください。情報源の原文と人の記述は翻訳せず保持します。外部参照にはアクセスしません。")})
	for si, source := range sources {
		for ei, e := range source.Evidence {
			value, err := technical(e.Values)
			if err != nil {
				return d, err
			}
			b := block{Technical: true, ID: fmt.Sprintf("evidence-%d-%d", si, ei), Title: fmt.Sprintf(l.text("Evidence %d / %d", "証拠 %d / %d"), si, ei), Fields: []field{{l.text("Source run", "収集元の実行 ID"), source.SnapshotID}, {l.text("Evidence ID", "証拠 ID"), e.EvidenceID}, {l.text("Source / location", "情報源・位置"), e.SourceKind + " / " + e.Location}, {l.text("Method / time (source text)", "取得方法・時刻（原文）"), e.Method + " / " + e.CollectedAt}, {l.text("Technical values", "技術値"), value}}}
			for _, c := range source.Coverage {
				b.Fields = append(b.Fields, field{l.text("Source coverage / limitations", "収集元の状態・制限"), string(c.Collector) + " / " + l.state(string(c.Status)) + " / " + strings.Join(c.Limitations, "; ")})
			}
			d.Blocks = append(d.Blocks, b)
		}
	}
	anchors := map[string]block{}
	for _, b := range d.Blocks {
		if b.FindingID != "" {
			anchors[b.FindingID] = b
		}
	}
	for sourceIndex, source := range sources {
		connections := PrivilegeConnections(source)
		for n := range d.Blocks {
			if d.Blocks[n].SourceIndex != sourceIndex {
				continue
			}
			if len(connections[d.Blocks[n].FindingID]) >= privilegeLinkLimit {
				d.Blocks[n].Fields = append(d.Blocks[n].Fields, field{l.text("Related navigation limit", "関連項目の表示上限"), l.text("Up to 64 related links per item; consult the full item inventory for other settings.", "関連リンクは各項目最大64件です。その他の設定は項目一覧で確認できます。")})
			}
			for _, id := range connections[d.Blocks[n].FindingID] {
				if target, ok := anchors[id]; ok && target.SourceIndex == sourceIndex {
					d.Blocks[n].Links = append(d.Blocks[n].Links, link{target.ID, l.text("Related account / service / sudo setting: ", "関連するアカウント・サービス・sudo設定：") + target.Title})
				}
			}
		}
	}
	m := NetworkMapFor(&reportfile.File{Current: current, Assessments: annotations}, r, language)
	attachMapAnchors(&m, d.Blocks)
	d.Network = &m
	physical := PhysicalMapFor(&reportfile.File{Current: current, Assessments: annotations}, r, language)
	attachPhysicalAnchors(&physical, d.Blocks)
	d.Physical = &physical
	return d, nil
}

// RenderWithOptions renders a self-contained report, preserving the legacy English Markdown default.
func RenderWithOptions(current, previous *i.Snapshot, annotations *i.Assessments, o Options) (string, error) {
	if err := ValidateOptions(o); err != nil {
		return "", err
	}
	o = o.normalized()
	if o.Format == "markdown" && o.Language == "en" {
		return Render(current, previous, annotations)
	}
	d, err := makeDocument(current, previous, annotations, o.Language)
	if err != nil {
		return "", err
	}
	return renderDocument(d, o)
}
func renderDocument(d document, o Options) (string, error) {
	var b bytes.Buffer
	if o.Format == "html" {
		t, err := template.New("report").Funcs(template.FuncMap{"escape": func(value any) string { return html.EscapeString(visible(fmt.Sprint(value))) }}).Parse(htmlReport)
		if err != nil {
			return "", err
		}
		if err = t.Execute(&b, d); err != nil {
			return "", err
		}
		return b.String(), nil
	}
	fmt.Fprintf(&b, "# %s\n\n%s\n\n## %s\n\n", d.Title, d.Warning+"\n\n"+d.Boundary, d.Summary)
	for _, m := range d.Metrics {
		fmt.Fprintf(&b, "- %s: %d\n", m.Label, m.Count)
	}
	for _, section := range d.Blocks {
		fmt.Fprintf(&b, "\n<a id=\"%s\"></a>\n\n## %s\n\n", section.ID, Text(section.Title))
		if section.Note != "" {
			fmt.Fprintf(&b, "%s\n\n", section.Note)
		}
		for _, f := range section.Fields {
			fmt.Fprintf(&b, "- %s: %s\n", Text(f.Label), Text(f.Value))
		}
		for _, q := range section.Questions {
			fmt.Fprintf(&b, "- %s\n", q)
		}
		for _, a := range section.Links {
			fmt.Fprintf(&b, "- [%s](#%s)\n", Text(a.Label), a.ID)
		}
	}
	return b.String(), nil
}

// visible makes control and formatting characters explicit before HTML escaping.
func visible(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) || unicode.Is(unicode.Zl, r) || unicode.Is(unicode.Zp, r) {
			fmt.Fprintf(&b, "\\u%04X", r)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}
