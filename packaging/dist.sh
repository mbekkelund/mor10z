#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
version=0.1.0
stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT
mkdir -p "$stage/mor10z-$version" dist
cp -R cmd internal packaging go.mod go.sum Makefile README.md "$stage/mor10z-$version/"
if [[ -f LICENSE ]]; then cp LICENSE "$stage/mor10z-$version/"; fi
go_command=$(command -v "${GO:-go}")
go_command=$(realpath "$go_command")
(
  cd "$stage/mor10z-$version"
  "$go_command" mod vendor
)
tar -czf "dist/mor10z-$version.tar.gz" -C "$stage" "mor10z-$version"
cp packaging/PKGBUILD dist/PKGBUILD
checksum=$(sha256sum "dist/mor10z-$version.tar.gz" | cut -d ' ' -f 1)
sed -i "s/@SHA256@/$checksum/" dist/PKGBUILD
printf 'Source archive and local PKGBUILD written to dist/\n'
