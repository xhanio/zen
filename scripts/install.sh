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
#   install.sh --no-plugin ...    skip the Claude Code plugin wiring (server-only host)
#   install.sh --help
#
# It also wires up Claude Code, in two halves that are useless apart:
#
#   1. the zen-channel binary -> ~/.local/bin (override with ZEN_BIN_DIR), taken
#      from the image, which carries a build for every host OS/arch;
#   2. the `zen` plugin itself, installed from the xhanio marketplace on GitHub
#      (skills + .mcp.json, which is what declares `"command": "zen-channel"`).
#
# Neither half does anything alone — the plugin has nothing to spawn without the
# binary, and the binary has nothing to invoke it without the plugin — so one
# --no-plugin flag governs both. Pass it on a box that only runs the server.
# Both halves are best-effort: they warn but never fail the install.
#
# Alongside them it adds a `claude` alias carrying the channel flag to your
# shell rc (~/.zshrc, or ~/.bashrc / ~/.bash_profile on Linux / macOS bash;
# override with ZEN_SHELL_RC). Without that flag Claude Code loads the plugin
# but never registers zen's channel, so conversations in the web UI go nowhere.
# The alias sits in a marked block, so re-running rewrites it and --uninstall
# takes it back out. It is part of the plugin wiring, so --no-plugin skips it.
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

# Claude Code plugin, delivered from its own marketplace repo (not from the image).
# Use the full https:// URL, NOT the `xhanio/plugins` shorthand: the shorthand
# makes `claude plugin marketplace add` clone over SSH, which fails on any box
# without a GitHub SSH key — i.e. exactly the new user this installer targets.
MARKETPLACE_REPO="${ZEN_MARKETPLACE:-https://github.com/xhanio/plugins}"
MARKETPLACE="xhanio"                     # the name the marketplace declares in its own marketplace.json
PLUGIN_ID="zen@$MARKETPLACE"

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
install_channel_binary() {
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
         echo "        export PATH=\"$BINDIR:\$PATH\"   # add to $(shell_rc), then restart the shell" ;;
    esac
  else
    docker rm -f "$cid" >/dev/null 2>&1 || true
    warn "image has no $PLUGIN_IN_IMAGE/$name — skipping the plugin (older image without a baked plugin binary?)"
  fi
}

remove_channel_binary() {
  [ -f "$BINDIR/zen-channel" ] || return 0
  rm -f "$BINDIR/zen-channel" && log "Removed zen-channel from $BINDIR"
}

# --- half 1b: the `claude` alias that actually opens the channel --------------
# Installing the plugin is not enough: Claude Code only registers zen's channel
# when launched with --dangerously-load-development-channels, so a bare `claude`
# loads the skills and MCP tools but never receives conversation events. The
# alias bakes that flag in. Pass ONLY this flag with the plugin ref — adding a
# separate --channels silently breaks registration.
ALIAS_CMD="claude --dangerously-load-development-channels plugin:$PLUGIN_ID"
ALIAS_BEGIN="# >>> zen (Claude Code channel) >>>"
ALIAS_END="# <<< zen (Claude Code channel) <<<"

# Which rc file the user's interactive shell actually reads. bash splits by OS:
# macOS terminals start login shells (~/.bash_profile), Linux ones don't
# (~/.bashrc). Override with $ZEN_SHELL_RC for anything exotic.
shell_rc() {
  if [ -n "${ZEN_SHELL_RC:-}" ]; then printf '%s' "$ZEN_SHELL_RC"; return; fi
  case "$(basename "${SHELL:-}")" in
    zsh)  printf '%s' "${ZDOTDIR:-$HOME}/.zshrc" ;;
    bash) if [ "$(uname -s)" = Darwin ]; then printf '%s' "$HOME/.bash_profile"
          else                                printf '%s' "$HOME/.bashrc"; fi ;;
    *)    printf '%s' "$HOME/.profile" ;;   # ksh/sh/unknown: POSIX fallback
  esac
}

# Drop the marked block from $1, if present. Also used by install, so a changed
# flag or plugin id replaces the old line instead of stacking a second alias.
strip_alias_block() {
  local rc="$1" tmp
  [ -f "$rc" ] || return 0
  grep -qF "$ALIAS_BEGIN" "$rc" 2>/dev/null || return 0
  tmp=$(mktemp)
  awk -v b="$ALIAS_BEGIN" -v e="$ALIAS_END" '
    $0==b { skip=1; next }
    $0==e { skip=0; next }
    !skip { print }' "$rc" > "$tmp" && cat "$tmp" > "$rc"   # cat, not mv: keep the rc's inode/perms
  rm -f "$tmp"
}

