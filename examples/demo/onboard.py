"""Onboard a fresh Home Assistant without a browser and print a long-lived access token.

Runs inside the Home Assistant container, which has aiohttp. The steps are the same as in
internal/acctest/onboard.go:

    docker exec -i ha-demo python3 - < onboard.py
"""

import asyncio

import aiohttp

BASE = "http://localhost:8123"
CLIENT_ID = BASE + "/"


async def post(session, path, body, token=None):
    headers = {"Authorization": f"Bearer {token}"} if token else {}
    async with session.post(BASE + path, json=body, headers=headers) as resp:
        if resp.status != 200:
            raise SystemExit(f"POST {path}: HTTP {resp.status}: {await resp.text()}")
        return await resp.json()


async def main():
    async with aiohttp.ClientSession() as session:
        user = await post(session, "/api/onboarding/users", {
            "client_id": CLIENT_ID,
            "name": "Demo",
            "username": "demo",
            "password": "demo-password",
            "language": "en",
        })
        async with session.post(BASE + "/auth/token", data={
            "grant_type": "authorization_code",
            "code": user["auth_code"],
            "client_id": CLIENT_ID,
        }) as resp:
            access_token = (await resp.json())["access_token"]

        await post(session, "/api/onboarding/core_config", {}, access_token)
        await post(session, "/api/onboarding/analytics", {}, access_token)
        await post(session, "/api/onboarding/integration",
                   {"client_id": CLIENT_ID, "redirect_uri": CLIENT_ID}, access_token)

        async with session.ws_connect(BASE + "/api/websocket") as ws:
            await ws.receive_json()  # auth_required
            await ws.send_json({"type": "auth", "access_token": access_token})
            msg = await ws.receive_json()
            if msg["type"] != "auth_ok":
                raise SystemExit(f"authentication failed: {msg}")
            await ws.send_json({
                "id": 1,
                "type": "auth/long_lived_access_token",
                "client_name": "tofu demo",
                "lifespan": 365,
            })
            msg = await ws.receive_json()
            if not msg["success"]:
                raise SystemExit(f"long-lived token: {msg}")
            print(msg["result"])


asyncio.run(main())
