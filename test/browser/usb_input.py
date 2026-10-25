"""USB input routes, privilege links and immutable script-free reporting."""
import hashlib, json, pathlib, re, signal, subprocess, tempfile
from playwright.sync_api import sync_playwright, expect

out=pathlib.Path(tempfile.mkdtemp(prefix='avera-usb-input-browser-'))
report=json.loads(pathlib.Path('test/fixtures/prototype/reports/usb-input-demo.json').read_text())
snapshot=report['current']
# Exercise untrusted process names through both evidence and stable projections.
for evidence in snapshot['evidence']:
 if evidence['source_kind']=='usb_metadata':
  for device in evidence['values']['input']['devices']:
   for consumer in device['consumers']:consumer['process_name']='Fictional-Xorg <script>window.injected=1</script>'
  for node in evidence['values']['nodes']:
   for consumer in node['facts'].get('input',{}).get('consumers',[]):consumer['process_name']='Fictional-Xorg <script>window.injected=1</script>'
for finding in snapshot['findings']:
 if finding['kind']=='usb.device':
  for consumer in finding['facts']['input']['consumers']:consumer['process_name']='Fictional-Xorg <script>window.injected=1</script>'
  finding['fact_fingerprint']='sha256:'+hashlib.sha256(json.dumps(['avera.facts.v1','usb.device',finding['identity'],finding['facts']],ensure_ascii=False,sort_keys=True,separators=(',',':')).encode()).hexdigest()
source=json.dumps(report,ensure_ascii=False).encode();(out/'input.json').write_bytes(source)
server=subprocess.Popen(['./bin/avera','serve','--input',str(out/'input.json'),'--output-dir',str(out)],stderr=subprocess.PIPE,text=True)
try:
 url=re.search(r'http://[0-9.:]+',server.stderr.readline()).group()
 with sync_playwright() as p:
  browser=p.chromium.launch(headless=True,args=['--no-sandbox']);page=browser.new_page(viewport={'width':1280,'height':960});errors=[];page.on('pageerror',lambda error:errors.append(str(error)))
  page.goto(url);page.locator('#physical-map .physical-tree').first.wait_for()
  physical=page.locator('#physical-map');assert '/dev/input/event5' in physical.inner_text() and 'Fictional-Xorg' in physical.inner_text()
  assert '実効UID 0' in physical.inner_text() and '実効GID 44' in physical.inner_text()
  assert 'カーネルの入力機器（evdev入力口なし）' in physical.inner_text() and 'kbd' in physical.inner_text()
  assert page.evaluate('window.injected') is None
  account=page.locator('#items > details').filter(has=page.locator(':scope > summary',has_text='ローカルアカウント root'));assert account.count()==1
  rootid=account.get_attribute('id');physical.locator('a[href="#'+rootid+'"]').click();assert account.get_attribute('open') is not None
  mouse=page.locator('#items > details').filter(has=page.locator(':scope > summary',has_text='Fictional operator mouse'))
  physical.locator('a').filter(has_text='Fictional operator mouse').click();assert mouse.get_attribute('open') is not None
  assert '実際の入力配送や操作先アプリは検証していません' in mouse.inner_text()
  mouse.locator('.related-privileges a[href="#'+rootid+'"]').click();assert account.get_attribute('open') is not None
  before=page.request.get(url+'/download.json').json()['current']
  mouse.locator('[name=purpose]').fill('既存アプリの操作用。架空の運用担当が管理')
  mouse.locator('[name=safeguards]').fill('運用マニュアルで物理管理と接続担当者を定義済み')
  mouse.locator('[name=rationale]').fill('入力口を開くプロセスと実行権限を確認して継続利用を判断')
  mouse.locator('[name=actor]').fill('fictional-reviewer');mouse.locator('select').select_option('accepted');mouse.locator('button[type=submit]').click()
  page.locator('#status').filter(has_text='保存しました').wait_for();expect(mouse.locator(':scope > summary')).to_contain_text('追加対応不要')
  assert page.request.get(url+'/download.json').json()['current']==before
  assert (out/'input.json').read_bytes()==source
  exported=page.request.get(url+'/export.html').text();assert '<script>' not in exported and '&lt;script&gt;' in exported
  (out/'report.html').write_text(exported)
  offline=browser.new_page(java_script_enabled=False);offline.goto((out/'report.html').as_uri()+'#report-configuration')
  assert offline.locator('#physical-map').is_visible()
  offline.locator('#physical-map a').filter(has_text='Fictional-Xorg').click();assert offline.locator('#report-details').is_visible()
  assert 'ローカルアカウント root' in offline.locator(offline.evaluate('location.hash')).inner_text()
  for href in offline.locator('a[href^="#"]').evaluate_all('(nodes)=>nodes.map(n=>n.getAttribute("href").slice(1))'):
   assert offline.locator('[id="'+href+'"]').count()==1,href
  for target in [page,offline]:
   target.set_viewport_size({'width':390,'height':844});assert target.evaluate('document.documentElement.scrollWidth <= innerWidth')
  offline.emulate_media(media='print');assert offline.locator('#physical-map').is_visible()
  page.set_viewport_size({'width':1280,'height':960});physical.screenshot(path=str(out/'physical.png'))
  assert not errors,errors;browser.close()
 print('PASS: USB/input/process/privilege tree, root account navigation, kernel-only path, explicit limits, immutable save, escaping and offline/mobile/print HTML')
 print('Browser artifacts:',out)
finally:
 server.send_signal(signal.SIGTERM);server.wait(timeout=10);assert server.returncode==0
