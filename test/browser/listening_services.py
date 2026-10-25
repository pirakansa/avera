"""Port-based service inventory: synthetic observations, immutable save and export."""
import copy, json, pathlib, re, signal, subprocess, tempfile
from playwright.sync_api import sync_playwright

out = pathlib.Path(tempfile.mkdtemp(prefix='avera-listening-services-'))
snapshot = json.loads(pathlib.Path('test/fixtures/prototype/contracts/snapshot-first.json').read_text())
snapshot['contract_revision'] = 'services-2026-10-25.1'
first = snapshot['evidence'][0]
first['values']['service'] = {'process_name': 'fictional-worker <script>window.injected=1</script>', 'effective_uid': 2000, 'unit_candidate': 'fictional-web.service', 'cgroup_path': '/system.slice/fictional-web.service', 'diagnostics': []}
second = copy.deepcopy(first)
second['evidence_id'] = 'fictional-second-holder'
second['values']['pid'] = 54321
second['values']['service']['process_name'] = 'fictional-second-worker'
second['values']['service']['effective_uid'] = 0
second['values']['service']['unit_candidate'] = ''
second['values']['service']['cgroup_path'] = '/fictional-container'
snapshot['evidence'].append(second)
for finding in snapshot['findings']:
    if first['evidence_id'] in finding['evidence_ids']:
        finding['evidence_ids'].append(second['evidence_id'])
(out / 'snapshot.json').write_text(json.dumps(snapshot))
subprocess.run(['./bin/avera', 'review', '--single-report', '--snapshot', str(out / 'snapshot.json'), '--output', str(out / 'report.json')], check=True, capture_output=True)
server = subprocess.Popen(['./bin/avera', 'serve', '--input', str(out / 'report.json'), '--output-dir', str(out)], stderr=subprocess.PIPE, text=True)
try:
    url = re.search(r'http://[0-9.:]+', server.stderr.readline()).group()
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True, args=['--no-sandbox'])
        page = browser.new_page(viewport={'width': 1280, 'height': 960})
        page.goto(url)
        page.locator('#items > details').first.wait_for()
        item = page.locator('#items > details').filter(has=page.locator(':scope > summary', has_text=':80 '))
        item.locator(':scope > summary').click()
        inventory = item.locator('.service-inventory')
        assert 'fictional-web.service' in inventory.inner_text()
        assert '2000' in inventory.inner_text() and '54321' not in inventory.inner_text()
        assert 'PID' not in inventory.inner_text()
        assert 'fictional-second-worker' in inventory.inner_text()
        assert inventory.locator('dd', has_text=re.compile('^0$')).count() >= 1
        assert '不明・未収集' in inventory.inner_text()
        assert page.evaluate('window.injected') is None
        assert 'fictional-web.service' in page.locator('#network-map').inner_text()
        assert page.locator('#reference-items form').count() == 0
        item.locator('[name=purpose]').fill('公開文書の配信。管理担当：架空の運用チーム')
        item.locator('[name=rationale]').fill('待受ポートとサービスの用途を確認')
        item.locator('[name=actor]').fill('fictional-reviewer')
        item.locator('select').select_option('accepted')
        item.locator('button[type=submit]').click()
        page.locator('#status').filter(has_text='保存しました').wait_for()
        data = page.request.get(url + '/download.json').json()
        assert data['current']['evidence'][0]['values']['service'] == first['values']['service']
        exported = page.request.get(url + '/export.html').text()
        assert 'fictional-web.service' in exported and '&lt;script&gt;' in exported
        (out / 'report.html').write_text(exported)
        preview = browser.new_page(java_script_enabled=False)
        preview.goto((out / 'report.html').as_uri())
        preview.locator('.report-tabs a[href="#report-configuration"]').click()
        assert 'fictional-web.service' in preview.locator('#network-map').inner_text()
        preview.locator('#network-map .map-card a').filter(has_text=':80').first.click()
        assert preview.locator('#report-details').is_visible()
        assert '実行ユーザ（プロセス実効UID）' in preview.locator('#report-details').inner_text()
        page.screenshot(path=str(out / 'editor.png'), full_page=True)
        preview.screenshot(path=str(out / 'details.png'), full_page=True)
        for target in [page, preview]:
            target.set_viewport_size({'width': 390, 'height': 844})
            assert target.evaluate('document.documentElement.scrollWidth <= innerWidth')
        browser.close()
    print('PASS: port-linked multiple holders, effective UID 0, unknown service, reference scope, immutable save, escaped offline HTML and narrow layouts')
    print('Browser artifacts:', out)
finally:
    server.send_signal(signal.SIGTERM)
    server.wait(timeout=10)
    assert server.returncode == 0
