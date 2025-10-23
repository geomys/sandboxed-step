#!/bin/bash
set -e

echo "Downloading latest runsc binary..."
curl -fsSL https://storage.googleapis.com/gvisor/releases/release/latest/x86_64/runsc -o runsc
chmod +x runsc
