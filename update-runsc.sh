#!/bin/bash
set -e

echo "Downloading latest runsc binary..."
curl -fsSL https://storage.googleapis.com/gvisor/releases/release/latest/x86_64/runsc -o runsc
curl -fsSL https://storage.googleapis.com/gvisor/releases/release/latest/x86_64/runsc.sha512 -o runsc.sha512
sha512sum --check runsc.sha512
rm runsc.sha512

# GitHub rejects files over 100 MiB. Current upstream binaries include enough
# debug information to exceed that limit, so strip the vendored copy.
strip --strip-all runsc
chmod +x runsc
