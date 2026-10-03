# 4. REST plus one lazily opened WebSocket; each operation uses whatever HA offers

Date: 2026-10-03 · Status: Accepted

## Context

The original spec assigned transports per phase, and it was wrong in places. Checked against
home-assistant/core `dev`:

- Config flows (init, step, abort), reconfigure flows, options flows, and config entry deletion
  are **REST only** (`/api/config/config_entries/...`). There is no WebSocket
  `config_entries/delete`.
- Automation, script, and scene config is REST (`/api/config/<domain>/config/<id>`). A write
  triggers a reload asynchronously.
- Input helpers are **WebSocket only** (`input_boolean/create|update|delete|list`, …). There is no
  REST config endpoint for them.
- The area, floor, label, entity, and device registries, Lovelace, and `config_entries/get` are
  WebSocket.

## Decision

- One `HAClient` holds a REST client and a WebSocket client. Each operation uses the transport HA
  exposes for it.
- The WebSocket connection is opened **lazily** on the first command, authenticated, and shared by
  all resources of one provider instance. Responses are matched to commands by `id`.
- On connection loss, every in-flight command fails with a retryable error. The next command
  dials again. There is no background reconnect loop.

## Consequences

- Plans that only use REST data sources never open a socket.
- Most resource families use both transports (e.g. an automation writes over REST and reads its
  entity over REST state). The client package is organised by HA concept, not by transport.
