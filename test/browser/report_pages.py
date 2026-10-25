"""Offline HTML page navigation and print regression using a fictional report."""
import pathlib, signal, subprocess, tempfile, re
from playwright.sync_api import sync_playwright

out = pathlib.Path(tempfile.mkdtemp(prefix='avera-report-pages-'))
server = subprocess.Popen(['./bin/avera', 'serve', '--input', 'test/fixtures/prototype/reports/fictional-report.json', '--output-dir', str(out)], stderr=subprocess.PIPE, text=True)
try:
    url = re.search(r'http://[0-9.:]+', server.stderr.readline()).group()
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True, args=['--no-sandbox'])
        page = browser.new_page(viewport={'width': 1280, 'height': 900}, java_script_enabled=False)
        response = page.request.get(url + '/preview')
        assert response.ok
        artifact = out / 'report.html'
        artifact.write_text(response.text())
        page.goto(artifact.as_uri())
        summary = page.locator('#report-summary')
        configuration = page.locator('#report-configuration')
        details = page.locator('#report-details')
        assert summary.is_visible() and not configuration.is_visible() and not details.is_visible()
        assert '収集範囲・確認できていない事項' in summary.inner_text()
        page.screenshot(path=str(out / 'summary.png'), full_page=True)
        tabs = page.locator('.report-tabs')
        tabs.locator('a[href="#report-configuration"]').click()
        assert configuration.is_visible() and not summary.is_visible() and not details.is_visible()
        page.screenshot(path=str(out / 'configuration.png'), full_page=True)
        link = configuration.locator('.map-card a').first
        destination = link.get_attribute('href')
        link.click()
        assert details.is_visible() and page.locator(destination).is_visible() and not configuration.is_visible()
        page.go_back()
        assert configuration.is_visible() and not details.is_visible()
        tabs.locator('a[href="#report-details"]').focus()
        page.keyboard.press('Enter')
        assert details.is_visible()
        page.screenshot(path=str(out / 'details.png'), full_page=True)
        page.emulate_media(media='print')
        assert summary.is_visible() and configuration.is_visible() and details.is_visible()
        assert not tabs.is_visible()
        page.pdf(path=str(out / 'report.pdf'))
        page.emulate_media(media='screen')
        page.set_viewport_size({'width': 390, 'height': 844})
        for target in ['report-summary', 'report-configuration', 'report-details']:
            tabs.locator('a[href="#' + target + '"]').click()
            assert page.locator('#' + target).is_visible()
            assert page.evaluate('document.documentElement.scrollWidth <= innerWidth')
        tabs.locator('a[href="#report-summary"]').click()
        page.screenshot(path=str(out / 'mobile.png'), full_page=True)
        browser.close()
    print('PASS: offline script-free summary, page navigation, diagram links, back, keyboard, print and narrow layouts')
    print('Browser artifacts:', out)
finally:
    server.send_signal(signal.SIGTERM)
    server.wait(timeout=10)
    assert server.returncode == 0
