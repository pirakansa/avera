package report

import (
	"fmt"

	"github.com/pirakansa/avera/internal/reportfile"
)

// RenderFile includes human-described paths without converting them into observed facts.
func RenderFile(f *reportfile.File, o Options) (string, error) {
	if err := f.Validate(); err != nil {
		return "", err
	}
	if err := ValidateOptions(o); err != nil {
		return "", err
	}
	o = o.normalized()
	if o.Format != "html" && len(f.Routes) == 0 && len(f.Policies) == 0 && !hasDetectedTargets(f) {
		return RenderWithOptions(f.Current, f.Previous, f.Assessments, o)
	}
	result, err := f.Review()
	if err != nil {
		return "", err
	}
	d, err := documentFromReview(f.Current, f.Previous, f.PolicyJudgments(), o.Language, result)
	if err != nil {
		return "", err
	}
	l := localizer{o.Language == "ja"}
	blocks := map[string]int{}
	for n, b := range d.Blocks {
		if b.FindingID != "" {
			blocks[b.FindingID] = n
		}
	}
	routes := block{ID: "routes", Title: l.text("Human-described forwarding paths", "人が記入した転送・関連付け"), Note: l.text("Relationships are human statements; collected firewall rules are separate observations. Effective paths, reverse proxies and reachability are not verified.", "関連付けは人による申告です。収集した転送ルールとは区別します。実際の経路・プロキシ設定・到達性を検証したものではありません。")}
	for _, route := range f.Routes {
		status := l.text("Prior path context; confirm changed or missing observations", "過去の経路説明です。観測の変化や不足があるため再確認してください")
		if f.RouteCurrent(route) {
			status = l.text("Saved observation bases match; forwarding remains unverified", "保存時の観測内容と一致しています。転送経路は未検証です")
		}
		routes.Fields = append(routes.Fields, field{l.text("Path statement / status", "経路の説明・状態"), route.Explanation + " / " + status}, field{l.text("Actor / date", "記入者・日時"), route.Actor + " / " + route.RecordedAt}, field{l.text("Entrance / internal finding IDs", "入口・内部サービスの観測 ID"), route.EntranceID + " → " + route.InternalID})
		for _, id := range []string{route.EntranceID, route.InternalID} {
			if n, ok := blocks[id]; ok {
				routes.Links = append(routes.Links, link{d.Blocks[n].ID, d.Blocks[n].Title})
			}
		}
		if n, ok := blocks[route.EntranceID]; ok {
			b := &d.Blocks[n]
			b.Fields = append(b.Fields, field{l.text("Human-described path (not verified)", "転送・関連付けの申告（未検証）"), route.Explanation + " / " + status})
			if target, ok := blocks[route.InternalID]; ok {
				b.Links = append(b.Links, link{d.Blocks[target].ID, l.text("Related local service", "関連するローカル限定サービス")})
			}
		}
	}
	if len(f.Routes) > 0 {
		d.Blocks = append(d.Blocks, routes)
	}

	for _, finding := range f.Current.Findings {
		targets := f.DetectedTargets(finding)
		if len(targets) == 0 {
			continue
		}
		n, ok := blocks[finding.FindingID]
		if !ok {
			continue
		}
		b := &d.Blocks[n]
		b.Fields = append(b.Fields, field{l.text("Configured destination match", "設定に記載された転送先との一致"), l.text("Literal DNAT destination matched to observed local services; effective reachability is not verified.", "DNAT のアドレス・ポートと観測サービスを照合しました。実際の到達性は検証していません。")})
		for _, target := range targets {
			if n, ok := blocks[target.FindingID]; ok {
				b.Links = append(b.Links, link{d.Blocks[n].ID, d.Blocks[n].Title})
			}
		}
	}
	m := NetworkMapFor(f, result.Review, o.Language)
	attachMapAnchors(&m, d.Blocks)
	d.Network = &m
	physical := PhysicalMapFor(f, result.Review, o.Language)
	attachPhysicalAnchors(&physical, d.Blocks)
	d.Physical = &physical
	text, err := renderDocument(d, o)
	if err != nil {
		return "", fmt.Errorf("render portable report: %w", err)
	}
	return text, nil
}

func hasDetectedTargets(f *reportfile.File) bool {
	for _, v := range f.Current.Findings {
		if len(f.DetectedTargets(v)) > 0 {
			return true
		}
	}
	return false
}
