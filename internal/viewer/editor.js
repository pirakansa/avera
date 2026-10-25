'use strict';
let revision = 0;
const drafts = new Map();
let expanded = new Set();
let currentItems = [];
let routes = [];
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
    currentItems = state.Items;
    routes = state.Routes || [];
    $('target').textContent = '対象: ' + state.Target + ' / 実行: ' + state.Run;
    $('metrics').replaceChildren();
    for (const [key, label] of [
            ['current', '主な確認対象'],
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
    $('reference-count').textContent = state.ReferenceCount;
    $('reference-items').replaceChildren();
    $('routes').replaceChildren();
    for (const route of [...routes, ...(state.RouteHistory || [])]) renderRoute(route, (state.RouteHistory || []).includes(route));
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
    box.id = 'finding-' + f.finding_id;
    if (item.reference) {
        box.append(node('summary', title + ' — 参考情報（個別判断不要）'), node('p', 'ループバック限定の接続口です。主な確認対象の件数には含めず、自動的な追加対応不要の判断もしません。'), node('pre', JSON.stringify({
            identity: f.identity,
            facts: f.facts
        }, null, 2)));
        for (const a of item.judgments.filter(a => a.actor || a.purpose || a.rationale || a.exposure || a.safeguards || (a.evidence_refs || []).length || a.decision !== 'unreviewed')) box.append(judgment(a));
        $('reference-items').append(box);
        return;
    }
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
    const relatedControls = [];
    if (f.kind === 'network.endpoint') {
        const section = node('fieldset');
        section.append(node('legend', '転送先・関連するローカルサービス（人の申告）'));
        section.append(node('p', '入口から内部サービスまでの経路と保護策をここで説明します。チェックしても転送設定を検証したことにはなりません。'));
        const savedRoutes = routes.filter(route => route.entrance_id === f.finding_id);
        const selected = draft ? draft.related_ids : savedRoutes.map(route => route.internal_id);
        for (const target of currentItems.filter(item => item.reference)) {
            const endpoint = target.finding.identity;
            const wrapper = node('label', undefined, 'route-choice');
            const checkbox = node('input');
            checkbox.type = 'checkbox';
            checkbox.name = 'related_ids';
            checkbox.value = target.finding.finding_id;
            checkbox.checked = (selected || []).includes(checkbox.value);
            wrapper.append(checkbox, node('span', String(endpoint.transport).toUpperCase() + ' ' + endpoint.local_address + ':' + endpoint.local_port));
            section.append(wrapper);
            relatedControls.push(checkbox);
        }
        if (!relatedControls.length) section.append(node('p', '今回の収集では関連付けできるローカル限定サービスはありません。'));
        if (savedRoutes.some(route => !routeMatches(route))) section.append(node('p', '保存後に観測が変化したか、関連するサービスが未観測です。以前の関連付けを再確認してください。', 'notice'));
        const wrapper = node('label', '転送経路と保護策の説明');
        const description = node('textarea');
        description.name = 'route_explanation';
        description.maxLength = 20000;
        description.value = draft ? draft.route_explanation || '' : [...new Set(savedRoutes.map(route => route.explanation))].join('\n');
        wrapper.append(description);
        controls.route_explanation = description;
        section.append(wrapper);
        form.append(section);
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
        draft.related_ids = relatedControls.filter(control => control.checked).map(control => control.value);
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
        if (f.kind === 'network.endpoint') {
            edit.related_ids = relatedControls.filter(control => control.checked).map(control => control.value);
            edit.route_explanation = controls.route_explanation.value;
        }
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
                const detail = response.status === 409 ? '別の保存が先に行われました。入力内容を控えてからページを再読み込みしてください。' : response.status === 400 ? '保存できません。判断・判断者・判断理由を確認してください。関連付けを選んだ場合は転送経路の説明も必要です。' : '保存に失敗しました。入力は画面に残っています。出力先を確認してください。';
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

function routeMatches(route) {
    const entrance = currentItems.find(item => item.finding.finding_id === route.entrance_id);
    const internal = currentItems.find(item => item.finding.finding_id === route.internal_id);
    return entrance && internal && !entrance.reference && internal.reference && entrance.finding.fact_fingerprint === route.entrance_fingerprint && internal.finding.fact_fingerprint === route.internal_fingerprint;
}

function renderRoute(route, archived) {
    const box = node('div', undefined, 'prior');
    box.append(node('p', archived ? '置き換え前の関連付け（履歴）' : routeMatches(route) ? '保存時の観測内容と一致・転送経路は未検証' : '観測の変化・未観測あり。関連付けの再確認が必要です'), node('p', route.explanation), node('p', route.actor + ' / ' + route.recorded_at));
    for (const id of [route.entrance_id, route.internal_id]) {
        const item = currentItems.find(item => item.finding.finding_id === id);
        if (!item) {
            box.append(node('p', '今回未観測の関連項目: ' + id));
            continue;
        }
        const endpoint = item.finding.identity;
        const anchor = node('a', String(endpoint.transport).toUpperCase() + ' ' + endpoint.local_address + ':' + endpoint.local_port);
        anchor.href = '#finding-' + id;
        anchor.addEventListener('click', () => {
            if (item.reference) $('local-reference').open = true;
            const target = $('finding-' + id);
            if (target) target.open = true;
        });
        box.append(anchor, node('span', ' '));
    }
    $('routes').append(box);
}
