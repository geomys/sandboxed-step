#!/bin/bash
set -e

echo "Building generate-config for Linux amd64..."

# Build for Linux amd64
GOOS=linux GOARCH=amd64 go build -o generate-config generate-config.go

chmod +x generate-config

echo "Built generate-config binary"
echo "Size: $(ls -lh generate-config | awk '{print $5}')"
