#!/bin/sh
set -eu
# Linux runner helper for the GitLab example.
version=1.3.0
case "$(uname -m)" in
    x86_64) arch=amd64 ;;
    aarch64|arm64) arch=arm64 ;;
    *) echo 'Unsupported ORAS runner architecture' >&2; exit 1 ;;
esac
destination=$(realpath "$1")
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
cd "$tmp"
file="oras_${version}_linux_${arch}.tar.gz"
base="https://github.com/oras-project/oras/releases/download/v${version}"
curl -fsSL "$base/$file" -o "$file"
curl -fsSL "$base/oras_${version}_checksums.txt" -o checksums.txt
awk -v file="$file" '$2 == file {print}' checksums.txt > selected-checksum.txt
test "$(wc -l < selected-checksum.txt)" -eq 1
sha256sum -c selected-checksum.txt
mkdir -p "$destination"
tar -xzf "$file" -C "$destination" oras
