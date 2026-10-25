package report

import (
	"fmt"
	"slices"
	"strings"

	i "github.com/pirakansa/avera/internal/inventory"
)

// AccountBoundary is a scope statement, never an outstanding review task.
func AccountBoundary(language string) string {
	return (localizer{language == "ja"}).text("Individual account judgments cover observed in-scope service users, USB input consumers and UID 1000+ user-account candidates; lower-UID accounts without such a service match remain references. UID thresholds do not establish login access. This report also covers runtime groups, local memberships, configured sudo rules and connected USB/removable non-root storage candidates. Storage review asks for connection purpose and necessity, not content or file-permission inspection. External identities, actual SSH login access, management sockets and other non-sudo privilege paths are outside this review; no additional judgment is requested for those exclusions.", "アカウントの個別判断対象は、主な待受サービス・USB入力を開くプロセスの実行ユーザとUID 1000以上のユーザ候補です。それ以外の低UIDアカウントは参考情報とし、個別判断を求めません。UIDはログイン可否の判定ではありません。実行中グループ、設定上の所属グループ、sudo設定、rootfs用ディスクを除くUSB接続・取り外し可能なストレージ候補も確認対象です。ストレージは接続の用途・必要性を確認し、内容やファイル権限は調べません。外部管理のアカウント、SSHでの実際のログイン可否、管理用ソケットなどsudo以外の権限経路は対象外です。これらの対象外事項について追加の調査・判断は求めません。")
}

// LocalGroups returns the sanitized observed directory, including groups with no members.
func LocalGroups(snapshot *i.Snapshot) []i.LocalGroup {
	groups := []i.LocalGroup{}
	for _, entry := range snapshot.Evidence {
		if values, ok := entry.Values.(i.AccountValues); ok {
			groups = append(groups, values.Groups...)
		}
	}
	return groups
}

// PrivilegeFields formats configured accounts and sudo rules without evaluating grants.
func PrivilegeFields(finding i.Finding, language string) []ServiceField {
	l := localizer{language == "ja"}
	if rule, ok := finding.Facts.(i.SudoFacts); ok {
		commands := []string{}
		for _, command := range rule.Commands {
			arguments := command.Arguments
			if arguments == "" {
				arguments = l.text("arguments unrestricted by this command entry", "このコマンド指定に引数制限なし")
			}
			tags := strings.Join(command.Tags, ", ")
			if tags == "" {
				tags = l.text("no explicit password tag", "パスワード要求タグの明示なし")
			}
			commands = append(commands, command.Path+" · "+arguments+" · "+tags)
		}
		members := []string{}
		for _, member := range rule.LocalMembers {
			members = append(members, fmt.Sprintf("%s (UID %d)", member.Name, member.UID))
		}
		subject := rule.Principal.Name
		if rule.Principal.Type == "group" {
			subject = "%" + subject
		}
		return []ServiceField{{l.text("Configured sudo subject", "sudo設定の対象"), subject}, {l.text("Run-as users", "実行先ユーザ"), strings.Join(rule.RunAsUsers, ", ")}, {l.text("Run-as groups", "実行先グループ"), strings.Join(rule.RunAsGroups, ", ")}, {l.text("Configured commands / arguments / password tags", "設定されたコマンド・引数・パスワード要求タグ"), strings.Join(commands, "\n")}, {l.text("Locally resolved subjects", "ローカルで対応付けた対象ユーザ"), strings.Join(members, "; ")}, {l.text("Interpretation / local context complete", "設定解釈・ローカル対応付け完了"), fmt.Sprintf("%t / %t", rule.InterpretationComplete, rule.LocalContextComplete)}}
	}
	if unparsed, ok := finding.Facts.(i.UnparsedFacts); ok {
		return []ServiceField{{l.text("Unsupported configuration text", "解釈できなかった設定"), unparsed.Raw}, {l.text("Diagnostic", "診断コード"), unparsed.DiagnosticCode}}
	}
	facts, ok := finding.Facts.(i.AccountFacts)
	if !ok {
		return nil
	}
	groups := []string{}
	for _, group := range facts.Groups {
		groups = append(groups, fmt.Sprintf("%s (GID %d, %s)", group.Name, group.GID, l.text(group.Membership, map[string]string{"primary": "基本グループ", "supplementary": "補助グループ"}[group.Membership])))
	}
	return []ServiceField{{l.text("Local account", "ローカルアカウント"), finding.Identity.(i.AccountIdentity).Name}, {"UID", fmt.Sprint(facts.UID)}, {l.text("Configured primary GID", "設定上の基本GID"), fmt.Sprint(facts.PrimaryGID)}, {l.text("Configured local memberships", "設定上の所属グループ"), strings.Join(groups, "; ")}, {l.text("Home directory", "ホームディレクトリ"), facts.Home}, {l.text("Configured shell (not login permission)", "設定されたシェル（ログイン可否の判定ではありません）"), facts.Shell}}
}

