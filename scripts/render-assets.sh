#!/bin/sh
# Renders the PNG assets from their SVG/HTML sources with headless Chrome — no
# dependency, no CI step. Set CHROME to the browser binary; the default is the
# Windows Chrome seen from WSL.
set -eu
CHROME="${CHROME:-/mnt/c/Program Files/Google/Chrome/Application/chrome.exe}"
cd "$(dirname "$0")/.."
url() { if command -v wslpath >/dev/null 2>&1; then echo "file:///$(wslpath -w "$PWD/$1" | sed 's#\\#/#g')"; else echo "file://$PWD/$1"; fi; }
out() { if command -v wslpath >/dev/null 2>&1; then wslpath -w "$PWD/$1"; else echo "$PWD/$1"; fi; }
shot() { # <source> <png> <width> <height>
  "$CHROME" --headless=new --disable-gpu --hide-scrollbars --allow-file-access-from-files \
    --default-background-color=00000000 --window-size="$3,$4" --screenshot="$(out "$2")" "$(url "$1")" 2>/dev/null
}
tmp=scripts/.icon.html
for s in 512:logo.png 180:apple-touch-icon.png; do
  n=${s%%:*}; f=${s#*:}
  printf '<html><body style="margin:0;background:transparent"><img src="../docs/assets/logo.svg" width="%s" height="%s"></body></html>' "$n" "$n" > "$tmp"
  shot "$tmp" "docs/assets/$f" "$n" "$n"
done
rm -f "$tmp"
shot scripts/og-card.html docs/assets/og-card.png 1200 630
ls -la docs/assets/*.png
