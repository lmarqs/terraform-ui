#!/usr/bin/env bash
set -euo pipefail

REPO="lmarqs/terraform-ui"
INSTALL_DIR="${TFUI_INSTALL_DIR:-$HOME/.local/lib/terraform-ui}"

main() {
  local version="${1:-latest}"

  if [ "$version" = "latest" ]; then
    version=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)
  fi

  echo "Installing terraform-ui $version to $INSTALL_DIR..."

  mkdir -p "$INSTALL_DIR"
  curl -fsSL "https://github.com/$REPO/archive/refs/tags/$version.tar.gz" | tar -xz --strip-components=1 -C "$INSTALL_DIR"

  mkdir -p "$HOME/.local/bin"
  ln -sf "$INSTALL_DIR/bin/tfui" "$HOME/.local/bin/tfui"

  echo ""
  echo "Installed! Run directly:"
  echo "  tfui plan --dir ./my-module"
  echo ""
  echo "Or source as a library:"
  echo "  source \"$INSTALL_DIR/lib/tfui.sh\""
}

main "$@"
