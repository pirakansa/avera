"""Optional browser smoke test against a running first-snapshot fixture session."""

import asyncio,json,pathlib,sys,tempfile
from playwright.async_api import async_playwright

async def main():
 url=sys.argv[1]
 artifact_dir=pathlib.Path(tempfile.mkdtemp(prefix="avera-viewer-browser-"))
 async with async_playwright() as p:
  browser=await p.chromium.launch(headless=True,args=['--no-sandbox'])
  page=await browser.new_page(viewport={'width':1280,'height':900})
  errors=[];page.on('pageerror',lambda e:errors.append(str(e)))
  await page.goto(url)
  await page.locator('#items > details').first.wait_for()
  # Edit two items before saving either: the second draft must survive a save/reload.
  cards=page.locator('#items > details')
  http=None;other=None
  for j in range(await cards.count()):
   card=cards.nth(j);title=await card.locator(':scope > summary').inner_text()
   if ':80 ' in title:http=card
   if title.startswith('sudo '):other=card
  assert http is not None and other is not None
  assert await page.locator('#reference-items form').count()==0
  assert await page.locator('#reference-count').inner_text()=='1'
  other_id=await other.get_attribute('data-finding')
  await http.locator(':scope > summary').click()
  await other.locator(':scope > summary').click()
  await other.locator('[name=purpose]').fill('未保存の別項目の入力')
  await http.locator('[name=purpose]').fill('公開する日本語ドキュメント <script>window.injected=1</script>')
  await http.locator('[name=rationale]').fill('公開の目的と設定を確認しました')
  await http.locator('[name=actor]').fill('fictional-reviewer')
  await http.locator('select').select_option('accepted')
  await http.locator('button[type=submit]').click()
  await page.locator('#status').filter(has_text='保存しました').wait_for()
  await page.wait_for_timeout(250)
  # New DOM nodes replace cards, but values must remain.
  other=page.locator('#items > details[data-finding="'+other_id+'"]')
  assert await other.locator('[name=purpose]').input_value()=='未保存の別項目の入力'
  assert await page.evaluate('window.injected') is None
  state=await (await page.request.get(url+'api/state')).json()
  assert state['revision']==1
  active=[a for it in state['Items'] for a in it['judgments'] if a.get('actor')=='fictional-reviewer']
  assert len(active)==1 and active[0]['decision']=='accepted'
  # Downloads warn about remaining unsaved draft; accept the saved-state export.
  page.on('dialog',lambda dialog:dialog.accept())
  async with page.expect_download() as info:await page.locator('#json-download').click()
  download=await info.value
  path=await download.path();report=json.loads(pathlib.Path(path).read_text());assert report['format']=='avera.report.v3'
  async with page.expect_download() as info:await page.locator('#html-export').click()
  path=await (await info.value).path();html=pathlib.Path(path).read_text();assert '確認レポート' in html and '&lt;script&gt;' in html
  await page.locator('#preview-tab').click()
  await page.frame_locator('#preview').locator('h1').wait_for()
  assert '確認レポート' in await page.frame_locator('#preview').locator('h1').inner_text()
  await page.locator('#edit-tab').click()
  await page.set_viewport_size({'width':390,'height':844})
  assert await page.evaluate('document.documentElement.scrollWidth <= window.innerWidth')
  await page.screenshot(path=str(artifact_dir/'mobile.png'),full_page=True)
  await page.set_viewport_size({'width':1280,'height':900})
  await page.screenshot(path=str(artifact_dir/'desktop.png'),full_page=True)
  assert not errors,errors
  print('Browser artifacts:',artifact_dir)
  print('PASS: browser editing, draft preservation, save, JSON/HTML downloads, preview, escaping and mobile layout')
  await browser.close()
asyncio.run(main())
