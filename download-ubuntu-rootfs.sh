#!/bin/bash
set -e

echo "Downloading Ubuntu 24.04 amd64 rootfs using Docker..."

echo "Pulling Ubuntu 24.04 amd64 image..."
docker pull --platform linux/amd64 ubuntu:24.04

echo "Running container to install packages..."
CONTAINER_ID=$(docker run -d --platform linux/amd64 ubuntu:24.04 tail -f /dev/null)

echo "Installing sudo in container..."
docker exec "$CONTAINER_ID" apt-get update
docker exec "$CONTAINER_ID" apt-get install -y sudo

echo "Stopping container..."
docker stop "$CONTAINER_ID"

echo "Exporting container filesystem..."
docker export "$CONTAINER_ID" | gzip > ubuntu-24.04-rootfs.tar.gz

echo "Cleaning up container..."
docker rm "$CONTAINER_ID"

echo "Done! Created ubuntu-24.04-rootfs.tar.gz"
echo "Size: $(ls -lh ubuntu-24.04-rootfs.tar.gz | awk '{print $5}')"
