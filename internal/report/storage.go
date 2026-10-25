package report

import (
	"fmt"
	"strings"

	i "github.com/pirakansa/avera/internal/inventory"
)

// StorageFields presents connection and mount metadata, never access certification.
func StorageFields(finding i.Finding, language string) []ServiceField {
	facts, ok := finding.Facts.(i.StorageFacts)
	if !ok {
		return nil
	}
	l := localizer{language == "ja"}
	unknown := l.text("Unknown / not reported", "不明・情報なし")
	value := func(text string) string {
		if text == "" {
			return unknown
		}
		return text
	}
	connection := l.text("Removable storage candidate", "取り外し可能なストレージ候補")
	if facts.Connection == "usb" {
		connection = l.text("USB-connected storage candidate", "USB接続のストレージ候補")
	}
	mode := l.text("Read/write flag (not a permission guarantee)", "読み書き可能フラグ（実際の権限を保証しません）")
	if facts.ReadOnly {
		mode = l.text("Read-only device flag", "デバイスの読み取り専用フラグ")
	}
	fields := []ServiceField{{l.text("Connection classification", "接続の分類"), connection}, {l.text("Vendor", "メーカー"), value(facts.Vendor)}, {l.text("Model", "製品名"), value(facts.Model)}, {l.text("Observed serial", "観測したシリアル番号"), value(facts.Serial)}, {l.text("Capacity", "容量"), fmt.Sprintf("%.2f GiB (%d bytes)", float64(facts.SizeBytes)/(1<<30), facts.SizeBytes)}, {l.text("Device read-only state", "デバイスの読み取り専用状態"), mode}}
	mounted := false
	for _, volume := range facts.Volumes {
		if volume.Partition == 0 && len(volume.Mounts) == 0 && len(facts.Volumes) > 1 {
			continue
		}
		label := l.text("Whole device", "デバイス全体")
		if volume.Partition > 0 {
			label = fmt.Sprintf(l.text("Partition %d", "パーティション %d"), volume.Partition)
		}
		points := []string{}
		for _, mount := range volume.Mounts {
			mounted = true
			access := l.text("rw flag", "rwフラグ")
			if mount.ReadOnly {
				access = l.text("ro flag", "roフラグ")
			}
			points = append(points, mount.Path+" · "+mount.Filesystem+" · "+access)
		}
		if len(points) == 0 {
			points = append(points, l.text("Not mounted in the observed namespace", "観測した名前空間では未マウント"))
		}
		fields = append(fields, ServiceField{label, strings.Join(points, "\n")})
	}
	if !mounted {
		fields = append(fields, ServiceField{l.text("Mount state", "マウント状態"), l.text("Connected but not mounted; still requires a connection-purpose judgment.", "接続中・未マウントです。接続の用途を確認する対象に含めます。")})
	}
	if strings.HasPrefix(finding.Identity.(i.StorageIdentity).Key, "unverified-device:") {
		fields = append(fields, ServiceField{l.text("Repeat matching", "次回との対応付け"), l.text("No unique persistent identifier; a new collection may require a fresh judgment.", "一意な継続識別子がありません。再収集では別の項目として判断が必要になる場合があります。")})
	}
	return fields
}

// StorageStatus distinguishes uncollected inputs, successful empty and incomplete reads.
func StorageStatus(snapshot *i.Snapshot, language string) string {
	l := localizer{language == "ja"}
	count := 0
	for _, finding := range snapshot.Findings {
		if finding.Kind == i.ExternalStorage {
			count++
		}
	}
	for _, coverage := range snapshot.Coverage {
		if coverage.Collector != i.Storage {
			continue
		}
		switch coverage.Status {
		case i.Complete:
			return fmt.Sprintf(l.text("Connected USB/removable non-root storage candidates: %d. Mounted and unmounted devices are included.", "接続中のUSB・取り外し可能なストレージ候補：%d台。rootfs用ディスクを除き、未マウントも含めています。"), count)
		case i.Partial, i.Failed:
			return fmt.Sprintf(l.text("Storage collection is incomplete (%d candidates). Inspect source coverage and diagnostics; this is not an absence claim.", "ストレージの収集は不完全です（候補%d台）。収集範囲と診断を確認してください。未検出は不存在の意味ではありません。"), count)
		}
	}
	return l.text("External storage was not collected in this file. A new collection can supply it.", "このファイルでは外付けストレージは未収集です。再収集で取得できます。")
}
