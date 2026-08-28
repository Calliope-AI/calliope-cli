#!/usr/bin/env bash
# Instalador de calliope. Uso:
#   curl -fsSL https://raw.githubusercontent.com/Calliope-AI/calliope-cli/main/install.sh | bash
set -euo pipefail

REPO="Calliope-AI/calliope-cli"
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

# El nombre está acoplado al name_template de archives en .goreleaser.yaml
# ("{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}": sistema en
# minúsculas, arquitectura literal, versión sin «v»). Tocar uno obliga a
# tocar el otro.
archivo="calliope_${version#v}_${so}_${arco}.tar.gz"
url="https://github.com/${REPO}/releases/download/${version}/${archivo}"

echo "Descargando calliope ${version} (${so}/${arco})…"
curl -fsSL "$url" -o "$tmp/$archivo"
curl -fsSL "https://github.com/${REPO}/releases/download/${version}/checksums.txt" -o "$tmp/checksums.txt"

# sha256sum es lo habitual en Linux; en macOS solo está shasum. Se falla con
# un mensaje claro si no hay ninguno, en vez de dejar que `command not found`
# lo explique peor.
if command -v sha256sum >/dev/null 2>&1; then
  verificador=(sha256sum -c -)
elif command -v shasum >/dev/null 2>&1; then
  verificador=(shasum -a 256 -c -)
else
  echo "No se encontró sha256sum ni shasum: no se puede verificar el checksum." >&2
  exit 1
fi

# Se verifica el checksum antes de instalar nada. -F porque el nombre del
# archivo lleva puntos (versión y extensión) que un grep sin -F trataría
# como «cualquier carácter», no como puntos literales.
(cd "$tmp" && grep -F -- " ${archivo}" checksums.txt | "${verificador[@]}")

tar -xzf "$tmp/$archivo" -C "$tmp"
install -m 0755 "$tmp/calliope" "$DESTINO/calliope"

echo "calliope instalado en $DESTINO/calliope"
"$DESTINO/calliope" version
