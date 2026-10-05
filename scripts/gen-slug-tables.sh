#!/usr/bin/env bash
# Regenerate internal/client/slug_tables.go from the slugify dependencies of a Home Assistant
# image, so that client.Slugify keeps matching HA. Needs Docker.
# Usage: scripts/gen-slug-tables.sh [image-tag]   (default: acctest.DefaultImageTag)
set -euo pipefail

cd "$(git -C "$(dirname "$0")" rev-parse --show-toplevel)"
tag="${1:-$(sed -n 's/^[[:space:]]*DefaultImageTag = "\(.*\)"/\1/p' internal/acctest/acctest.go)}"

docker run --rm --entrypoint python "ghcr.io/home-assistant/home-assistant:${tag}" -c '
import json
from html.entities import name2codepoint
from text_unidecode import unidecode
print(json.dumps({
    "entities": name2codepoint,
    "translit": [unidecode(chr(cp)) for cp in range(0x10000)],
}))
' | go run ./scripts/genslug internal/client/slug_tables.go
