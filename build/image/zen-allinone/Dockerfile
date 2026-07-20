# Stage 1: build the SPA bundle (mirrors build/image/zen-frontend/Dockerfile).
# BASE is a global ARG (declared before the first FROM) so stage 2's FROM can use it.
ARG BASE=ubuntu:22.04
# Pinned to BUILDPLATFORM: the bundle is arch-agnostic static output, so it is
# built once natively and copied into every target arch. Without the pin, a
# multi-arch build re-runs npm ci + vite under QEMU for each arch, which is
# enormously slow and produces identical output.
FROM --platform=$BUILDPLATFORM node:20-alpine AS spa

WORKDIR /build
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --no-audit --no-fund

COPY frontend/ ./
# vite.config.ts reads the app version from the repo's project.yaml — make it
# reachable or the bundle reports 'dev'.
COPY project.yaml /project.yaml
RUN npm run build

# Stage 2: ubuntu runtime — nginx + zen-backend + zen-mcp daemon.
FROM ${BASE}

ARG CONFIG_TGT
# Set by buildx per target ("amd64", "arm64"). Selects the arch-matched binaries
# built by gopro as <name>_<goos>_<goarch>. A bare `COPY bin/zen-backend` would
# silently package the build host's arch into every image.
ARG TARGETARCH

RUN apt-get update && apt-get install -y \
    nginx \
    ca-certificates \
    tzdata \
    && rm -rf /var/lib/apt/lists/*

RUN mkdir -p /app /data \
    /var/log/zen-backend /etc/zen-backend \
    /var/log/zen-mcp /etc/zen-mcp

# Backend binary + config + migrations.
COPY bin/zen-backend_linux_${TARGETARCH} /app/zen-backend
COPY ${CONFIG_TGT}/zen-backend/config.yaml /etc/zen-backend/config.yaml
COPY ${CONFIG_TGT}/zen-backend/migrations /etc/zen-backend/migrations

# MCP daemon binary + config.
COPY bin/zen-mcp_linux_${TARGETARCH} /app/zen-mcp
COPY ${CONFIG_TGT}/zen-mcp/config.yaml /etc/zen-mcp/config.yaml

# zen-channel plugin binaries, carried for `docker cp` — nothing here ever runs
# in this container. The Zen Claude Code plugin spawns zen-channel on the USER'S
# machine (its .mcp.json declares `"command": "zen-channel"`), so the binary must
# match their host, and TARGETARCH cannot pick it: TARGETARCH is this image's
# arch, which says nothing about where the operator sits. Zen on a Linux server
# with Claude Code on a Mac laptop is an ordinary self-hosted setup, and there
# the image is linux/amd64 while the binary needed is darwin/arm64.
#
# So every host build ships regardless of image arch. ~42MB of cargo, which is
# the price of the image being the only delivery path a new user has.
#
#   docker cp zen-allinone:/app/plugin/zen-channel_darwin_arm64 \
#     /usr/local/bin/zen-channel
COPY bin/zen-channel_darwin_arm64 /app/plugin/zen-channel_darwin_arm64
COPY bin/zen-channel_darwin_amd64 /app/plugin/zen-channel_darwin_amd64
COPY bin/zen-channel_linux_arm64  /app/plugin/zen-channel_linux_arm64
COPY bin/zen-channel_linux_amd64  /app/plugin/zen-channel_linux_amd64

# The rendered compose file, carried for `docker cp` so the image is a
# self-contained deliverable: pull it, extract this, `docker compose up`. It
# pins zen-allinone:latest, so the extracted stack pulls the newest image.
#
# This makes the image build depend on a GENERATED file: run
# `gopro generate docker-compose -e allinone` BEFORE `gopro build image`, or the
# COPY bakes a stale compose (dist/ is gitignored, so it is whatever was last
# rendered). That reverses the usual order where compose is generated last.
#
#   docker cp zen-allinone:/app/deploy/docker-compose.yaml ./docker-compose.yaml
COPY dist/allinone/docker-compose.yaml /app/deploy/docker-compose.yaml

# NOTE: install.sh is deliberately NOT baked in. It used to live at
# /app/install.sh as a stopgap while the repo was private; now that it is public
# the durable path is the raw URL, which needs no image pull just to read a
# shell script and cannot go stale against the tag it installs:
#
#   curl -fsSL https://raw.githubusercontent.com/xhanio/zen/main/scripts/install.sh -o zen-install.sh
#
# The compose above and the /app/plugin binaries stay — those genuinely have to
# ship with the image, since they are build outputs rather than source.

# nginx: drop Ubuntu's default site, install ours (http{} includes conf.d/*.conf).
RUN rm -f /etc/nginx/sites-enabled/default
COPY build/image/zen-allinone/nginx.conf /etc/nginx/conf.d/zen.conf

# SPA bundle.
COPY --from=spa /build/dist /usr/share/nginx/html

# Supervising entrypoint.
COPY build/image/zen-allinone/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 80
ENTRYPOINT ["/entrypoint.sh"]
