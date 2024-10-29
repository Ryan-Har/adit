#!/bin/bash

set -e

# Identify platform
os=$(uname | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)

case "$arch" in
  x86_64) arch="amd64" ;;
  aarch64) arch="arm64" ;;
esac

# Download binary
release_url="https://github.com/Ryan-Har/adit/releases/latest/download/client-${os}-${arch}"
curl -L -o /usr/local/bin/adit "$release_url"
chmod +x /usr/local/bin/adit

echo "Adit installed successfully!"
