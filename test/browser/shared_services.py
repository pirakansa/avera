"""Optional browser regression for literal destination hints and shared service judgments."""
import pathlib,tempfile,re,signal,subprocess,json
from playwright.sync_api import sync_playwright
out=pathlib.Path(tempfile.mkdtemp(prefix='avera-forwarding-trial-'))
snapshot=json.loads(pathlib.Path('test/fixtures/prototype/contracts/snapshot-first.json').read_text())
snapshot['contract_revision']='netfilter-2026-10-25.1'
rule='-A PREROUTING -i eth0 -p tcp --dport 8443 -j DNAT --to-destination 127.0.0.1:9000'
context='*nat\n:PREROUTING ACCEPT [0:0]\n'+rule+'\nCOMMIT'
def digest(value):
 import hashlib
 return 'sha256:'+hashlib.sha256(json.dumps(value,ensure_ascii=False,sort_keys=True,separators=(',',':')).encode()).hexdigest()
import hashlib
facts={'rule':rule,'context_digest':'sha256:'+hashlib.sha256(context.encode()).hexdigest(),'scope_id':next(c['scope'] for c in snapshot['coverage'] if c['collector']=='network')}
identity={'source_path':'/netfilter/iptables-save','line':1,'ordinal':0}
finding={'collector':'network','kind':'network.forwarding','identity':identity,'facts':facts,'evidence_ids':['forwarding-iptables-save']}
finding['finding_id']=digest(['avera.identity.v1',snapshot['target_id'],'network','network.forwarding',identity])
finding['fact_fingerprint']=digest(['avera.facts.v1','network.forwarding',identity,facts])
snapshot['findings'].append(finding)
snapshot['evidence'].append({'evidence_id':'forwarding-iptables-save','source_kind':'command_output','location':'command:iptables-save','collected_at':snapshot['collected_at'],'method':'Fictional bounded ruleset','values':{'excerpt':context}})
(out/'snapshot.json').write_text(json.dumps(snapshot))
subprocess.run(['./bin/avera','review','--single-report','--snapshot',str(out/'snapshot.json'),'--output',str(out/'report.json')],check=True,capture_output=True)
server=subprocess.Popen(['./bin/avera','serve','--input',str(out/'report.json'),'--output-dir',str(out)],stderr=subprocess.PIPE,text=True)
try:
 url=re.search(r'http://[0-9.:]+',server.stderr.readline()).group()
 with sync_playwright() as p:
  browser=p.chromium.launch(headless=True,args=['--no-sandbox']);page=browser.new_page(viewport={'width':1280,'height':960});errors=[];page.on('pageerror',lambda e:errors.append(str(e)));page.goto(url);page.locator('#items > details').first.wait_for()
  assert page.locator('#items > details').count()==4
  assert page.locator('#reference-items > details').count()==1 and page.locator('#reference-items form').count()==0
  state=page.request.get(url+'/api/state').json();assert state['Summary']['current']==4 and state['Summary']['unreviewed']==4 and state['ReferenceCount']==1
  http=page.locator('#items > details').filter(has=page.locator(':scope > summary',has_text='転送設定 '));http.locator(':scope > summary').click()
  assert '127.0.0.1:9000' in http.inner_text()
  http.locator('[name=purpose]').fill('Public entrance for internal service');http.locator('[name=rationale]').fill('Reviewed public entry and forwarding policy');http.locator('[name=actor]').fill('fictional-owner');http.locator('select').select_option('accepted')
  assert http.locator('[name=related_ids]').count()==0
  assert '設定に記載された転送先と一致するサービス' in http.inner_text()
  http.locator('button[type=submit]').click();page.locator('#status').filter(has_text='保存しました').wait_for();page.wait_for_timeout(200)
  state=page.request.get(url+'/api/state').json();assert len(state['Routes'] or [])==0 and state['Summary']['accepted']==1
  assert len(next(item for item in state['Items'] if item['finding']['kind']=='network.forwarding')['detected_targets'])==1
  assert page.evaluate('window.injected') is None
  body=page.request.get(url+'/download.json').json();assert body['format']=='avera.report.v3' and len(body.get('routes',[]))==0
  preview=browser.new_page();preview.goto(url+'/preview');assert preview.locator('#local-reference').count()==1
  assert '転送設定' in preview.inner_text('body') and '127.0.0.1:9000' in preview.inner_text('body')
  assert '設定に記載された転送先との一致' in preview.inner_text('body') and '<script>' not in preview.content()
  web=page.locator('#items > details').filter(has=page.locator(':scope > summary',has_text=':80 '));web.locator(':scope > summary').click()
  assert web.locator('[name=reuse]').count()==0
  assert '保存した説明と判断は' in web.inner_text()
  web.locator('[name=actor]').fill('fictional-owner');web.locator('[name=rationale]').fill('Static public web service; upstream load balancer controls apply');web.locator('select').select_option('accepted');web.locator('button[type=submit]').click();page.locator('#status').filter(has_text='保存しました').wait_for();page.wait_for_timeout(200)
  body=page.request.get(url+'/download.json').json();assert len(body['policies'])==1 and body['policies'][0]['port']==80
  assert '共通設定の判断を適用しています' in page.inner_text('body')
  preview.reload();assert '共通設定' in preview.inner_text('body')
  preview.screenshot(path=str(out/'report.png'),full_page=True)
  page.screenshot(path=str(out/'editor.png'),full_page=True)
  page.set_viewport_size({'width':390,'height':844});assert page.evaluate('document.documentElement.scrollWidth <= innerWidth')
  assert not errors,errors
  browser.close()
 print('PASS: automatic literal destination hint, automatic shared policy save, v3 JSON, shared HTML explanation and mobile layout')
 print('Trial artifacts:',out)
finally:
 server.send_signal(signal.SIGTERM);server.wait(timeout=10);assert server.returncode==0
