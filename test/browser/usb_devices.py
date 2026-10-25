"""USB HID inventory, composite judgments and offline physical topology."""
import json, pathlib, re, signal, subprocess, tempfile
from playwright.sync_api import sync_playwright, expect

out=pathlib.Path(tempfile.mkdtemp(prefix='avera-usb-browser-'))
source=pathlib.Path('test/fixtures/prototype/reports/usb-demo.json').read_bytes()
(out/'input.json').write_bytes(source)
server=subprocess.Popen(['./bin/avera','serve','--input',str(out/'input.json'),'--output-dir',str(out)],stderr=subprocess.PIPE,text=True)
try:
 url=re.search(r'http://[0-9.:]+',server.stderr.readline()).group()
 with sync_playwright() as p:
  browser=p.chromium.launch(headless=True,args=['--no-sandbox'])
  page=browser.new_page(viewport={'width':1280,'height':960});errors=[];page.on('pageerror',lambda error:errors.append(str(error)))
  page.goto(url);page.locator('#physical-map .physical-tree').first.wait_for()
  assert 'Fictional desk hub' in page.locator('#physical-map').inner_text()
  assert 'DEMO-OPS-USB' in page.locator('#physical-map').inner_text()
  assert page.locator('#physical-map .physical-tree .physical-tree').count()>=2
  main=page.locator('#items > details').filter(has=page.locator(':scope > summary',has_text='USB HID機器'))
  assert main.count()==2
  grouped=page.locator('#usb-storage-reference-items > details');assert grouped.count()==1
  assert grouped.locator('form').count()==0
  composite=main.filter(has=page.locator(':scope > summary',has_text='Fictional input + transfer'))
  fid=composite.get_attribute('id')
  page.locator('#physical-map a[href="#'+fid+'"]').click();assert composite.get_attribute('open') is not None
  assert '同じ機器にまとめたストレージ観測' in composite.inner_text()
  assert composite.locator('[name=exposure]').count()==0
  before=page.request.get(url+'/download.json').json()['current']
  composite.locator('[name=purpose]').fill('○○アプリの入力と保存先。運用担当が管理')
  composite.locator('[name=safeguards]').fill('上位提供の運用マニュアルで接続担当者と物理管理を定義済み')
  composite.locator('[name=rationale]').fill('機器全体の用途と保護策を確認して継続利用を判断')
  composite.locator('[name=actor]').fill('fictional-reviewer')
  composite.locator('select').select_option('accepted')
  composite.locator('button[type=submit]').click();page.locator('#status').filter(has_text='保存しました').wait_for()
  expect(composite.locator(':scope > summary')).to_contain_text('追加対応不要')
  after=page.request.get(url+'/download.json').json();assert after['current']==before
  assert (out/'input.json').read_bytes()==source
  assert '上位提供の運用マニュアル' in page.locator('#physical-map').inner_text()
  exported=page.request.get(url+'/export.html').text();(out/'report.html').write_text(exported)
  offline=browser.new_page(java_script_enabled=False);offline.goto((out/'report.html').as_uri())
  offline.locator('.report-tabs a[href="#report-configuration"]').click()
  assert offline.locator('#physical-map').is_visible()
  assert offline.locator('#network-map').is_visible()
  assert '上位提供の運用マニュアル' in offline.locator('#physical-map').inner_text()
  link=offline.locator('#physical-map a').filter(has_text='Fictional input + transfer');link.click()
  assert offline.locator('#report-details').is_visible()
  assert offline.locator(offline.evaluate('location.hash')).is_visible()
  for href in offline.locator('a[href^="#"]').evaluate_all('(nodes)=>nodes.map(n=>n.getAttribute("href").slice(1))'):
   assert offline.locator('[id="'+href+'"]').count()==1,href
  for target in [page,offline]:
   target.set_viewport_size({'width':390,'height':844});assert target.evaluate('document.documentElement.scrollWidth <= innerWidth')
  offline.emulate_media(media='print');assert offline.locator('#physical-map').is_visible()
  page.screenshot(path=str(out/'editor.png'),full_page=True)
  offline.screenshot(path=str(out/'report.png'),full_page=True)
  assert not errors,errors
  browser.close()
 print('PASS: USB physical tree, manual safeguards, composite single judgment, immutable save, offline navigation and narrow/print layouts')
 print('Browser artifacts:',out)
finally:
 server.send_signal(signal.SIGTERM);server.wait(timeout=10);assert server.returncode==0
