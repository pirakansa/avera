"""Storage inventory: unmounted devices, purpose-only scope, immutable save/export."""
import hashlib, json, pathlib, re, signal, subprocess, tempfile
from playwright.sync_api import sync_playwright, expect

out=pathlib.Path(tempfile.mkdtemp(prefix='avera-storage-browser-'))
snapshot=json.loads(pathlib.Path('test/fixtures/prototype/contracts/snapshot-first.json').read_text())
snapshot['contract_revision']='storage-2026-10-25.1'
snapshot['coverage'].append({'collector':'storage','collector_version':'fictional','scope':'fictional USB/removable non-root storage','status':'complete','attempted_sources':['/sys/class/block','/proc/self/mountinfo'],'diagnostic_codes':[],'limitations':['Fictional devices only; no real USB devices or file contents inspected.']})
def digest(value):
 return 'sha256:'+hashlib.sha256(json.dumps(value,ensure_ascii=False,sort_keys=True,separators=(',',':')).encode()).hexdigest()
for name,mounted in [('Fictional backup SSD',True),('Fictional transfer USB <script>window.injected=1</script>',False)]:
 key='wwid:demo-'+('backup' if mounted else 'transfer')
 identity={'key':key}
 facts={'connection':'usb','vendor':'Fictional','model':name,'serial':'DEMO-BACKUP' if mounted else 'DEMO-TRANSFER','size_bytes':32<<30,'read_only':False,'observation_complete':True,'volumes':[{'partition':1 if mounted else 0,'mounts':[{'path':'/srv/app-data','filesystem':'ext4','read_only':False}] if mounted else []}]}
 eid='demo-storage-backup' if mounted else 'demo-storage-transfer'
 snapshot['evidence'].append({'evidence_id':eid,'source_kind':'local_file','location':'/sys/class/block/fictional-'+eid,'collected_at':snapshot['collected_at'],'method':'Fictional storage metadata, no contents','values':{'excerpt':json.dumps(facts)}})
 snapshot['findings'].append({'collector':'storage','kind':'storage.external','identity':identity,'facts':facts,'finding_id':digest(['avera.identity.v1',snapshot['target_id'],'storage','storage.external',identity]),'fact_fingerprint':digest(['avera.facts.v1','storage.external',identity,facts]),'evidence_ids':[eid]})
(out/'snapshot.json').write_text(json.dumps(snapshot))
subprocess.run(['./bin/avera','review','--single-report','--snapshot',str(out/'snapshot.json'),'--output',str(out/'report.json')],check=True,capture_output=True)
server=subprocess.Popen(['./bin/avera','serve','--input',str(out/'report.json'),'--output-dir',str(out)],stderr=subprocess.PIPE,text=True)
try:
 url=re.search(r'http://[0-9.:]+',server.stderr.readline()).group()
 with sync_playwright() as p:
  browser=p.chromium.launch(headless=True,args=['--no-sandbox']);page=browser.new_page(viewport={'width':1280,'height':960});errors=[];page.on('pageerror',lambda error:errors.append(str(error)))
  page.goto(url);page.locator('#items > details').first.wait_for()
  assert '2台' in page.locator('#storage-status').inner_text()
  items=page.locator('#items > details').filter(has=page.locator(':scope > summary',has_text=re.compile('^外付けストレージ')))
  assert items.count()==2
  unmounted=items.filter(has=page.locator(':scope > summary',has_text='Fictional transfer'))
  unmounted.locator(':scope > summary').click()
  assert '接続中・未マウント' in unmounted.inner_text()
  assert unmounted.locator('[name=exposure], [name=safeguards], [name=references]').count()==0
  assert page.evaluate('window.injected') is None
  before=page.request.get(url+'/download.json').json()['current']
  unmounted.locator('[name=purpose]').fill('○○アプリの保存先。管理担当：架空の運用チーム')
  unmounted.locator('[name=rationale]').fill('接続の用途が確認でき、業務上必要です')
  unmounted.locator('[name=actor]').fill('fictional-reviewer')
  unmounted.locator('select').select_option('accepted')
  unmounted.locator('button[type=submit]').click()
  page.locator('#status').filter(has_text='保存しました').wait_for()
  expect(unmounted.locator(':scope > summary')).to_contain_text('追加対応不要')
  after=page.request.get(url+'/download.json').json()
  assert after['current']==before
  assert any(a['selector']['kind']=='storage.external' and a['decision']=='accepted' for a in after['assessments']['entries'])
  preview=browser.new_page(java_script_enabled=False)
  exported=page.request.get(url+'/export.html').text();assert '<script>' not in exported and '&lt;script&gt;' in exported
  (out/'report.html').write_text(exported);preview.goto((out/'report.html').as_uri());preview.locator('.report-tabs a[href="#report-details"]').click()
  assert '○○アプリの保存先' in preview.locator('#report-details').inner_text() and '接続中・未マウント' in preview.locator('#report-details').inner_text()
  assert '読み書き可能フラグ' in preview.locator('#report-details').inner_text()
  page.screenshot(path=str(out/'editor.png'),full_page=True)
  for target in [page,preview]:
   target.set_viewport_size({'width':390,'height':844});assert target.evaluate('document.documentElement.scrollWidth <= innerWidth')
  assert not errors,errors
  browser.close()
 print('PASS: connected mounted/unmounted storage, concise reason form, accepted judgment, immutable JSON, escaped offline HTML and narrow layouts')
 print('Browser artifacts:',out)
finally:
 server.send_signal(signal.SIGTERM);server.wait(timeout=10);assert server.returncode==0
