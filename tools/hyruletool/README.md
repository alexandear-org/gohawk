# Hyrule compiler pin

This separate tool module pins the compiler without raising the consumer's Go
version. `scripts/hyrule.sh` builds the sibling checkout through `tools/hyrule`
when present, or this pinned dependency on clean clones. Set `HYRULE_PINNED=1`
to exercise the pinned version even with a local checkout.

For local development, clone `kojah/hyrule` beside this repository and add
`../hyrule` to the local `go.work`. The `tools/hyrule` symlink points there.
Keep workspace files local. Update the pin with `GOWORK=off go get -tool github.com/kojah/hyrule/cmd/hyrule@VERSION` from this directory, then verify
consumer generation. Compiler tests and examples belong in the Hyrule repository.
