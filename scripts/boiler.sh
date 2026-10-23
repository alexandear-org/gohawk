#!/bin/sh
# Build a local Boiler checkout when present; otherwise use the pinned tool module.
# Run from the consumer root so the default boiler.yaml resolves there.
set -eu
boiler_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
mkdir -p "$boiler_root/.build"
if [ -f "$boiler_root/tools/boiler/go.mod" ] && [ "${BOILER_PINNED:-0}" != 1 ]; then
    boiler_source=$(CDPATH= cd -P -- "$boiler_root/tools/boiler" && pwd)
else
    boiler_source="$boiler_root/tools/boilertool"
fi
GOWORK=off "${GO:-go}" -C "$boiler_source" build -o "$boiler_root/.build/boiler" github.com/kojah/boiler/cmd/boiler
cd "$boiler_root"
exec "$boiler_root/.build/boiler" "$@"
