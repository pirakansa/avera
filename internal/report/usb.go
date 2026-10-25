package report

import (
	"fmt"
	"strings"

	i "github.com/pirakansa/avera/internal/inventory"
	"github.com/pirakansa/avera/internal/reportfile"
)

// USBFields describes declared functions, never input activity or trust.
func USBFields(finding i.Finding, language string) []ServiceField {
	f, ok := finding.Facts.(i.USBFacts)
	if !ok {
		return nil
	}
	l := localizer{language == "ja"}
	functions := []string{}
	for _, v := range f.Interfaces {
		label := "USB class " + v.Class
		if v.Class == "03" {
			label = l.text("HID (declared)", "HID（機器が申告する機能）")
			if v.Subclass == "01" && v.Protocol == "01" {
				label = l.text("Keyboard (boot HID)", "キーボード（boot HID）")
			} else if v.Subclass == "01" && v.Protocol == "02" {
				label = l.text("Mouse (boot HID)", "マウス（boot HID）")
			}
		}
		if v.Class == "08" {
			label = l.text("Storage (declared)", "ストレージ（機器が申告する機能）")
		}
		functions = append(functions, v.Number+": "+label)
	}
	return []ServiceField{{l.text("USB device", "USB機器"), f.Manufacturer + " " + f.Product}, {"VID:PID", f.VendorID + ":" + f.ProductID}, {l.text("Observed serial", "観測したシリアル番号"), f.Serial}, {l.text("Declared interfaces", "申告されたインターフェース"), strings.Join(functions, "\n")}, {l.text("Grouped storage observations", "同じ機器にまとめたストレージ観測"), fmt.Sprint(len(f.StorageFingerprints))}, {l.text("Review scope", "確認する範囲"), l.text("Record intended use, responsible team, permitted handling and physical/manual safeguards. These are human statements, not verified port restrictions.", "用途・想定・管理担当と、取り扱いルールや運用マニュアルなどの保護策を記録します。人の申告であり、USBポートの接続制限や実施を検証したものではありません。")}}
}

type PhysicalNode struct {
	Card     NetworkCard
	Children []*PhysicalNode
}
type PhysicalMap struct {
	Title, Note, Outside, Host, Status, LimitNote string
	Nodes                                         []*PhysicalNode
}