install_shell_alias() {
  local rc; rc=$(shell_rc)
  strip_alias_block "$rc"
  # A missing rc is normal on a fresh box — create it rather than skipping.
  { printf '%s\n' "$ALIAS_BEGIN"
    printf "alias claude='%s'\n" "$ALIAS_CMD"
    printf '%s\n' "$ALIAS_END"
  } >> "$rc" || { warn "could not write $rc — add this yourself:"
                  echo "        alias claude='$ALIAS_CMD'"; return 0; }
  log "claude alias -> $rc  (run 'source $rc' or open a new shell)"
}

remove_shell_alias() {
  local rc; rc=$(shell_rc)
  grep -qF "$ALIAS_BEGIN" "$rc" 2>/dev/null || return 0
  strip_alias_block "$rc" && log "Removed the claude alias from $rc"
}

# --- half 2: the Claude Code plugin -----------------------------------------
# `claude plugin` is idempotent — re-adding a marketplace that is already on
# disk, or re-installing an installed plugin, both succeed and exit 0 — so these
# are safe to re-run from --update without any "is it already there?" probing.

have_claude() { command -v claude >/dev/null 2>&1; }

warn_no_claude() {
  warn "Claude Code ('claude') is not on your PATH — skipping the plugin. Once it is installed:"
  echo "        claude plugin marketplace add $MARKETPLACE_REPO"
  echo "        claude plugin install $PLUGIN_ID"
}

install_claude_plugin() {
  if ! have_claude; then warn_no_claude; return 0; fi
  if ! claude plugin marketplace add "$MARKETPLACE_REPO" >/dev/null 2>&1; then
    warn "could not add the '$MARKETPLACE' marketplace — skipping the plugin"
    echo "        retry: claude plugin marketplace add $MARKETPLACE_REPO"
    return 0
  fi
  if claude plugin install "$PLUGIN_ID" >/dev/null 2>&1; then
    log "Claude Code plugin -> $PLUGIN_ID  (restart Claude Code to load it)"
  else
    warn "could not install $PLUGIN_ID"
    echo "        retry: claude plugin install $PLUGIN_ID"
  fi
}

update_claude_plugin() {
  if ! have_claude; then warn_no_claude; return 0; fi
  claude plugin marketplace update "$MARKETPLACE" >/dev/null 2>&1 || true
  if claude plugin update "$PLUGIN_ID" >/dev/null 2>&1; then
    log "Claude Code plugin updated  (restart Claude Code to apply)"
  else
    install_claude_plugin   # not installed yet (or the marketplace went away) — install it now
  fi
}

remove_claude_plugin() {
  have_claude || return 0     # no Claude Code, nothing to undo — stay silent
  if claude plugin uninstall "$PLUGIN_ID" -y >/dev/null 2>&1; then
    log "Removed the Claude Code plugin ($PLUGIN_ID)"
  fi
  # The '$MARKETPLACE' marketplace stays registered on purpose: it is inert with
  # nothing installed from it, and it may serve other plugins the user wants.
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
    if [ "$want_plugin" = 1 ]; then
      install_channel_binary
      install_claude_plugin
      install_shell_alias
    fi
    ;;

  update)
    [ -f "$compose" ] || die "no install found at $dir (run without --update first)"
    log "Pulling the newest image and recreating (data kept)"
    compose_up up -d --pull always
    if [ "$want_plugin" = 1 ]; then
      install_channel_binary   # refresh the binary from the newly-pulled image
      update_claude_plugin     # and the plugin from its marketplace
      install_shell_alias      # rewrites the block, so a changed flag propagates
    fi
    log "Updated."
    ;;

  uninstall)
    [ -f "$compose" ] || die "no install found at $dir"
    log "Stopping and removing the stack (project '$PROJECT')"
    compose_up down            # removes containers + network; the bind-mounted ./data is untouched
    rm -f "$compose"
    if [ "$want_plugin" = 1 ]; then
      remove_channel_binary
      remove_claude_plugin
      remove_shell_alias
    fi
    # Leave $dir/data in place; only tidy the dir if it is now empty (no data).
    rmdir "$dir" 2>/dev/null || true   # never rm -rf a chosen dir
    log "Uninstalled. Your data is still at $dir/data."
    echo "    erase data too : sudo rm -rf $dir    (files were written by the container as root)"
    echo "    remove image   : docker rmi $IMAGE"
    ;;
esac
