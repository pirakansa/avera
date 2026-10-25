"""Optional browser regression for reference grouping and human-described paths."""
import pathlib,tempfile,re,signal,subprocess,json
from playwright.sync_api import sync_playwright
out=pathlib.Path(tempfile.mkdtemp(prefix='avera-route-trial-'))
subprocess.run(['./bin/avera','review','--single-report','--snapshot','test/fixtures/prototype/contracts/snapshot-first.json','--output',str(out/'report.json')],check=True,capture_output=True)
server=subprocess.Popen(['./bin/avera','serve','--input',str(out/'report.json'),'--output-dir',str(out)],stderr=subprocess.PIPE,text=True)
try:
 url=re.search(r'http://[0-9.:]+',server.stderr.readline()).group()
 with sync_playwright() as p:
  browser=p.chromium.launch(headless=True,args=['--no-sandbox']);page=browser.new_page(viewport={'width':1280,'height':960});errors=[];page.on('pageerror',lambda e:errors.append(str(e)));page.goto(url);page.locator('#items > details').first.wait_for()
  assert page.locator('#items > details').count()==3
  assert page.locator('#reference-items > details').count()==1 and page.locator('#reference-items form').count()==0
  state=page.request.get(url+'/api/state').json();assert state['Summary']['current']==3 and state['Summary']['unreviewed']==3 and state['ReferenceCount']==1
  http=page.locator('#items > details').filter(has=page.locator(':scope > summary',has_text=':80 '));http.locator(':scope > summary').click()
  http.locator('[name=purpose]').fill('Public entrance for internal service');http.locator('[name=rationale]').fill('Reviewed public entry and forwarding policy');http.locator('[name=actor]').fill('fictional-owner');http.locator('select').select_option('accepted')
  http.locator('[name=related_ids]').check();http.locator('[name=route_explanation]').fill('Proxy forwards to local admin; firewall constraints are human statements <script>window.injected=1</script>')
  http.locator('button[type=submit]').click();page.locator('#status').filter(has_text='保存しました').wait_for();page.wait_for_timeout(200)
  state=page.request.get(url+'/api/state').json();assert len(state['Routes'])==1 and state['Summary']['accepted']==1
  assert page.evaluate('window.injected') is None
  body=page.request.get(url+'/download.json').json();assert body['format']=='avera.report.v2' and len(body['routes'])==1
  preview=browser.new_page();preview.goto(url+'/preview');assert preview.locator('#local-reference').count()==1 and preview.locator('#routes').count()==1
  assert '転送経路は未検証' in preview.inner_text('body') and '<script>' not in preview.content()
  preview.screenshot(path=str(out/'report.png'),full_page=True)
  page.screenshot(path=str(out/'editor.png'),full_page=True)
  page.set_viewport_size({'width':390,'height':844});assert page.evaluate('document.documentElement.scrollWidth <= innerWidth')
  assert not errors,errors
  browser.close()
 print('PASS: reference counts/read-only UI, entrance route save, single JSON roundtrip, escaped linked HTML, and mobile layout')
 print('Trial artifacts:',out)
finally:
 server.send_signal(signal.SIGTERM);server.wait(timeout=10);assert server.returncode==0
