# Sonora

**Sonora** is a self-hosted music streaming server written in Go. It's built
to be lightweight enough to run on a Raspberry Pi or a small VPS, speaks the
**OpenSubsonic** protocol so you can use the mobile apps that already exist
for it, and pairs with a custom terminal client as its main differentiator.

> Status: early development, but usable. The ingestion pipeline, streaming,
> authentication, lyrics, and a broad OpenSubsonic subset all work
> end-to-end — verified against a real client (Feishin). Docker packaging
> (with auto-migration on startup) is ready.

## Why

Most self-hosted music servers either force you into a heavyweight stack or
lock you into their own app. Sonora aims for the opposite:

- **One static Go binary.** No JVM, no Node runtime, no bloated dependency
  tree. Runs comfortably on constrained hardware (Raspberry Pi, small VPS,
  a Proxmox LXC).
- **OpenSubsonic-compatible.** Point any existing Subsonic/OpenSubsonic
  client at it (DSub, Symfonium, play:Sub, etc.) and it just works.
- **A first-class terminal client.** A Bubble Tea TUI ships alongside it for
  people who live in the terminal — see [Companion CLI](#companion-cli).
- **High audio fidelity.** Bit-perfect FLAC passthrough when the client
  supports it, adaptive transcoding (Opus/AAC) when it doesn't.
- **Legally clean by design.** Sonora only serves a library you already own.
  There is no scraping, downloading, or acquisition tooling in this repo —
  it manages files you put there, nothing else.

## Architecture

```
Ingestion (polling watcher, configurable interval)
    → ffprobe (metadata) + chromaprint (fingerprint, dedup)
    → normalize to FLAC when needed
    → ReplayGain (EBU R128)
    → cover art extraction, with an iTunes Search API fallback when the
      embedded picture can't be read locally
    → PostgreSQL (metadata)

API (Go)
├── Native API — JWT auth, free-form design, used by the CLI client
├── OpenSubsonic layer — adapter/translator over the same domain models
│   ├── auth: MD5(password + salt) via query params (fixed contract)
│   ├── getArtists / getArtist / getAlbum / getAlbumList2 / getGenres /
│   │   getSong / search3
│   ├── getCoverArt
│   ├── getLyricsBySongId (OpenSubsonic, synced) + getLyrics (legacy),
│   │   with a local .lrc parser and an LRCLIB fallback (exact match,
│   │   then fuzzy search)
│   ├── getPlaylists (read-only) / scrobble
│   └── stream — Range requests, FLAC passthrough or on-demand transcode
└── Storage — originals on local disk, transcode cache on local disk or
    S3-compatible storage (R2 / MinIO)
```

The native API and the OpenSubsonic layer are two views over the same
domain model — OpenSubsonic support is an adapter, not a fork of the core
logic.

## Features

### Done / working

- PostgreSQL schema (tracks, albums, artists, users, playlists) with
  `golang-migrate` migrations and `sqlc`-generated queries.
- Polling-based library watcher (configurable interval, default 30s):
  scans the library and ingests any file not yet in the database. Works
  identically whether Sonora runs natively or inside Docker with a bind
  mount — unlike `fsnotify`, which doesn't reliably see host-side changes
  through a Docker Desktop/OrbStack bind mount.
- Full ingestion pipeline, watcher to database with no manual steps:
  metadata extraction (`ffprobe`, with case-insensitive tag lookup),
  ReplayGain analysis (EBU R128 via `ffmpeg loudnorm`), cover art
  extraction (with an iTunes Search API fallback), artist/album dedup,
  insert. Albums are grouped by the `album_artist` tag (falling back to
  `artist`) so a featured-artist track doesn't fork off a duplicate album.
- Chromaprint audio fingerprinting (`fpcalc`) on ingest: an exact-match
  fingerprint lookup flags likely duplicate tracks (e.g. the same song
  re-encoded at a different bitrate/format) with a warning log —
  ingestion isn't blocked, since a false positive would otherwise
  silently drop a legitimate track.
- Streaming handler: HTTP Range requests (`http.ServeContent` over an
  `io.ReadSeeker`), FLAC passthrough.
- Authentication: bcrypt password storage for the future native API,
  reversible AES-256-GCM storage + Subsonic token auth
  (`MD5(password + salt)`) gating every OpenSubsonic endpoint, CLI
  user provisioning (`sonora create-user`).
- Lyrics: `.lrc` parser producing OpenSubsonic `structuredLyrics`
  (millisecond timestamps), falling back to the [LRCLIB] API (exact
  match, then fuzzy search) when no local file exists.
- OpenSubsonic support, verified end-to-end against a real client
  (Feishin): `ping`, `getArtists`, `getArtist`, `getAlbum`, `getAlbumList2`,
  `getGenres`, `getSong`, `getCoverArt`, `stream`, `scrobble`, `search3`,
  `getLyricsBySongId`, `getLyrics`, `getPlaylists`, `getPlaylist`,
  `createPlaylist`, `updatePlaylist`, `deletePlaylist`, `star`, `unstar`,
  `getStarred2`, `getMusicFolders`, `getOpenSubsonicExtensions`, `getUser`,
  `getLicense`.
  Responses support both XML (protocol default) and JSON (`f=json`).
- Docker packaging: multi-stage `Dockerfile` (Alpine, `ffmpeg`/`ffprobe`
  bundled), `docker-compose.yml` with PostgreSQL + healthcheck, and
  automatic schema migration on startup (no external `migrate` CLI
  needed).

### Planned

- [ ] JWT auth for the native API (used by the future `sonora-cli` client).
- [ ] Multi-arch builds (amd64/arm64), GitHub Actions CI, image published
      to `ghcr.io`.

## Companion CLI

A terminal client lives in a separate repository: **sonora-cli** (Go +
[Bubble Tea]). It talks to any OpenSubsonic-compatible server, not just
Sonora. Highlights:

- Playback delegated to `mpv` as a subprocess, controlled over its IPC
  socket (play/pause/seek/volume).
- Cover art rendered in-terminal via [rasterm] when the terminal supports a
  graphics protocol (Kitty, iTerm2, Sixel), falling back to ASCII art
  otherwise.
- Synced lyrics in a side panel, highlighting the current line based on
  playback position reported by `mpv`.

See [`stmp`](https://github.com/wildeyedskies/stmp) for a similar prior art
project.

## Getting started

```bash
cd deploy
cp .env.example .env   # edit values, especially SONORA_JWT_SECRET
docker compose up -d
docker compose exec sonora sonora create-user --username admin --password <password> --admin
```

Configuration is entirely via environment variables — no hardcoded config
files. Point your library at the `SONORA_LIBRARY_HOST_PATH` folder (or
`deploy/library` by default) and Sonora will pick up files automatically.

## Tech stack

- **Language:** Go
- **Database:** PostgreSQL
- **Media tooling:** `ffmpeg` / `ffprobe`, `chromaprint` (`fpcalc`)
- **Protocol:** [OpenSubsonic](https://opensubsonic.netlify.app/)
- **Object storage (optional):** R2 / MinIO for transcode cache

## Prior art / references

- [Navidrome](https://github.com/navidrome/navidrome) — Go, real-world
  Subsonic API implementation used as a reference.
- [OpenSubsonic API spec](https://opensubsonic.netlify.app/)
- [LRCLIB] — open API for synced lyrics.

[LRCLIB]: https://lrclib.net/
[Bubble Tea]: https://github.com/charmbracelet/bubbletea
[rasterm]: https://github.com/BourgeoisBear/rasterm

## License

Sonora is licensed under the [GNU Affero General Public License v3.0](LICENSE).
The AGPL was chosen deliberately: if you run a modified version of Sonora
as a network service, you must make the modified source available to your
users. This keeps the project open even when deployed, not just when
distributed.
