# Zen

**Zen turns a long piece of writing into a deck of short, titled cards** — then
lets you group them, search them, and refine them with an AI agent. It is a
self-hosted knowledge tool: your notes live in a SQLite file on your own
machine, and the intelligence comes from *your* Claude Code session over MCP —
there is no LLM baked into Zen.

## The idea

A spec, a design doc, a research dump — long text is hard to navigate and hard
to reuse. Zen breaks it into **cards**:

- A **card** has a title and a body (markdown, HTML, or plain text).
- Cards live in flat **groups** and carry **tags**.
- A card can be **decomposed** into child cards along its own headings — a
  *lossless split*, so the pieces reproduce the original. The reverse,
  **compose**, joins cards back into one.
- A card that has children *is* a **document** — the multi-section unit of a
  group.
- Cards sit at an abstraction **level** within their group (principle → detail),
  and full-text **search** runs over everything via SQLite FTS5.

## AI, over MCP — not inside Zen

Zen ships no model. Instead it exposes its operations as **MCP tools**
(`card.create`, `decompose`, `compose`, `search`, `group.*`, …) that an external
agent drives. The companion **Claude Code plugin** wires this up both ways:

- you chat from Zen's web UI and the message is pushed into your Claude Code
  session as a channel event;
- Claude replies and edits your cards through the MCP tools, staying anchored to
  the card or group you were looking at.

See [`plugins/zen/README.md`](plugins/zen/README.md) for the plugin.

## Run it

Zen ships as one self-contained Docker image, with a one-file installer that
sets it up. With only Docker installed:

```bash
curl -fsSL https://raw.githubusercontent.com/xhanio/zen/main/scripts/install.sh -o zen-install.sh
bash zen-install.sh
```

That pulls the image, starts Zen at **http://localhost:38000**, and drops the
matching `zen-channel` plugin binary into `~/.local/bin` (override with
`ZEN_BIN_DIR`; the installer warns if that isn't on your `PATH`). Your cards
live in `~/zen/data` — a plain folder you can back up by copying. Manage it with
`bash zen-install.sh --update` / `--uninstall`.

Swap `main` for a tag (`.../zen/v1.0.1/scripts/install.sh`) to pin the installer
to a release.

The installer also registers the Claude Code plugin for you — it adds the
[`xhanio` marketplace](https://github.com/xhanio/plugins) and installs the `zen`
plugin, alongside the `zen-channel` binary it needs. Restart Claude Code
afterwards to load it. Pass `--no-plugin` on a host that only runs the server.

Loading the plugin is not quite enough for the chat-from-the-UI half. Claude
Code only registers Zen's channel when launched with an extra flag, so the
installer also adds this alias to your shell rc (`~/.zshrc`, or `~/.bashrc` /
`~/.bash_profile` on Linux / macOS bash; override with `ZEN_SHELL_RC`):

```bash
alias claude='claude --dangerously-load-development-channels plugin:zen@xhanio'
```

Open a new shell and `claude` does the right thing. Without that flag the
plugin still loads its skills and MCP tools — cards, search, decompose all work
— but messages you send from Zen's web UI never reach your session. Channels
are a research preview; that flag is what registers one today, and it must be
passed *alone* (never alongside `--channels`). Use `\claude` to launch without
it. See [`plugins/zen/README.md`](plugins/zen/README.md) for the details.

The installer reads a few environment variables, all optional:

| Variable | Default | Use it to |
|---|---|---|
| `ZEN_IMAGE` | `docker.io/xhanio/zen-allinone:latest` | pull the server image from a mirror, a private registry, or a local build |
| `ZEN_MARKETPLACE` | `https://github.com/xhanio/plugins` | add the plugin marketplace from a fork or internal mirror |
| `ZEN_BIN_DIR` | `~/.local/bin` | put the `zen-channel` binary somewhere else on your `PATH` |
| `ZEN_SHELL_RC` | per shell/OS | write the `claude` alias to a specific rc file |

One separate variable, `ZEN_BACKEND_URL`, is read by the `zen-channel` binary at
run time rather than by the installer — set it in your shell when Zen is
published on a port other than 38000. It steers the channel connection only, not
the `zen` MCP server URL pinned in the plugin's `.mcp.json`, so a non-default
port takes more than this one variable; see
[`plugins/zen/README.md`](plugins/zen/README.md).

## Develop

Prerequisites: **Go 1.26+**, **Node 20+**, and [`gopro`](https://github.com/xhanio/gopro)
(the build tool) on your `$PATH`. The backend needs **cgo** and builds with
`-tags sqlite_fts5`, so a C toolchain (`build-essential` / Xcode CLT) has to be
present — without it the build fails on the SQLite driver rather than on
anything Zen-specific.

```bash
make deps      # npm install in frontend/
make dev       # zen-backend + Vite dev server together (Ctrl-C stops both)
make test      # go test ./...  +  frontend vitest
make help      # all targets
```

`make dev` serves the SPA against a local backend on a `/tmp/zen-local.db`
SQLite file; `make reset-db` wipes it to re-run migrations.

## Architecture

Three Go binaries and a web UI, over one SQLite database:

| Component | What it is | Stack |
|---|---|---|
| **zen-backend** | REST + WebSocket API, migrations, FTS5 search | Go, Echo, SQLite (cgo, `sqlite_fts5`) |
| **zen-mcp** | the `zen` HTTP MCP server (card/group/search tools) | Go, `modelcontextprotocol/go-sdk` |
| **zen-channel** | the plugin's local stdio MCP server (channel push + `reply`) | Go, pure (no cgo) |
| **zen-frontend** | the single-page web UI | Vue 3, Vite, TypeScript, Pinia, Tailwind |

SQLite is deliberate: single-file, single-binary, trivial to back up. FTS5
powers search, and `-tags sqlite_fts5` is required at build time — the backend
creates FTS5 virtual tables at boot.

## Deployment

Built and packaged with [`gopro`](https://github.com/xhanio/gopro) from
[`project.yaml`](project.yaml). Two delivery modes:

- **all-in-one** (`-e allinone`) — one image (`docker.io/xhanio/zen-allinone`)
  running nginx + backend + MCP daemon under a supervising entrypoint; the
  `install.sh` above targets this. `linux/amd64`; Docker Desktop emulates it on
  Apple Silicon.
- **prod** (`-e prod`) — the same three services as separate images behind a
  compose file, for a conventional multi-container deploy. Build and bring it
  up with:

  ```bash
  gopro build binary -e prod
  gopro generate config -e prod
  gopro build image -e prod              # tags localhost:5000/zen-*:<image_tag>
  gopro generate docker-compose -e prod
  docker compose -f dist/prod/docker-compose.yaml -p prod up -d
  ```

  Pass `-p prod` every time: the database lives on the named volume
  `prod_zen-data`, and without an explicit project name the working directory
  names it instead — you get a fresh empty volume and what looks like data
  loss. The `localhost:5000` prefix is a local tag, not a registry to push to.

On a release, bump the top-level `version:` in `project.yaml` **and** the
per-environment `image_tag:` under each env — they don't follow each other, so
bumping only `version` builds images under the previous tag.

## Layout

```
pkg/                  backend, MCP, channel, and service packages
frontend/             Vue SPA
build/                Dockerfiles + binary build sources
env/                  per-environment gopro config + compose templates
scripts/install.sh    the installer users curl down (served raw from GitHub)
plugins/zen/          the Claude Code plugin (skills + .mcp.json)
project.yaml          gopro build/deploy configuration
```
