#!/usr/bin/env bash
# Instalador de calliope. Uso:
#   curl -fsSL https://raw.githubusercontent.com/calliope/calliope-cli/main/install.sh | bash
set -euo pipefail

REPO="calliope/calliope-cli"
DESTINO="${CALLIOPE_INSTALL_DIR:-/usr/local/bin}"

so=$(uname -s | tr '[:upper:]' '[:lower:]')
arco=$(uname -m)
case "$arco" in
  x86_64) arco=amd64 ;;
  aarch64|arm64) arco=arm64 ;;
  *) echo "Arquitectura no soportada: $arco" >&2; exit 1 ;;
esac

version=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep '"tag_name"' | head -1 | cut -d'"' -f4)
if [ -z "$version" ]; then
  echo "No se pudo determinar la última versión." >&2
  exit 1
fi

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

archivo="calliope_${version#v}_${so}_${arco}.tar.gz"
url="https://github.com/${REPO}/releases/download/${version}/${archivo}"

echo "Descargando calliope ${version} (${so}/${arco})…"
curl -fsSL "$url" -o "$tmp/$archivo"
curl -fsSL "https://github.com/${REPO}/releases/download/${version}/checksums.txt" -o "$tmp/checksums.txt"

# Se verifica el checksum antes de instalar nada.
(cd "$tmp" && grep " ${archivo}\$" checksums.txt | shasum -a 256 -c -)

tar -xzf "$tmp/$archivo" -C "$tmp"
install -m 0755 "$tmp/calliope" "$DESTINO/calliope"

echo "calliope instalado en $DESTINO/calliope"
"$DESTINO/calliope" version
