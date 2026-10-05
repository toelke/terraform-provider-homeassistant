#!/usr/bin/env bash
# Remove the demo container, and with it all of Home Assistant's state.
set -euo pipefail
docker rm -f ha-demo >/dev/null
