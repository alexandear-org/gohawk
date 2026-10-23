#!/bin/sh
# Build a local Hyrule checkout when present; otherwise use the pinned tool module.
# Run from the consumer root so host package loading uses its workspace.
set -eu
hyrule_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
mkdir -p "$hyrule_root/.build"
if [ -f "$hyrule_root/tools/hyrule/go.mod" ] && [ "${HYRULE_PINNED:-0}" != 1 ]; then
    hyrule_source=$(CDPATH= cd -P -- "$hyrule_root/tools/hyrule" && pwd)
else
    hyrule_source="$hyrule_root/tools/hyruletool"
fi
GOWORK=off "${GO:-go}" -C "$hyrule_source" build -o "$hyrule_root/.build/hyrule" github.com/kojah/hyrule/cmd/hyrule
cd "$hyrule_root"
exec "$hyrule_root/.build/hyrule" "$@"
