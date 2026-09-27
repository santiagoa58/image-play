#!/usr/bin/env bash
# Generate every image fixture using the Darth Vader text and mono font.
# Usage: scripts/test-wordcloud-images.sh [output-directory] [extra CLI flags...]
set -euo pipefail
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"
output_directory="${1:-testdata/out}"
if (($#)); then shift; fi
mkdir -p "$output_directory"
build_directory="$(mktemp -d)"
trap 'rm -rf "$build_directory"' EXIT
go build -o "$build_directory/mosaic" ./cmd/mosaic
for input in testdata/images/*; do
  [[ -f "$input" ]] || continue
  name="$(basename "$input")"
  case "$name" in
    darth_vader_og.jpg) stem=vader ;;
    deepseek-logo-icon.png) stem=deepseek_logo ;;
    *) stem="${name%.*}" ;;
  esac
  "$build_directory/mosaic" -effect wordcloud -in "$input" \
    -text testdata/text/darth_vader.txt \
    -font 'fonts/NotoSansMono-VariableFont_wdth,wght.ttf' \
    -out "$output_directory/$stem.png" "$@"
done
