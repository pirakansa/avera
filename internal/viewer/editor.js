'use strict';
let revision = 0;
const drafts = new Map();
let expanded = new Set();
const $ = id => document.getElementById(id);
const names = {
    accepted: '追加対応不要',
    action_required: '対応が必要',
    unreviewed: '未確認',
    applicable: '適用可能',
    needs_review: '再確認が必要',
    conflict: '判断の競合',
    none: '判断なし',
    changed: '変更あり',
    unchanged: '変更なし',
    new: '新規',
    not_compared: '比較なし',
    unknown: '不明',
    not_observed: '今回は未観測'
};
const text = value => names[value] || value;

function node(tag, value, cls) {
    const e = document.createElement(tag);
    if (value !== undefined) e.textContent = value;
    if (cls) e.className = cls;
    return e;
}

function message(value, error = false) {
    $('status').textContent = value;
    $('status').className = error ? 'error' : 'success';
}

function judgment(a) {
    const box = node('div', undefined, 'prior');
    box.append(node('p', text(a.decision) + ' / ' + (a.actor || '未記入') + ' / ' + (a.assessed_at || '未記入')));
    for (const [key, label] of [
            ['purpose', '目的'],
            ['exposure', '利用者・公開範囲'],
            ['safeguards', '保護策（人の申告）'],
            ['rationale', '判断理由']
        ]) box.append(node('p', label + ': ' + (a[key] || '未記入')));
    for (const ref of a.evidence_refs || []) box.append(node('p', '証拠の参照: ' + ref.reference));
    return box;
}
async function load() {
    const response = await fetch('/api/state');
    if (!response.ok) throw new Error('読み込みに失敗しました');
    const state = await response.json();
    revision = state.revision;
    $('target').textContent = '対象: ' + state.Target + ' / 実行: ' + state.Run;
    $('metrics').replaceChildren();
    for (const [key, label] of [
            ['current', '今回の項目'],
            ['accepted', '追加対応不要'],
            ['action_required', '対応が必要'],
            ['unreviewed', '未確認'],
            ['needs_review', '再確認が必要（内数）'],
            ['conflict', '判断の競合（内数）'],
            ['historical', '過去の項目'],
            ['unmatched_assessments', '対応する観測がない判断']
        ]) {
        const card = node('div', undefined, 'metric');
        card.append(node('strong', state.Summary[key] || 0), node('span', label));
        if (state.Summary[key] > 0 && (key === 'action_required' || key === 'unreviewed')) {
            card.classList.add(key === 'action_required' ? 'metric-action' : 'metric-unreviewed');
            card.append(node('span', key === 'action_required' ? '優先して対応' : '判断が必要', 'metric-notice'));
        }
        $('metrics').append(card);
    }
    $('coverage').textContent = JSON.stringify(state.Coverage, null, 2);
    $('history').replaceChildren();
    for (const a of state.History.entries) $('history').append(judgment(a));
    $('unmatched').replaceChildren();
    for (const a of state.Unmatched.entries) $('unmatched').append(judgment(a));
    expanded = new Set([...document.querySelectorAll('#items > details[open]')].map(item => item.dataset.finding));
    $('items').replaceChildren();
    for (const item of state.Items) renderItem(item);
}

