package report

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/pirakansa/avera/internal/assessment"
	i "github.com/pirakansa/avera/internal/inventory"
	"github.com/pirakansa/avera/internal/reportfile"
)

const networkMapLimit = 64

// NetworkMap is a presentation projection, never a reachability or topology claim.
type NetworkMap struct {
	Title, Note, Outside, Host, Legend, ReferencesLabel, Empty, LimitNote string
	Coverage                                                              []string
	Rows                                                                  []NetworkRow
	References                                                            []NetworkCard
}
type NetworkRow struct {
	Source         NetworkCard
	Targets        []NetworkCard
	Relation, Mode string
}
type NetworkCard struct {
	FindingID, Anchor, Title, Detail, Decision, Tone, Purpose, Safeguards string
}

// NetworkMapFor uses only current observations, current derived judgments and
// active human relationships. It never inspects a host or reads archived history.
func NetworkMapFor(f *reportfile.File, review *i.Review, language string) NetworkMap {
	l := localizer{language == "ja"}
	m := NetworkMap{Title: l.text("Network configuration overview", "ネットワーク構成の概要"), Host: f.Current.TargetID,
		Note:            l.text("This diagram shows settings observed within the PC. Upstream access controls can also support the judgment rationale. External reachability is not verified. Open an item for full evidence and judgment; long labels are abbreviated here.", "この図はPC内で観測した設定の関係を示します。上位のアクセス制限も判断の根拠として記載できます。外部からの到達性を検証した図ではありません。項目を開くと証拠と判断の詳細を確認できます。長い説明は図では省略します。"),
		Outside:         l.text("Include safeguards provided by upstream routers, load balancers and cloud services in the judgment rationale.", "上位ルーター・ロードバランサー・クラウド側の保護策は、判断理由に補足してください"),
		Legend:          l.text("Solid arrow: literal configured destination match. Dashed arrow: human-described relationship. No arrow: no relationship determined; not proof of no forwarding.", "実線の矢印：設定に記載された転送先との一致 ／ 破線の矢印：人が記入した関連付け ／ 矢印なし：関連先を特定していない状態（転送がないことの証明ではありません）"),
		ReferencesLabel: l.text("Observed loopback services (reference)", "観測したループバックサービス（参考情報）"), Empty: l.text("No current network observations; inspect collection coverage. This does not prove absence.", "今回のネットワーク観測はありません。収集状況を確認してください。不在を証明するものではありません。"),
		Rows: []NetworkRow{}, References: []NetworkCard{}, Coverage: []string{},
	}
	byID := map[string]i.ReviewItem{}
	for _, item := range review.Items {
		if item.Presence == i.Current {
			byID[item.FindingID] = item
		}
	}
	judgments := map[string]i.Assessment{}
	if f.Assessments != nil {
		for _, a := range f.PolicyJudgments().Entries {
			judgments[a.AssessmentID] = a
		}
	}
	cards := map[string]NetworkCard{}
	for _, finding := range f.Current.Findings {
		if finding.Collector != i.Network {
			continue
		}
		card := NetworkCard{FindingID: finding.FindingID}
		switch id := finding.Identity.(type) {
		case i.EndpointIdentity:
			card.Title = strings.ToUpper(id.Transport) + " " + id.LocalAddress + fmt.Sprintf(":%d", id.LocalPort)
			if facts, ok := finding.Facts.(i.NetworkFacts); ok {
				owners := []string{}
				for _, owner := range facts.Owners {
					owners = append(owners, owner.Executable)
				}
				card.Detail = mapText(strings.Join(owners, ", "))
			}
		case i.SourceIdentity:
			if finding.Kind != i.Forwarding {
				continue
			}
			card.Title = fmt.Sprintf(l.text("Forwarding · %s #%d", "転送設定 · %s #%d"), filepath.Base(id.SourcePath), id.Line)
			if facts, ok := finding.Facts.(i.ForwardingFacts); ok {
				card.Detail = mapText(facts.Rule)
			}
		default:
			continue
		}
		item := byID[finding.FindingID]
		if !assessment.IsReference(finding) {
			card.Decision = l.state(string(item.CurrentDecision))
			if item.CurrentDecision == i.ActionRequired {
				card.Tone = "action"
			} else if item.CurrentDecision == i.Unreviewed {
				card.Tone = "unreviewed"
			} else {
				card.Tone = "accepted"
			}
			if len(item.AssessmentIDs) == 1 {
				a := judgments[item.AssessmentIDs[0]]
				if a.Purpose != "" {
					card.Purpose = l.text("Purpose: ", "目的：") + mapText(a.Purpose)
				}
				if a.Safeguards != "" {
					card.Safeguards = l.text("Safeguards (human statement): ", "保護策（人の申告）：") + mapText(a.Safeguards)
				}
			}
		} else {
			card.Decision = l.text("Reference, not a safety judgment", "参考情報・安全性の判断ではありません")
		}
		cards[finding.FindingID] = card
	}
	for _, finding := range f.Current.Findings {
		card, ok := cards[finding.FindingID]
		if !ok {
			continue
		}
		if assessment.IsReference(finding) {
			m.References = append(m.References, card)
			continue
		}
		row := NetworkRow{Source: card, Targets: []NetworkCard{}, Mode: "unknown", Relation: l.text("Forwarding/relay relationship not determined", "転送・中継先との対応は未特定")}
		for _, target := range f.DetectedTargets(finding) {
			if c, ok := cards[target.FindingID]; ok {
				row.Targets = append(row.Targets, c)
			}
		}
		if len(row.Targets) > 0 {
			row.Mode = "observed"
			row.Relation = l.text("Configured destination match (reachability unverified)", "設定の転送先と一致（到達性は未検証）")
		}
		// Human relationships are separate rows so they never look like observed rules.
		human := NetworkRow{Source: card, Targets: []NetworkCard{}, Mode: "human", Relation: l.text("Human-described path (unverified)", "人が記入した関連付け（未検証）")}

		for _, route := range f.Routes {
			if route.EntranceID != finding.FindingID {
				continue
			}
			target, ok := cards[route.InternalID]
			if !ok {
				target = NetworkCard{Title: l.text("Related service not observed this run", "関連サービスは今回は未観測"), Detail: route.InternalID, Decision: l.text("Recheck the relationship", "関連付けを再確認してください"), Tone: "unreviewed"}
			}
			human.Targets = append(human.Targets, target)
			if !f.RouteCurrent(route) {
				human.Relation = l.text("Human path: changed or missing observations; recheck", "人の関連付け：観測に変化・未観測あり・再確認")
			}
		}

		if len(row.Targets) > 0 || len(human.Targets) == 0 {
			m.Rows = append(m.Rows, row)
		}
		if len(human.Targets) > 0 {
			m.Rows = append(m.Rows, human)
		}
	}
	slices.SortStableFunc(m.Rows, func(a, b NetworkRow) int {
		if c := strings.Compare(a.Source.Title, b.Source.Title); c != 0 {
			return c
		}
		return strings.Compare(a.Mode, b.Mode)
	})
	slices.SortFunc(m.References, func(a, b NetworkCard) int { return strings.Compare(a.Title, b.Title) })
	if len(m.Rows) == 0 && len(m.References) > 0 {
		m.Empty = l.text("Only loopback services were observed; other entrances may be outside collection coverage.", "今回の観測はループバックサービスのみです。収集範囲外に入口がないことを示すものではありません。")
	}
	omitted := 0
	if len(m.Rows) > networkMapLimit {
		omitted += len(m.Rows) - networkMapLimit
		m.Rows = m.Rows[:networkMapLimit]
	}
	if len(m.References) > networkMapLimit {
		omitted += len(m.References) - networkMapLimit
		m.References = m.References[:networkMapLimit]
	}
	for n := range m.Rows {
		if len(m.Rows[n].Targets) > networkMapLimit {
			omitted += len(m.Rows[n].Targets) - networkMapLimit
			m.Rows[n].Targets = m.Rows[n].Targets[:networkMapLimit]
		}
	}
	if omitted > 0 {
		m.LimitNote = fmt.Sprintf(l.text("%d additional rows/services omitted from the diagram; see the full inventory.", "図では残り %d 件の行・サービスを省略しています。全件は確認項目で参照してください。"), omitted)
	}
	for _, coverage := range f.Current.Coverage {
		if coverage.Collector == i.Network {
			m.Coverage = append(m.Coverage, l.text("Collection status: ", "収集状況：")+l.state(string(coverage.Status)))
			m.Coverage = append(m.Coverage, coverage.DiagnosticCodes...)
		}
	}
	if len(m.Coverage) == 0 {
		m.Coverage = append(m.Coverage, l.text("Network coverage unavailable", "ネットワークの収集範囲が不明です"))
	}
	return m
}
func mapText(value string) string {
	runes := []rune(value)
	const maxMapLabel = 200
	if len(runes) > maxMapLabel {
		return string(runes[:maxMapLabel]) + "…"
	}
	return value
}
func attachMapAnchors(m *NetworkMap, blocks []block) {
	anchors := map[string]string{}
	for _, b := range blocks {
		if b.FindingID != "" {
			anchors[b.FindingID] = b.ID
		}
	}
	for n := range m.Rows {
		m.Rows[n].Source.Anchor = anchors[m.Rows[n].Source.FindingID]
		for j := range m.Rows[n].Targets {
			m.Rows[n].Targets[j].Anchor = anchors[m.Rows[n].Targets[j].FindingID]
		}
	}
	for n := range m.References {
		m.References[n].Anchor = anchors[m.References[n].FindingID]
	}
}
