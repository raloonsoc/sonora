# Sonora

**Sonora** is a self-hosted music streaming server written in Go. It's built
to be lightweight enough to run on a Raspberry Pi or a small VPS, speaks the
**OpenSubsonic** protocol so you can use the mobile apps that already exist
for it, and pairs with a custom terminal client as its main differentiator.

> Status: early development. The ingestion pipeline and the minimal
> OpenSubsonic subset work end-to-end against a real Postgres-backed
> server. Not usable with a real Subsonic client yet — authentication
> isn't wired up.

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
Ingestion (fsnotify watcher, 5s debounce)
    → ffprobe (metadata) + chromaprint (fingerprint, dedup)
    → normalize to FLAC when needed
    → ReplayGain (EBU R128)
    → cover art extraction
    → PostgreSQL (metadata)

API (Go)
├── Native API — JWT auth, free-form design, used by the CLI client
├── OpenSubsonic layer — adapter/translator over the same domain models
│   ├── auth: MD5(password + salt) via query params (fixed contract)
│   ├── getArtists / getArtist / getAlbum / search3
│   ├── getCoverArt
│   ├── getLyricsBySongId (OpenSubsonic, synced) + getLyrics (legacy)
│   ├── getPlaylists / createPlaylist / scrobble
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
- Filesystem watcher (`fsnotify`) with debounce to avoid processing
  half-copied files.
- Full ingestion pipeline, watcher to database with no manual steps:
  metadata extraction (`ffprobe`), ReplayGain analysis (EBU R128 via
  `ffmpeg loudnorm`), cover art extraction, artist/album dedup, insert.
- Streaming handler: HTTP Range requests (`http.ServeContent` over an
  `io.ReadSeeker`), FLAC passthrough.
- OpenSubsonic minimal subset, all working against a real Postgres-backed
  server: `ping`, `getArtists`, `getAlbum`, `getCoverArt`, `stream`,
  `search3`.

### Planned

- [ ] Authentication: Subsonic token auth (`MD5(password + salt)`) gating
      every OpenSubsonic endpoint; JWT auth for the native API.
- [ ] On-demand transcoding (Opus/AAC) with a worker pool, cache lookup
      before re-encoding.
- [ ] Chromaprint/`fpcalc` fingerprinting for duplicate detection.
- [ ] Playlists, scrobbling.
- [ ] Lyrics: `.lrc` parser producing OpenSubsonic `structuredLyrics`
      (millisecond timestamps), with optional fallback to the [LRCLIB]
      API when no local file exists.
- [ ] Docker packaging: multi-stage `Dockerfile`, `docker-compose.yml`
      with PostgreSQL, multi-arch builds (amd64/arm64), GitHub Actions CI,
      image published to `ghcr.io`.

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

Not ready yet. Once the first runnable version lands, this section will
cover:

```bash
docker compose up -d
```

with configuration entirely via environment variables — no hardcoded
config files.

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