function renderItem(item) {
    const f = item.finding,
        r = item.review,
        g = item.guidance;
    const id = f.identity;
    const title = f.kind === 'network.endpoint' ? String(id.transport).toUpperCase() + ' ' + id.local_address + ':' + id.local_port : (f.kind === 'sudo.rule' ? 'sudo ' : '解釈できない設定 ') + id.source_path + ':' + id.line;
    const box = node('details');
    box.dataset.finding = f.finding_id;
    box.open = expanded.has(f.finding_id) || drafts.has(f.finding_id);
    box.append(node('summary', title + ' — ' + text(r.current_decision) + ' / ' + text(r.applicability) + ' / ' + text(r.change)));
    box.append(node('p', g.Observed), node('p', g.Why));
    const qs = node('ul');
    for (const q of g.Questions) qs.append(node('li', q));
    box.append(qs);
    const facts = node('details');
    facts.append(node('summary', '観測内容（編集できません）'), node('pre', JSON.stringify({
        identity: f.identity,
        facts: f.facts
    }, null, 2)));
    box.append(facts);
    const savedJudgments = item.judgments.filter(a => a.actor || a.purpose || a.exposure || a.safeguards || a.rationale || (a.evidence_refs || []).length || a.decision !== 'unreviewed');
    if (savedJudgments.length) {
        box.append(node('h3', '保存済みの説明・判断'));
        for (const a of savedJudgments) box.append(judgment(a));
    }
    if (r.applicability === 'needs_review' || r.applicability === 'conflict') box.append(node('p', '以前の判断は今回「追加対応不要」と判断したことを意味しません。変化や不足する証拠を確認し、今回の判断を明示的に保存してください。', 'notice'));
    const prior = item.judgments.length === 1 ? item.judgments[0] : {};
    const draft = drafts.get(f.finding_id);
    const a = draft ? {
        ...prior,
        ...draft
    } : prior;
    const form = node('form');
    form.append(node('h3', '今回の説明・判断'));
    const controls = {};
    for (const [key, label] of [
            ['purpose', '目的・必要な機能'],
            ['exposure', '利用者・公開範囲'],
            ['safeguards', '保護策（ツールで検証した事実とは区別します）'],
            ['rationale', '今回の判断理由'],
            ['actor', '判断者'],
            ['references', '証拠の参照（1行に1件・URLを取得しません）']
        ]) {
        const wrapper = node('label', label);
        const control = node(key === 'actor' ? 'input' : 'textarea');
        control.name = key;
        control.value = key === 'references' ? (draft ? draft.references : (a.evidence_refs || []).map(x => x.reference).join('\n')) : a[key] || '';
        control.maxLength = 20000;
        wrapper.append(control);
        controls[key] = control;
        form.append(wrapper);
    }
    const select = node('select');
    for (const value of ['unreviewed', 'action_required', 'accepted']) {
        const option = node('option', text(value));
        option.value = value;
        select.append(option);
    }
    select.value = a.decision || 'unreviewed';
    const label = node('label', '判断');
    label.append(select);
    form.append(label);
    const button = node('button', '今回の判断を保存（新しい JSON を作成）');
    button.type = 'submit';
    form.append(button);
    form.addEventListener('input', () => {
        const draft = {
            decision: select.value
        };
        for (const key of Object.keys(controls)) draft[key] = controls[key].value;
        drafts.set(f.finding_id, draft);
        window.onbeforeunload = () => true;
    });
    form.addEventListener('submit', async event => {
        event.preventDefault();
        button.disabled = true;
        const edit = {
            finding_id: f.finding_id,
            decision: select.value
        };
        for (const key of ['purpose', 'exposure', 'safeguards', 'rationale', 'actor']) edit[key] = controls[key].value;
        edit.evidence = [...new Set(controls.references.value.split('\n').map(x => x.trim()).filter(Boolean))].map(reference => ({
            reference,
            provenance: 'human_reported'
        }));
        try {
            const response = await fetch('/api/assess', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    revision,
                    edit
                })
            });
            if (!response.ok) {
                const detail = response.status === 409 ? '別の保存が先に行われました。入力内容を控えてからページを再読み込みしてください。' : response.status === 400 ? '保存できません。判断・判断者・判断理由を確認してください。' : '保存に失敗しました。入力は画面に残っています。出力先を確認してください。';
                throw new Error(detail);
            }
            const result = await response.json();
            message('保存しました: ' + result.saved + '\nこのファイル、または「JSON を持ち帰る」で取得したファイルを、次回の収集に使ってください。');
            drafts.delete(f.finding_id);
            window.onbeforeunload = drafts.size ? () => true : null;
            await load();
        } catch (error) {
            message(error.message, true);
        } finally {
            button.disabled = false;
        }
    });
    box.append(form);
    $('items').append(box);
}
$('edit-tab').addEventListener('click', () => {
    $('editor').hidden = false;
    $('preview').hidden = true;
});
$('preview-tab').addEventListener('click', () => {
    $('preview').src = '/preview';
    $('preview').hidden = false;
    $('editor').hidden = true;
});
load().catch(error => message(error.message, true));

for (const id of ['json-download', 'html-export']) $(id).addEventListener('click', event => {
    if (drafts.size && !window.confirm('未保存の入力は出力に含まれません。保存済みの内容を出力しますか？')) event.preventDefault();
});
