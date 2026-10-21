#!/bin/sh
# Build the local Hyrule module selected by go.work, then run against this repo.
set -eu
hyrule_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
mkdir -p "$hyrule_root/.build"
cd "$hyrule_root"
GOWORK="$hyrule_root/go.work" "${GO:-go}" build -o "$hyrule_root/.build/hyrule" github.com/kojah/lockdoc/tools/hyrule/cmd/hyrule
exec "$hyrule_root/.build/hyrule" "$@"
