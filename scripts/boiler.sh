#!/bin/sh
# Build the representation generator from the same workspace as Hyrule.
set -eu
boiler_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
mkdir -p "$boiler_root/.build"
cd "$boiler_root"
GOWORK="$boiler_root/go.work" "${GO:-go}" build -o "$boiler_root/.build/boiler" github.com/kojah/lockdoc/tools/boiler/cmd/boiler
exec "$boiler_root/.build/boiler" "$@"
