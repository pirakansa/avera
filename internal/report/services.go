package report

import (
	"fmt"
	"strings"

	i "github.com/pirakansa/avera/internal/inventory"
)

type ServiceField struct{ Label, Value string }

// ServiceView separates process observations from the human purpose and judgment.
type ServiceView struct {
	Note    string
	Summary string
	Fields  []ServiceField
}

// EndpointServices projects only evidence explicitly linked to each endpoint.
// Legacy snapshots retain executable/socket UID information with explicit unknowns.
func EndpointServices(snapshot *i.Snapshot, language string) map[string]ServiceView {
	l := localizer{language == "ja"}
	evidence := map[string]i.ProcessValues{}
	for _, entry := range snapshot.Evidence {
		if values, ok := entry.Values.(i.ProcessValues); ok {
			evidence[entry.EvidenceID] = values
		}
	}
	out := map[string]ServiceView{}
	unknown := l.text("Unknown / not collected", "不明・未収集")
	for _, finding := range snapshot.Findings {
		if finding.Kind != i.Endpoint {
			continue
		}
		view := ServiceView{Note: l.text("Observed socket holders only. Socket UID and process effective UID are different observations. A cgroup unit name is a systemd service candidate, not a verified unit or activation target. Record the service purpose, necessity and responsible team in the human explanation.", "観測した待受を保持するプロセスの情報です。ソケットUIDとプロセスの実効UIDは別の情報です。cgroupの名前はsystemdサービスの候補であり、起動先やユニットの確認結果ではありません。用途・必要性・管理担当は人による説明に記録してください。"), Fields: []ServiceField{}}
		seen := map[string]bool{}
		summaries := []string{}
		for _, id := range finding.EvidenceIDs {
			values, ok := evidence[id]
			if !ok || values.PID == nil {
				continue
			}
			key := fmt.Sprintf("%d", *values.PID)
			if seen[key] {
				continue
			}
			seen[key] = true
			name, uid, unit, path, diagnostics := unknown, unknown, unknown, unknown, unknown
			if service := values.Service; service != nil {
				if service.ProcessName != "" {
					name = service.ProcessName
				}
				if service.EffectiveUID != nil {
					uid = fmt.Sprint(*service.EffectiveUID)
				}
				if service.UnitCandidate != "" {
					unit = service.UnitCandidate
				}
				if service.CgroupPath != "" {
					path = service.CgroupPath
				}
				diagnostics = strings.Join(service.Diagnostics, ", ")
			}
			executable := unknown
			sockets := []string{}
			for _, owner := range values.Observed.Owners {
				if owner.Executable != "" {
					executable = owner.Executable
				}
				sockets = append(sockets, fmt.Sprint(owner.UID))
			}
			view.Fields = append(view.Fields, ServiceField{l.text("Process name", "プロセス名"), name}, ServiceField{l.text("Executable", "実行ファイル"), executable}, ServiceField{l.text("Process effective UID", "実行ユーザ（プロセス実効UID）"), uid}, ServiceField{l.text("Socket UID", "ソケットUID"), strings.Join(sockets, ", ")}, ServiceField{l.text("systemd service candidate (cgroup)", "systemdサービス候補（cgroup）"), unit}, ServiceField{l.text("Observed cgroup path", "観測したcgroupパス"), path})
			if diagnostics != "" {
				view.Fields = append(view.Fields, ServiceField{l.text("Service attribution diagnostics", "サービス特定の診断"), diagnostics})
			}
			if values.Service != nil && (values.Service.ProcessName != "" || values.Service.UnitCandidate != "") {
				summaries = append(summaries, fmt.Sprintf("%s · %s · UID %s", name, unit, uid))
			}
		}
		if len(seen) == 0 {
			view.Fields = append(view.Fields, ServiceField{l.text("Process / service", "プロセス・サービス"), unknown})
			if facts, ok := finding.Facts.(i.NetworkFacts); ok {
				for _, owner := range facts.Owners {
					executable := owner.Executable
					if executable == "" {
						executable = unknown
					}
					view.Fields = append(view.Fields, ServiceField{l.text("Executable / socket UID", "実行ファイル・ソケットUID"), fmt.Sprintf("%s / %d", executable, owner.UID)})
				}
			}
		}
		view.Summary = strings.Join(summaries, "; ")
		out[finding.FindingID] = view
	}
	return out
}
