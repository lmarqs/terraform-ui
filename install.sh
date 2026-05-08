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

  echo ""
  echo "Installed! Add this to your script:"
  echo "  source \"$INSTALL_DIR/lib/tfui.sh\""
}

main "$@"
