package report

import (
	"fmt"
	"path"
	"strings"

	i "github.com/pirakansa/avera/internal/inventory"
)

func inputConsumerText(c i.USBInputConsumer, language string) string {
	l := localizer{language == "ja"}
	return fmt.Sprintf("%s · %s · %s %d · %s %d · %s %v", c.ProcessName, c.Executable, l.text("effective UID", "実効UID"), c.EffectiveUID, l.text("effective GID", "実効GID"), c.EffectiveGID, l.text("runtime supplementary groups", "実行中の補助グループ"), c.SupplementaryGroups)
}
func usbInputFields(f i.USBFacts, language string) []ServiceField {
	l := localizer{language == "ja"}
	if f.Input == nil {
		return []ServiceField{{l.text("Input route collection", "入力経路の収集"), l.text("Not collected in this observation. This does not establish no consumers; recollect for metadata.", "この観測では入力経路は未収集です。受け取り先がないことを示しません。再収集で取得できます。")}}
	}
	v := f.Input
	fields := []ServiceField{{l.text("Input route collection", "入力経路の収集"), l.state(string(v.Status))}, {l.text("Observed kernel input handlers", "観測したカーネルの入力ハンドラ"), strings.Join(v.Handlers, ", ")}}
	if len(v.Consumers) == 0 {
		message := l.text("No directly open evdev descriptor observed at collection time. Kernel handlers and other input paths may still operate.", "観測時点では、evdev入力口を直接開いているプロセスは見つかりませんでした。カーネルのハンドラや他の入力経路が動く可能性はあります。")
		if v.Status != i.Complete {
			message = l.text("Consumer enumeration is incomplete; no absence conclusion.", "受け取り先の列挙は不完全です。不在とは判断できません。")
		}
		fields = append(fields, ServiceField{l.text("Processes with open input descriptors", "入力口を開いているプロセス"), message})
	}
	for _, c := range v.Consumers {
		fields = append(fields, ServiceField{l.text("Observed open input descriptor holder", "入力口を開いているプロセス（観測）"), inputConsumerText(c, language)})
	}
	if len(v.DiagnosticCodes) > 0 {
		fields = append(fields, ServiceField{l.text("Input route diagnostics", "入力経路の診断"), strings.Join(v.DiagnosticCodes, ", ")})
	}
	fields = append(fields, ServiceField{l.text("Input relationship boundary", "入力の関連付けの範囲"), l.text("An open descriptor is not proof of delivered events or the focused application. kbd is a kernel virtual-terminal handler, not a link to a particular tty or shell. No input contents were read.", "開かれた入力口との対応です。実際の入力配送や操作先アプリは検証していません。kbdはカーネルの仮想端末ハンドラで、特定のttyやシェルへの対応ではありません。入力内容は読み取っていません。")})
	return fields
}

// USBObservationFields includes transient device names only as observation context.
func USBObservationFields(snapshot *i.Snapshot, finding i.Finding, language string) []ServiceField {
	fields := USBFields(finding, language)
	id, ok := finding.Identity.(i.USBIdentity)
	if !ok {
		return fields
	}
	l := localizer{language == "ja"}
	for _, d := range i.USBInputFor(snapshot, id.Key) {
		label := d.Node
		if label == "" {
			label = l.text("Kernel handler only / no evdev node", "カーネルのハンドラ・evdev入力口なし")
		}
		fields = append(fields, ServiceField{l.text("Observed input attachment", "観測した入力口の対応"), label + " · " + d.DeviceNumber + "\n" + d.SysfsPath})
	}
	return fields
}

func inputPhysicalNodes(snapshot *i.Snapshot, key, language string) []*PhysicalNode {
	l := localizer{language == "ja"}
	out := []*PhysicalNode{}
	for _, d := range i.USBInputFor(snapshot, key) {
		title := d.Node
		if title == "" {
			title = l.text("Kernel input device (no evdev node)", "カーネルの入力機器（evdev入力口なし）")
		}
		inputName := path.Base(path.Dir(d.SysfsPath))
		if d.Node == "" {
			inputName = path.Base(d.SysfsPath)
		}
		inputTitle := l.text("Linux input device · ", "Linuxの入力機器 · ") + inputName
		if d.Node == "" {
			inputTitle = title
		}
		node := &PhysicalNode{Card: NetworkCard{Title: inputTitle, Detail: l.text("sysfs parent matches USB device", "sysfsの親がUSB機器と一致"), Decision: l.text("Observed attachment; not event delivery", "観測した接続・入力配送は未検証")}, Children: []*PhysicalNode{}}
		kernelHandlers := []string{}
		for _, h := range d.Handlers {
			if h != "evdev" {
				kernelHandlers = append(kernelHandlers, h)
			}
		}
		if len(kernelHandlers) > 0 {
			node.Children = append(node.Children, &PhysicalNode{Card: NetworkCard{Title: l.text("Kernel input handlers", "カーネルの入力ハンドラ"), Detail: strings.Join(kernelHandlers, ", "), Decision: l.text("kbd does not identify a particular tty or shell", "kbdから特定のtty・シェルは特定しません")}, Children: []*PhysicalNode{}})
		}
		consumerParent := node
		if d.Node != "" {
			event := &PhysicalNode{Card: NetworkCard{Title: d.Node, Detail: d.DeviceNumber, Decision: l.text("evdev character-device interface", "evdevの文字デバイス入力口")}, Children: []*PhysicalNode{}}
			node.Children = append(node.Children, event)
			consumerParent = event
		}
		for _, c := range d.Consumers {
			matches := []string{}
			for _, f := range snapshot.Findings {
				if account, ok := f.Facts.(i.AccountFacts); ok && account.UID == c.EffectiveUID {
					matches = append(matches, f.FindingID)
				}
			}
			accountID := ""
			if len(matches) == 1 {
				accountID = matches[0]
			}
			consumerParent.Children = append(consumerParent.Children, &PhysicalNode{Card: NetworkCard{FindingID: accountID, Anchor: "finding-" + accountID, Title: c.ProcessName, Detail: inputConsumerText(c, language), Decision: l.text("Character-device descriptor open at observation", "観測時点で文字デバイスのFDを保持")}, Children: []*PhysicalNode{}})
		}
		out = append(out, node)
	}
	return out
}
