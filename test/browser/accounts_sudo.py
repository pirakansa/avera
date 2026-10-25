"""Fictional accounts, runtime groups, sudo relationships and scoped review."""
import hashlib, json, pathlib, re, signal, subprocess, tempfile
from playwright.sync_api import sync_playwright, expect

out = pathlib.Path(tempfile.mkdtemp(prefix='avera-accounts-sudo-'))
snapshot = json.loads(pathlib.Path('test/fixtures/prototype/contracts/snapshot-first.json').read_text())
snapshot['contract_revision'] = 'accounts-2026-10-25.1'
def digest(value):
    return 'sha256:' + hashlib.sha256(json.dumps(value, ensure_ascii=False, sort_keys=True, separators=(',', ':')).encode()).hexdigest()
groups = [{'name': 'fictional-users', 'gid': 1000, 'members': []}, {'name': 'fictional-maint', 'gid': 3000, 'members': ['operator']}, {'name': 'empty', 'gid': 4000, 'members': []}]
snapshot['coverage'].append({'collector': 'accounts', 'collector_version': 'fictional', 'status': 'complete', 'scope': 'local passwd/group files', 'attempted_sources': ['/etc/passwd', '/etc/group'], 'diagnostic_codes': [], 'limitations': []})
snapshot['evidence'].append({'evidence_id': 'fictional-group-directory', 'source_kind': 'local_identity', 'location': '/etc/group', 'collected_at': snapshot['collected_at'], 'method': 'Fictional sanitized group directory', 'values': {'groups': groups}})
for name in ['operator', 'alias']:
    identity = {'name': name}
    memberships = [{'name': 'fictional-users', 'gid': 1000, 'membership': 'primary'}]
    if name == 'operator': memberships.insert(0, {'name': 'fictional-maint', 'gid': 3000, 'membership': 'supplementary'})
    facts = {'uid': 1000, 'primary_gid': 1000, 'home': '/home/' + name, 'shell': '/bin/sh', 'groups': memberships, 'context_complete': True}
    finding = {'collector': 'accounts', 'kind': 'account.local', 'identity': identity, 'facts': facts, 'evidence_ids': ['fictional-group-directory']}
    finding['finding_id'] = digest(['avera.identity.v1', snapshot['target_id'], 'accounts', 'account.local', identity])
    finding['fact_fingerprint'] = digest(['avera.facts.v1', 'account.local', identity, facts])
    snapshot['findings'].append(finding)
for evidence in snapshot['evidence']:
    if evidence['source_kind'] == 'proc' and evidence['evidence_id'] == 'e1':
        evidence['values']['service'] = {'process_name': 'fictional-web', 'effective_uid': 1000, 'effective_gid': 3000, 'supplementary_groups': [3000], 'unit_candidate': 'fictional-web.service', 'cgroup_path': '/system.slice/fictional-web.service', 'diagnostics': []}
