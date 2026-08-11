<div align="center">

<img src="docs/assets/banner.svg" alt="Sonora — your music, your server, your rules" width="100%">

**A self-hosted music streaming server in Go — lightweight, OpenSubsonic-compatible, and built for people who care about audio fidelity.**

[![CI](https://github.com/raloonsoc/sonora/actions/workflows/ci.yml/badge.svg)](https://github.com/raloonsoc/sonora/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/raloonsoc/sonora?sort=semver)](https://github.com/raloonsoc/sonora/releases)
[![Container](https://img.shields.io/badge/ghcr.io-raloonsoc%2Fsonora-2496ED?logo=docker&logoColor=white)](https://github.com/raloonsoc/sonora/pkgs/container/sonora)
[![License: AGPL v3](https://img.shields.io/badge/license-AGPL--3.0-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.26-00ADD8?logo=go&logoColor=white)](go.mod)

```console
docker pull ghcr.io/raloonsoc/sonora:latest
```

</div>

---

Sonora runs your music library as a streaming service you own. It speaks the
**OpenSubsonic** protocol, so the mobile and desktop clients that already
exist — Feishin, Symfonium, DSub, play:Sub — work against it out of the box.
It ships as a single static Go binary with no JVM and no Node runtime, which
means a Raspberry Pi or a €4/month VPS is enough to run it comfortably.

> [!NOTE]
> **Status: early but usable.** Ingestion, streaming, authentication, lyrics,
> playlists, favorites and a broad OpenSubsonic subset all work end-to-end,
> verified against a real client (Feishin). Multi-arch Docker images are
> published to `ghcr.io`. The native JWT API is the main piece still missing —
> see [Roadmap](#roadmap).

## Table of contents

- [Why Sonora](#why-sonora)
- [Quick start](#quick-start)
  - [Container images](#container-images)
- [How it works](#how-it-works)
- [Configuration](#configuration)
- [OpenSubsonic API coverage](#opensubsonic-api-coverage)
- [Roadmap](#roadmap)
- [Development](#development)
- [Companion CLI](#companion-cli)
- [License](#license)

## Why Sonora

Most self-hosted music servers either force a heavyweight stack on you or lock
you into their own client. Sonora takes the opposite position:

| | |
|---|---|
| **One static binary** | No JVM, no Node, no sprawling dependency tree. Runs on constrained hardware — Raspberry Pi, small VPS, a Proxmox LXC. |
| **Bring your own client** | OpenSubsonic-compatible, so the existing client ecosystem works immediately. You are not locked into a first-party app. |
| **Fidelity first** | Bit-perfect FLAC passthrough by default. Transcoding happens only when the source exceeds what clients reliably handle, and the client picks the target format. |
| **Terminal-native** | A Bubble Tea TUI client is developed alongside the server for people who live in a terminal. |
| **Legally clean by design** | Sonora serves a library you already own. There is no scraping, downloading, or acquisition tooling here — and that is a deliberate scope boundary, not an oversight. |

## Quick start

**Requirements:** Docker and Docker Compose. Everything else — `ffmpeg`,
`ffprobe`, `chromaprint` — is bundled in the image. Nothing to compile.

Create a `docker-compose.yml`:

```yaml
services:
  postgres:
    image: postgres:16-alpine
    restart: unless-stopped
    environment:
      POSTGRES_USER: sonora
      POSTGRES_PASSWORD: sonora
      POSTGRES_DB: sonora
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U sonora"]
      interval: 5s
      timeout: 3s
      retries: 10

  sonora:
    image: ghcr.io/raloonsoc/sonora:latest
    restart: unless-stopped
    depends_on:
      postgres:
        condition: service_healthy
    ports:
      - "4533:4533"
    environment:
      SONORA_DATABASE_URL: postgres://sonora:sonora@postgres:5432/sonora?sslmode=disable
      SONORA_JWT_SECRET: ${SONORA_JWT_SECRET:?generate with openssl rand -base64 32}
    volumes:
      - /path/to/your/music:/music:ro
      - cache:/cache

volumes:
  pgdata:
  cache:
```

Then bring it up and create your first user:

```bash
export SONORA_JWT_SECRET=$(openssl rand -base64 32)
docker compose up -d

docker compose exec sonora sonora create-user \
  --username admin --password '<password>' --admin
```

Sonora listens on **`:4533`**. Point any OpenSubsonic client at
`http://<host>:4533` with those credentials and your library will appear as
the watcher ingests it.

The database schema is migrated automatically on startup — no external
`migrate` CLI needed at runtime.

> [!TIP]
> Mount your library read-only (`:ro`, as above). Sonora never writes to your
> source files — cover art and transcodes go to the separate cache volume.

### Container images

Published to the GitHub Container Registry, built for **`linux/amd64`** and
**`linux/arm64`** (so a Raspberry Pi 4/5 works without changes):

```bash
docker pull ghcr.io/raloonsoc/sonora:latest
```

| Tag | Tracks |
|---|---|
| `latest` | Newest release. Convenient, but moves under you. |
| `0.1.0` | An exact release. **Recommended for real deployments.** |
| `0.1` | Latest patch within the `0.1` minor line. |
| `0` | Latest release within the `0.x` major line. |

> [!WARNING]
> While Sonora is pre-1.0, `0.x` and `0.1` can still bring breaking changes
> between minor versions. Pin an exact tag if you care about stability.

### Building from source instead

If you'd rather build the image yourself, the repo ships a compose file that
does exactly that:

```bash
git clone https://github.com/raloonsoc/sonora.git
cd sonora/deploy
cp .env.example .env   # set SONORA_JWT_SECRET, point SONORA_LIBRARY_HOST_PATH at your music
docker compose up -d --build
```

## How it works

### Ingestion

A polling watcher scans the library on a configurable interval (default 30s)
and ingests anything not yet in the database.

```
library file (.flac .mp3 .m4a .aac .opus .ogg .wav)
   │
   ├─ ffprobe ............... metadata (case-insensitive tag lookup)
   ├─ fpcalc ................ Chromaprint fingerprint → duplicate detection
   ├─ ffmpeg loudnorm ....... ReplayGain analysis (EBU R128)
   ├─ cover art ............. embedded picture, iTunes Search API fallback
   └─ artist/album dedup .... then INSERT → PostgreSQL
```

Polling rather than `fsnotify` is a deliberate choice: `fsnotify` does not
reliably observe host-side changes through a Docker Desktop/OrbStack bind
mount, which is exactly how most people will run this. Polling behaves
identically whether Sonora runs natively or containerized.

Duplicate detection is advisory. An exact fingerprint match — the same song
re-encoded at a different bitrate or format — logs a warning but does **not**
block ingestion, since a false positive would otherwise silently swallow a
legitimate track.

Albums are grouped by the `album_artist` tag (falling back to `artist`) so a
featured-artist track doesn't fork off a duplicate album. Multi-artist credits
are parsed on ingest (`&`, `,`, `feat.`, `ft.`, `featuring`) into a full
ordered artist list, while a single primary artist is retained per track.

### Streaming

```
GET /rest/stream?id=…&format=…
   │
   ├─ sample rate ≤ 48kHz, or format=raw → FLAC/original passthrough,
   │                                        bit-perfect
   └─ sample rate > 48kHz → on-demand transcode
                             ├─ format=aac → AAC 192k in a fragmented MP4
                             │   container (not raw ADTS — Safari, macOS/
                             │   iOS apps, and Amperfy fail to play plain
                             │   ADTS AAC)
                             ├─ format=mp3 → MP3 192k, the closest thing
                             │   to a universal fallback
                             ├─ no format, or anything else → Opus 128k
                             │   (default: best fidelity per bit of the
                             │   three, but the least universally
                             │   supported)
                             ├─ cached on local disk, one file per
                             │   (track, format) pair
                             ├─ per-path mutex (no duplicate transcode on
                             │   concurrent requests for the same track),
                             │   bounded by SONORA_TRANSCODE_WORKERS
                             │   concurrent ffmpeg processes
                             └─ swept after 30 days
```

`format` follows the OpenSubsonic `stream` parameter: clients that can't
decode Opus (Amperfy is the concrete case this was built for) ask for
`aac` or `mp3` instead, rather than being force-fed a format they can't
play.

All streaming goes through `http.ServeContent`, so HTTP Range requests —
seeking, resuming — are handled correctly.

### Lyrics

Local `.lrc` files sitting next to the audio are parsed into OpenSubsonic
`structuredLyrics` with millisecond timestamps. When no local file exists,
Sonora falls back to the [LRCLIB] API (exact match first, then fuzzy search).
The fallback can be disabled to keep lookups fully offline.

### Authentication

OpenSubsonic's contract requires the server to verify `MD5(password + salt)`,
which means it cannot store only a one-way hash. Sonora therefore keeps two
representations: a **bcrypt** hash for the future native API, and a
**reversible AES-256-GCM** encrypted copy used solely to satisfy the Subsonic
token handshake. Every OpenSubsonic endpoint is gated by that token auth.

## Configuration

Configuration is **environment variables only** — no config files, nothing
hardcoded — so third-party Docker/Compose deployments stay simple.

### Required

| Variable | Description |
|---|---|
| `SONORA_DATABASE_URL` | PostgreSQL connection string. |
| `SONORA_JWT_SECRET` | Secret for signing native API JWTs. Generate with `openssl rand -base64 32`. |

### Optional

| Variable | Default | Description |
|---|---|---|
| `SONORA_HTTP_ADDR` | `:4533` | Listen address. |
| `SONORA_LOG_LEVEL` | `info` | Log verbosity. |
| `SONORA_LIBRARY_PATH` | `/music` | Library root inside the container; the watcher scans it directly. |
| `SONORA_INGEST_POLL_INTERVAL_SECONDS` | `30` | Watcher polling interval. |
| `SONORA_COVER_ART_DIR` | `/cache/covers` | Extracted cover art storage. |
| `SONORA_TRANSCODE_CACHE_PATH` | `/cache` | Transcode cache directory. |
| `SONORA_TRANSCODE_WORKERS` | `2` | Concurrent `ffmpeg` transcode workers. |
| `SONORA_LYRICS_LRCLIB_FALLBACK` | `true` | Query LRCLIB when no local `.lrc` exists. Set `false` to stay fully offline. |

Compose-level paths (`SONORA_LIBRARY_HOST_PATH`, `SONORA_CACHE_HOST_PATH`)
control what gets bind-mounted from the host. See
[`deploy/.env.example`](deploy/.env.example).

## OpenSubsonic API coverage

Responses support both XML (protocol default) and JSON (`f=json`). Every
endpoint is registered both bare and with the legacy `.view` suffix that
older clients still send.

| Area | Endpoints |
|---|---|
| **System** | `ping` · `getLicense` · `getOpenSubsonicExtensions` · `getMusicFolders` |
| **Browsing** | `getArtists` · `getArtist` · `getAlbum` · `getAlbumList2` · `getSong` · `getGenres` |
| **Search** | `search3` |
| **Media** | `stream` · `getCoverArt` |
| **Playlists** | `getPlaylists` · `getPlaylist` · `createPlaylist` · `updatePlaylist` · `deletePlaylist` |
| **Favorites** | `star` · `unstar` · `getStarred2` |
| **Lyrics** | `getLyricsBySongId` (synced) · `getLyrics` (legacy) |
| **Other** | `scrobble` · `getUser` |

Browsing is **ID3-based** (`getArtists`/`getAlbum`). The legacy folder-based
endpoints (`getIndexes`, `getMusicDirectory`) are not implemented — modern
clients use the ID3 endpoints.

## Roadmap

- [ ] **JWT auth for the native API** — the last piece blocking `sonora-cli`.
      Subsonic token auth is unrelated and already done.
- [ ] Propagate `starred` to `search3` and `getArtist` (already present on
      `getSong`, `getAlbum`, `getAlbumList2`, `getStarred2`).
- [ ] Offload the transcode cache to S3-compatible storage (R2 / MinIO).
      Configuration keys are reserved but the backend is not implemented —
      the cache is local-disk only today.

<details>
<summary><strong>Already shipped</strong></summary>

- PostgreSQL schema with `golang-migrate` migrations and `sqlc`-generated
  type-safe queries; automatic migration on startup.
- Polling library watcher, container-safe.
- Full ingestion pipeline: metadata, ReplayGain (EBU R128), cover art with
  iTunes fallback, artist/album dedup, multi-artist parsing.
- Chromaprint fingerprinting for duplicate detection.
- Streaming with Range support, FLAC passthrough, on-demand transcode
  (Opus/AAC/MP3, negotiated via the OpenSubsonic `format` param) with
  bounded concurrency, per-path locking, and a 30-day cache sweep.
- bcrypt + AES-256-GCM credential storage, Subsonic token auth, and
  `sonora create-user` provisioning.
- Synced lyrics: `.lrc` parser plus LRCLIB fallback.
- 25 OpenSubsonic endpoints, XML and JSON, verified against Feishin.
- Multi-arch Docker images (amd64/arm64) published to `ghcr.io`, with CI
  running vet, lint, and the full test suite against a real PostgreSQL.

</details>

## Development

**Requirements:** Go 1.26+, PostgreSQL, `ffmpeg`/`ffprobe`, `chromaprint`
(`fpcalc`).

```bash
go build ./...
go vet ./...
```

### Tests

Tests need a real PostgreSQL database with migrations already applied:

```bash
export SONORA_TEST_DATABASE_URL="postgres://user:pass@localhost:5432/sonora_test?sslmode=disable"
migrate -database "$SONORA_TEST_DATABASE_URL" -path internal/db/migrations up

go test -race -cover -p 1 ./...
```

> [!IMPORTANT]
> `-p 1` is required, not cosmetic. Several test packages `TRUNCATE` shared
> tables in the same database, and Go runs packages in parallel by default —
> without it you get flaky foreign-key failures unrelated to your changes. CI
> uses the same flag.

The `migrate` CLI must be built with the `postgres` build tag, or it will
compile fine and then fail at runtime with `unknown driver postgres`:

```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.19.1
```

### Layout

```
cmd/sonora/            entrypoint + create-user subcommand
internal/config/       environment configuration
internal/ingest/       watcher, ffprobe, fingerprint, replaygain, cover art
internal/db/           migrations + sqlc-generated queries
internal/domain/       core models
internal/streaming/    Range handler, transcode, cache
internal/lyrics/       .lrc parser, LRCLIB client
internal/auth/         bcrypt, AES-256-GCM, Subsonic token auth
internal/subsonic/     OpenSubsonic endpoint handlers
deploy/                Dockerfile, compose, .env.example
```

Database access uses [`sqlc`](https://sqlc.dev) — hand-written SQL compiled
into type-safe Go — rather than an ORM, matching the lightweight design goal.

## Companion CLI

A terminal client lives in a separate repository: **`sonora-cli`** (Go +
[Bubble Tea]). It targets any OpenSubsonic server, not just Sonora.

- Playback delegated to `mpv` as a subprocess, driven over its IPC socket.
- Cover art rendered in-terminal via [rasterm] on terminals with a graphics
  protocol (Kitty, iTerm2, Sixel), with ASCII art as a fallback.
- Synced lyrics panel that highlights the current line from `mpv`'s reported
  playback position.

Prior art worth a look: [`stmp`](https://github.com/wildeyedskies/stmp).

## Prior art and references

- [Navidrome](https://github.com/navidrome/navidrome) — Go, a real-world
  Subsonic implementation used as a reference.
- [OpenSubsonic specification](https://opensubsonic.netlify.app/)
- [LRCLIB] — open API for synced lyrics.

## License

Sonora is licensed under the [GNU Affero General Public License v3.0](LICENSE).

The AGPL was chosen deliberately. If you run a modified version of Sonora as a
network service, you must make your modifications available to its users. This
keeps the project open when it is *deployed*, not merely when it is
distributed.

[LRCLIB]: https://lrclib.net/
[Bubble Tea]: https://github.com/charmbracelet/bubbletea
[rasterm]: https://github.com/BourgeoisBear/rasterm
