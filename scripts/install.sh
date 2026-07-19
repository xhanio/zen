#!/usr/bin/env bash
#
# Install / update / uninstall the Zen all-in-one server.
#
# Zen ships as one self-contained image (docker.io/xhanio/zen-allinone:latest)
# that carries its own docker-compose.yaml at /app/deploy. This script pulls the
# image, extracts that compose into a directory of your choosing, and brings the
# stack up — no repo checkout required.
#
# Usage:
#   install.sh [DIR]              install to DIR            (default: $HOME/zen)
#   install.sh --update [DIR]     pull newest image + recreate, keeping data
#   install.sh --uninstall [DIR]  stop and remove the stack (DIR/data kept)
#   install.sh --no-plugin ...    skip the zen-channel plugin binary (server-only host)
#   install.sh --help
#
# It also installs the matching zen-channel plugin binary to ~/.local/bin
# (override with ZEN_BIN_DIR) so the Zen Claude Code plugin can spawn it — its
# .mcp.json declares `"command": "zen-channel"`, and the image carries a build
# for every host OS/arch. Pass --no-plugin on a box that only runs the server.
#
# The database lives in DIR/data (a bind mount), so it sits right beside the
# compose file — back it up by copying that folder. It survives update and
# uninstall; uninstall never deletes it (the container writes it as root, so
# erasing it may need sudo — the command is printed).
set -euo pipefail

IMAGE="${ZEN_IMAGE:-docker.io/xhanio/zen-allinone:latest}"   # override for a mirror/private registry
PROJECT="zen"                            # docker compose -p (names the project + network: zen_default)
COMPOSE_IN_IMAGE="/app/deploy/docker-compose.yaml"
PLUGIN_IN_IMAGE="/app/plugin"            # zen-channel_<os>_<arch> plugin binaries baked in the image
DEFAULT_DIR="$HOME/zen"
BINDIR="${ZEN_BIN_DIR:-$HOME/.local/bin}"   # where the zen-channel plugin binary goes (must be on PATH)

log()  { printf '\033[1m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m!\033[0m  %s\n' "$*" >&2; }
die()  { printf '\033[1;31mx\033[0m  %s\n' "$*" >&2; exit 1; }

# --- parse args: one optional mode flag, --no-plugin, one optional directory -
mode=install
want_plugin=1
dir=""
for arg in "$@"; do
  case "$arg" in
    --update)          mode=update ;;
    --uninstall)       mode=uninstall ;;
    --install)         mode=install ;;
    --no-plugin)       want_plugin=0 ;;
    -h|--help)         mode=help ;;
    -*)                die "unknown option: $arg (see --help)" ;;
    *)                 [ -n "$dir" ] && die "only one directory may be given"; dir="$arg" ;;
  esac
done
dir="${dir:-$DEFAULT_DIR}"
compose="$dir/docker-compose.yaml"

if [ "$mode" = help ]; then
  # print the header comment (lines after the shebang, up to `set -euo pipefail`)
  awk 'NR>=3{ if ($0=="set -euo pipefail") exit; sub(/^# ?/,""); print }' "$0"
  exit 0
fi

# --- prerequisites ----------------------------------------------------------
command -v docker >/dev/null 2>&1 || die "docker is not installed"
docker compose version >/dev/null 2>&1 || die "the 'docker compose' plugin is not available"
docker info >/dev/null 2>&1 || die "cannot talk to the Docker daemon (is it running? do you have permission?)"

compose_up() { docker compose -f "$compose" -p "$PROJECT" "$@"; }

# port the compose publishes, for the closing message (best-effort)
host_port() { grep -oE '[0-9]+:80"' "$compose" 2>/dev/null | head -1 | cut -d: -f1; }