// PhysicalMapFor renders only observed attachment metadata, bounded to 64 nodes
// and eight levels. It does not infer unused ports, physical location or policy.
func PhysicalMapFor(f *reportfile.File, review *i.Review, language string) PhysicalMap {
	l := localizer{language == "ja"}
	m := PhysicalMap{Title: l.text("Physical configuration overview", "物理構成の概要"), Host: f.Current.TargetID, Note: l.text("Current USB attachment tree. HID devices require a purpose and a human decision; hubs and other functions provide context.", "観測時点のUSB接続関係です。HID機器の用途と判断を確認し、ハブやその他の機能は参考情報として表示します。"), Outside: l.text("Operating boundary (human statement): manuals, permitted handlers and physical management may provide safeguards, like an upstream network control. The tool does not verify them.", "運用上の保護（人の申告）：上位の運用マニュアル・接続を扱う担当者・物理管理などを保護策として記録できます。ツールはその実施を検証しません。"), Nodes: []*PhysicalNode{}}
	m.Status = l.text("USB was not collected in this file. Recollect to supply topology; absence is not established.", "このファイルではUSB接続は未収集です。再収集で取得できます。未収集は機器がないことを示しません。")
	for _, c := range f.Current.Coverage {
		if c.Collector == i.USB {
			m.Status = fmt.Sprintf(l.text("USB collection: %s. Unoccupied ports and future attachment permissions are outside scope.", "USBの収集状況：%s。空きポートや今後接続してよい機器の制限は対象外です。"), l.state(string(c.Status)))
		}
	}
	nodes := i.USBNodes(f.Current)
	items := map[string]i.ReviewItem{}
	for _, v := range review.Items {
		if v.Presence == i.Current {
			items[v.FindingID] = v
		}
	}
	judgments := map[string]i.Assessment{}
	if f.Assessments != nil {
		for _, a := range f.Assessments.Entries {
			judgments[a.AssessmentID] = a
		}
	}
	findings := map[string]i.Finding{}
	for _, v := range f.Current.Findings {
		if id, ok := v.Identity.(i.USBIdentity); ok {
			findings[id.Key] = v
		}
	}
	byKey := map[string]*PhysicalNode{}
	parents := map[string]string{}
	for n, v := range nodes {
		if n >= 64 {
			m.LimitNote = l.text("Diagram limited to 64 nodes. Consult the source JSON for all collected metadata.", "図は64機器まで表示します。収集した全情報はJSONで確認してください。")
			break
		}
		card := NetworkCard{Title: mapText(v.Facts.Product), Detail: v.PortPath + " · " + v.Facts.VendorID + ":" + v.Facts.ProductID, Decision: l.text("Reference topology; no individual judgment", "接続の参考情報・個別判断不要")}
		if card.Title == "" {
			card.Title = v.Facts.VendorID + ":" + v.Facts.ProductID
		}
		if v.Role == "root_hub" {
			card.Title = l.text("USB root hub", "USBルートハブ") + " · " + v.PortPath
		} else if v.Role == "hub" {
			card.Title = l.text("USB hub", "USBハブ") + " · " + card.Title
		}
		if finding, ok := findings[v.Key]; ok {
			card.FindingID = finding.FindingID
			card.Anchor = "finding-" + finding.FindingID
			r := items[finding.FindingID]
			card.Decision = l.state(string(r.CurrentDecision)) + " / " + l.state(string(r.Applicability))
			card.Tone = "unreviewed"
			if r.CurrentDecision == i.Accepted {
				card.Tone = "accepted"
			} else if r.CurrentDecision == i.ActionRequired {
				card.Tone = "action"
			}
			if len(r.AssessmentIDs) == 1 {
				a := judgments[r.AssessmentIDs[0]]
				if a.Purpose != "" {
					card.Purpose = l.text("Purpose (human): ", "用途（人の申告）：") + mapText(a.Purpose)
				}
				if a.Safeguards != "" {
					card.Safeguards = l.text("Safeguards (human): ", "保護策（人の申告）：") + mapText(a.Safeguards)
				}
			}
		}
		byKey[v.Key] = &PhysicalNode{Card: card, Children: []*PhysicalNode{}}
		parents[v.Key] = v.ParentKey
	}
	// Preserve metadata order. Flatten deep or omitted parents with an explicit note.
	for _, v := range nodes {
		node := byKey[v.Key]
		if node == nil {
			continue
		}
		depth := 0
		for key := parents[v.Key]; key != "" && depth < 9; key = parents[key] {
			depth++
		}
		parent := byKey[v.ParentKey]
		if parent != nil && depth < 8 {
			parent.Children = append(parent.Children, node)
		} else {
			if v.Role != "root_hub" {
				node.Card.Detail += " · " + l.text("Parent attachment unavailable or depth limited", "親の接続は不明・表示範囲外")
			}
			m.Nodes = append(m.Nodes, node)
		}
	}
	displayed := len(byKey)
	// Old storage artifacts lack reliable USB parent metadata; retain their context
	// without inventing an edge or a duplicate USB judgment.
	grouped := i.USBStorageOwners(f.Current)
	storageNodes := i.USBStorageNodes(f.Current)
	for _, v := range f.Current.Findings {
		if v.Kind != i.ExternalStorage || grouped[v.FindingID] != "" {
			continue
		}
		if displayed >= 64 {
			m.LimitNote = l.text("Diagram limited to 64 nodes; consult source JSON.", "図は64機器まで表示します。全情報はJSONで確認してください。")
			break
		}
		displayed++
		facts := v.Facts.(i.StorageFacts)
		r := items[v.FindingID]
		storageNode := &PhysicalNode{Card: NetworkCard{FindingID: v.FindingID, Anchor: "finding-" + v.FindingID, Title: mapText(facts.Model), Detail: l.text("Storage: USB parent association not available in the report", "ストレージ：レポート内ではUSB親接続との対応は未特定"), Decision: l.state(string(r.CurrentDecision))}, Children: []*PhysicalNode{}}
		if parent := byKey[storageNodes[v.FindingID]]; parent != nil {
			storageNode.Card.Detail = l.text("Observed storage function", "観測したストレージ機能")
			parent.Children = append(parent.Children, storageNode)
		} else {
			m.Nodes = append(m.Nodes, storageNode)
		}
		if len(m.Nodes) >= 64 {
			break
		}
	}
	return m
}
func attachPhysicalAnchors(m *PhysicalMap, blocks []block) {
	anchors := map[string]string{}
	for _, b := range blocks {
		if b.FindingID != "" {
			anchors[b.FindingID] = b.ID
		}
	}
	var visit func([]*PhysicalNode)
	visit = func(nodes []*PhysicalNode) {
		for _, n := range nodes {
			n.Card.Anchor = anchors[n.Card.FindingID]
			visit(n.Children)
		}
	}
	visit(m.Nodes)
}

const PhysicalMapStyles = `.physical-map .physical-tree{list-style:none;background:transparent;padding:.2rem 0 .2rem 1rem;margin:.4rem 0;border-left:2px solid #9ab9ae}.physical-map .physical-tree>li{padding:.3rem 0 .3rem .7rem;margin:0}.physical-map .map-card{max-width:42rem}.physical-map .map-host{overflow-wrap:anywhere}@media(max-width:650px){.physical-map .physical-tree{padding-left:.3rem}.physical-map .physical-tree>li{padding-left:.3rem}}@media print{.physical-map .physical-tree{break-inside:auto}.physical-map .map-card{break-inside:avoid}}`