# Keep the existing unsupported sudo observation to verify separate collection limits.
rule = next(f for f in snapshot['findings'] if f['kind'] == 'sudo.rule')
rule['facts']['principal'] = {'type': 'group', 'name': 'fictional-maint'}
rule['facts']['local_members'] = [{'name': 'operator', 'uid': 1000, 'membership': 'supplementary'}]
rule['fact_fingerprint'] = digest(['avera.facts.v1', rule['kind'], rule['identity'], rule['facts']])
(out / 'snapshot.json').write_text(json.dumps(snapshot))
subprocess.run(['./bin/avera', 'review', '--single-report', '--snapshot', str(out / 'snapshot.json'), '--output', str(out / 'report.json')], check=True, capture_output=True)
server = subprocess.Popen(['./bin/avera', 'serve', '--input', str(out / 'report.json'), '--output-dir', str(out)], stderr=subprocess.PIPE, text=True)
try:
    url = re.search(r'http://[0-9.:]+', server.stderr.readline()).group()
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True, args=['--no-sandbox'])
        page = browser.new_page(viewport={'width': 1280, 'height': 960})
        errors = []; page.on('pageerror', lambda error: errors.append(str(error)))
        page.goto(url); page.locator('#items > details').first.wait_for()
        state = page.request.get(url + '/api/state').json()
        assert state['Summary']['current'] == 5 and state['Summary']['unreviewed'] == 5
        assert len([item for item in state['Items'] if item['finding']['kind'] == 'account.local']) == 2
        assert '追加の調査・判断は求めません' in page.locator('#review-boundary').inner_text()
        web = page.locator('#items > details').filter(has=page.locator(':scope > summary', has_text=':80 '))
        web.locator(':scope > summary').click()
        assert '3000 (fictional-maint)' in web.locator('.service-inventory').inner_text()
        assert 'PID' not in web.locator('.service-inventory').inner_text()
        assert web.locator('.related-privileges a').count() == 2
        web.locator('.related-privileges a', has_text='アカウント operator').click()
        account = page.locator('#items > details').filter(has=page.locator(':scope > summary', has_text='ローカルアカウント operator'))
        assert account.get_attribute('open') is not None
        assert account.locator('.related-privileges a', has_text='sudo').count() == 1
        account.locator('[name=purpose]').fill('メンテナンス用。管理担当：架空の運用チーム <script>window.injected=1</script>')
        account.locator('[name=rationale]').fill('担当業務に必要なグループとsudo設定を確認')
        account.locator('[name=actor]').fill('fictional-owner')
        account.locator('select').select_option('accepted')
        account.locator('button[type=submit]').click()
        page.locator('#status').filter(has_text='保存しました').wait_for()
        account = page.locator('#items > details').filter(has=page.locator(':scope > summary', has_text='ローカルアカウント operator'))
        expect(account.locator(':scope > summary')).to_contain_text('追加対応不要')
        account.locator('.related-privileges a', has_text='sudo').click()
        sudo = page.locator('#items > details').filter(has=page.locator(':scope > summary', has_text=re.compile(r'^sudo ')))
        assert sudo.get_attribute('open') is not None
        assert 'sudo設定の対象' in sudo.inner_text() and '%fictional-maint' in sudo.inner_text()
        data = page.request.get(url + '/download.json').json()
        assert any(a['decision'] == 'accepted' and a['selector']['kind'] == 'account.local' for a in data['assessments']['entries'])
        metadata = next(e['values']['service'] for e in data['current']['evidence'] if e['evidence_id'] == 'e1')
        assert metadata['effective_gid'] == 3000 and metadata['supplementary_groups'] == [3000]
        exported = page.request.get(url + '/export.html').text()
        assert '&lt;script&gt;' in exported and '<script>' not in exported
        (out / 'report.html').write_text(exported)
        preview = browser.new_page(java_script_enabled=False); preview.goto((out / 'report.html').as_uri())
        assert '追加の調査・判断は求めません' in preview.locator('#report-summary').inner_text()
        preview.locator('.report-tabs a[href="#report-details"]').click()
        assert 'ローカルグループ一覧' in preview.locator('#report-details').inner_text()
        assert 'operator' in preview.locator('#account-groups').inner_text()
        assert page.evaluate('window.injected') is None
        page.screenshot(path=str(out / 'editor.png'), full_page=True)
        preview.screenshot(path=str(out / 'report.png'), full_page=True)
        for target in [page, preview]:
            target.set_viewport_size({'width': 390, 'height': 844})
            assert target.evaluate('document.documentElement.scrollWidth <= innerWidth')
        assert not errors, errors
        browser.close()
    print('PASS: local accounts, separate runtime groups, service/account/sudo links, account save, scope exclusions, escaped offline HTML and narrow layouts')
    print('Browser artifacts:', out)
finally:
    server.send_signal(signal.SIGTERM); server.wait(timeout=10); assert server.returncode == 0
