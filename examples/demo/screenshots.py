"""Take the README's screenshots of the demo dashboards.

Runs in the Playwright container against the HA that ./up.sh started and the example configured:

    docker run --rm --network host --user "$(id -u)" -e HOME=/tmp -v "$PWD/../..:/repo" -w /repo/examples/demo \
      -e HOMEASSISTANT_URL -e HOMEASSISTANT_TOKEN \
      mcr.microsoft.com/playwright/python:v1.55.0-noble \
      sh -c 'pip install -q playwright==1.55.0 && python screenshots.py ../../docs/images/'
"""

import json
import os
import sys
import time

from playwright.sync_api import sync_playwright

URL = os.environ["HOMEASSISTANT_URL"]
TOKEN = os.environ["HOMEASSISTANT_TOKEN"]
OUT = sys.argv[1]

PAGES = {
    "dashboard-living-room.png": ("/dashboard-home/living-room", (1280, 800)),
    "wall-tablet.png": ("/wall-tablet/0", (1024, 700)),
}

# The frontend keeps its tokens in localStorage; a long-lived token works there as well.
tokens = {
    "access_token": TOKEN,
    "token_type": "Bearer",
    "expires_in": 315360000,
    "expires": int(time.time() * 1000) + 315360000000,
    "refresh_token": "",
    "hassUrl": URL,
    "clientId": URL + "/",
}

os.makedirs(OUT, exist_ok=True)
with sync_playwright() as p:
    browser = p.chromium.launch()
    for name, (path, (width, height)) in PAGES.items():
        page = browser.new_page(viewport={"width": width, "height": height}, device_scale_factor=1)
        page.add_init_script(f"localStorage.setItem('hassTokens', {json.dumps(json.dumps(tokens))})")
        page.goto(URL + path)
        page.wait_for_load_state("networkidle")
        time.sleep(3)  # let the cards render their state
        page.screenshot(path=os.path.join(OUT, name))
        page.close()
    browser.close()