# extract the baked compose out of the image into $compose
extract_compose() {
  log "Pulling $IMAGE"
  # Prefer the registry copy; fall back to a locally-present image (offline, or
  # a local test build) rather than failing.
  docker pull -q "$IMAGE" >/dev/null 2>&1 \
    || docker image inspect "$IMAGE" >/dev/null 2>&1 \
    || die "cannot pull $IMAGE and it is not present locally"
  local cid
  cid=$(docker create "$IMAGE")
  # trap-free cleanup: remove the scratch container even if cp fails
  if ! docker cp "$cid:$COMPOSE_IN_IMAGE" "$compose" 2>/dev/null; then
    docker rm -f "$cid" >/dev/null 2>&1 || true
    die "image has no $COMPOSE_IN_IMAGE — is it an older zen-allinone without a baked compose?"
  fi
  docker rm -f "$cid" >/dev/null 2>&1 || true
  # Pin the compose to the exact image we pulled, so a $ZEN_IMAGE override (a
  # mirror, a local build) is what actually runs — not the docker.io ref the
  # baked compose was rendered with. No-op when $IMAGE is the default. Portable
  # sed via a temp file (avoids GNU/BSD -i differences).
  local tmp; tmp=$(mktemp)
  sed 's|^\( *image: *\).*|\1'"$IMAGE"'|' "$compose" > "$tmp" && mv "$tmp" "$compose"
}

# baked plugin-binary filename for THIS host — Claude Code runs here, so the
# binary must match this machine's OS/arch, not the image's.
host_plugin() {
  local os arch
  case "$(uname -s)" in
    Darwin) os=darwin ;;
    Linux)  os=linux ;;
    *)      die "no zen-channel plugin binary for OS $(uname -s)" ;;
  esac
  case "$(uname -m)" in
    x86_64|amd64)  arch=amd64 ;;
    arm64|aarch64) arch=arm64 ;;
    *)             die "no zen-channel plugin binary for arch $(uname -m)" ;;
  esac
  printf 'zen-channel_%s_%s' "$os" "$arch"
}

# copy the matching plugin binary out of the image onto PATH so the plugin's
# `"command": "zen-channel"` resolves. Best-effort: warns, never fails install.
install_plugin() {
  local name cid; name=$(host_plugin)
  cid=$(docker create "$IMAGE")
  mkdir -p "$BINDIR"
  if docker cp "$cid:$PLUGIN_IN_IMAGE/$name" "$BINDIR/zen-channel" 2>/dev/null; then
    docker rm -f "$cid" >/dev/null 2>&1 || true
    chmod +x "$BINDIR/zen-channel"
    log "zen-channel plugin -> $BINDIR/zen-channel  ($name)"
    case ":$PATH:" in
      *":$BINDIR:"*) : ;;
      *) warn "$BINDIR is not on your PATH — Claude Code can't find zen-channel until you add it:"
         echo "        export PATH=\"$BINDIR:\$PATH\"   # add to ~/.bashrc or ~/.zshrc, then restart the shell" ;;
    esac
  else
    docker rm -f "$cid" >/dev/null 2>&1 || true
    warn "image has no $PLUGIN_IN_IMAGE/$name — skipping the plugin (older image without a baked plugin binary?)"
  fi
}

remove_plugin() {
  [ -f "$BINDIR/zen-channel" ] || return 0
  rm -f "$BINDIR/zen-channel" && log "Removed zen-channel from $BINDIR"
}

case "$mode" in
  install)
    mkdir -p "$dir/data"       # the compose bind-mounts ./data (relative to $dir)
    extract_compose
    log "Starting the stack (project '$PROJECT')"
    compose_up up -d
    port=$(host_port); port="${port:-38000}"
    log "Zen is up at http://localhost:${port}"
    echo "    compose file : $compose"
    echo "    data         : $dir/data (bind mount — back up by copying it)"
    echo "    update later : $0 --update ${dir/#$HOME/\$HOME}"
    [ "$want_plugin" = 1 ] && install_plugin
    ;;

  update)
    [ -f "$compose" ] || die "no install found at $dir (run without --update first)"
    log "Pulling the newest image and recreating (data kept)"
    compose_up up -d --pull always
    [ "$want_plugin" = 1 ] && install_plugin   # refresh the plugin from the newly-pulled image
    log "Updated."
    ;;

  uninstall)
    [ -f "$compose" ] || die "no install found at $dir"
    log "Stopping and removing the stack (project '$PROJECT')"
    compose_up down            # removes containers + network; the bind-mounted ./data is untouched
    rm -f "$compose"
    [ "$want_plugin" = 1 ] && remove_plugin
    # Leave $dir/data in place; only tidy the dir if it is now empty (no data).
    rmdir "$dir" 2>/dev/null || true   # never rm -rf a chosen dir
    log "Uninstalled. Your data is still at $dir/data."
    echo "    erase data too : sudo rm -rf $dir    (files were written by the container as root)"
    echo "    remove image   : docker rmi $IMAGE"
    ;;
esac