// PrivilegeConnections links observed UID matches and configured sudo subjects.
// Links neither prove login access nor evaluate effective privileges.
const privilegeLinkLimit = 64

func PrivilegeConnections(snapshot *i.Snapshot) map[string][]string {
	out := map[string][]string{}
	byUID := map[uint32][]string{}
	accountsByName := map[string]string{}
	accountsByGroup := map[string][]string{}
	for _, finding := range snapshot.Findings {
		if facts, ok := finding.Facts.(i.AccountFacts); ok {
			accountsByName[finding.Identity.(i.AccountIdentity).Name] = finding.FindingID
			for _, group := range facts.Groups {
				accountsByGroup[group.Name] = append(accountsByGroup[group.Name], finding.FindingID)
			}
			byUID[facts.UID] = append(byUID[facts.UID], finding.FindingID)
		}
	}
	for key, ids := range byUID {
		slices.Sort(ids)
		byUID[key] = slices.Compact(ids)
	}
	for key, ids := range accountsByGroup {
		slices.Sort(ids)
		accountsByGroup[key] = slices.Compact(ids)
	}
	connect := func(a, b string) {
		if len(out[a]) < privilegeLinkLimit && !slices.Contains(out[a], b) {
			out[a] = append(out[a], b)
		}
		if len(out[b]) < privilegeLinkLimit && !slices.Contains(out[b], a) {
			out[b] = append(out[b], a)
		}
	}
	evidence := map[string]i.ProcessValues{}
	for _, entry := range snapshot.Evidence {
		if values, ok := entry.Values.(i.ProcessValues); ok {
			evidence[entry.EvidenceID] = values
		}
	}
	for _, finding := range snapshot.Findings {
		if facts, ok := finding.Facts.(i.USBFacts); ok && facts.Input != nil {
			for _, consumer := range facts.Input.Consumers {
				matches := byUID[consumer.EffectiveUID]
				for _, account := range matches[:min(len(matches), privilegeLinkLimit)] {
					connect(finding.FindingID, account)
				}
			}
		}
		if finding.Kind == i.Endpoint {
			for _, id := range finding.EvidenceIDs {
				value := evidence[id]
				if value.Service == nil || value.Service.EffectiveUID == nil {
					continue
				}
				matches := byUID[*value.Service.EffectiveUID]
				for _, account := range matches[:min(len(matches), privilegeLinkLimit)] {
					connect(finding.FindingID, account)
				}
			}
		}
		if rule, ok := finding.Facts.(i.SudoFacts); ok {
			matches := accountsByGroup[rule.Principal.Name]
			if rule.Principal.Type == "user" {
				matches = nil
				if id, ok := accountsByName[rule.Principal.Name]; ok {
					matches = []string{id}
				}
			}
			for _, name := range rule.RunAsUsers {
				if id, ok := accountsByName[name]; ok {
					connect(finding.FindingID, id)
				}
			}
			for _, accountID := range matches[:min(len(matches), privilegeLinkLimit)] {
				connect(finding.FindingID, accountID)
			}
		}

	}
	for id := range out {
		slices.Sort(out[id])
	}
	return out
}

func accountBlocks(snapshot *i.Snapshot, language string) []block {
	l := localizer{language == "ja"}
	group := block{ID: "account-groups", Title: l.text("Local group directory", "ローカルグループ一覧"), Note: l.text("Configured memberships only, not runtime groups or login/privilege certification. Empty groups are retained. Account and sudo judgments are reviewed separately.", "アカウント設定上の一覧です。実行中グループやログイン可否・実効権限を示しません。メンバーのないグループも残します。判断はアカウントとsudo設定ごとに行います。")}
	for _, g := range LocalGroups(snapshot) {
		primary := []string{}
		for _, finding := range snapshot.Findings {
			if facts, ok := finding.Facts.(i.AccountFacts); ok && facts.PrimaryGID == g.GID {
				primary = append(primary, finding.Identity.(i.AccountIdentity).Name)
			}
		}
		slices.Sort(primary)
		group.Fields = append(group.Fields, field{fmt.Sprintf("%s (GID %d)", g.Name, g.GID), l.text("Primary: ", "基本グループ：") + strings.Join(primary, ", ") + l.text(" / Explicit members: ", " ／ 明示メンバー：") + strings.Join(g.Members, ", ")})
	}
	collected := false
	for _, coverage := range snapshot.Coverage {
		if coverage.Collector == i.Accounts {
			collected = coverage.Status != i.NotCollected
		}
	}
	if !collected {
		group.Note += l.text(" Local accounts/groups were not collected in this file; a new collection can supply them.", " このファイルではローカルアカウント・グループは未収集です。再収集で取得できます。")
	}
	if len(group.Fields) == 0 {
		group.Note += l.text(" No group entries were collected; inspect source coverage.", " グループ項目はありません。収集状態を確認してください。")
	}
	return []block{group}
}
