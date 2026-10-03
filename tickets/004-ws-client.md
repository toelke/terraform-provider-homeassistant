---
status: done
depends_on: [001]
adrs: [0004, 0011]
spec: [overview.md]
---

# WebSocket client: lazy dial, auth, multiplexing

## Scope
- Dial `/api/websocket` lazily on the first command and perform the auth handshake.
- Multiplex commands by incrementing `id`, with one reader goroutine dispatching to waiters.
- Respect context cancellation and timeouts.
- On connection loss, fail all pending commands with a retryable error; the next command dials
  again.
- Map `success: false` to a typed error carrying `code` and `message`.
- Expose it through an interface, so resources can be tested against fakes.

## Acceptance criteria
- [x] Unit tests against an in-process fake WS server cover: auth ok, auth invalid, concurrent
      commands answered out of order, drop followed by redial, and context cancel.
- [x] Plans that use only REST data sources open no socket (unit test on the client).
