package ingest

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/raloonsoc/sonora/internal/db/sqlc"
)

func ProcessFile(ctx context.Context, path string, queries *sqlc.Queries, coverArtDir string) error {
	output, err := probeFile(path)
	if err != nil {
		return fmt.Errorf("ingest: processing %s: %w", path, err)
	}

	track, err := trackFromProbe(path, output)
	if err != nil {
		return fmt.Errorf("ingest: normalizing %s: %w", path, err)
	}

	artistName := output.Format.Tags["artist"]
	if artistName == "" {
		return fmt.Errorf("ingest: %s missing artist tag", path)
	}

	artistNames := SplitArtistNames(artistName)

	var trackArtists []sqlc.Artist
	for _, name := range artistNames {
		trackArtist, err := queries.GetArtistByName(ctx, name)
		if err != nil {
			trackArtist, err = queries.CreateArtist(ctx, name)
			if err != nil {
				return fmt.Errorf("ingest: creating artist %q: %w", name, err)
			}
		}
		trackArtists = append(trackArtists, trackArtist)
	}

	albumArtistName := output.Format.Tags["album_artist"]
	if albumArtistName == "" {
		albumArtistName = artistName
	}

	albumArtist, err := queries.GetArtistByName(ctx, albumArtistName)
	if err != nil {
		albumArtist, err = queries.CreateArtist(ctx, albumArtistName)
		if err != nil {
			return fmt.Errorf("ingest: creating album artist %q: %w", albumArtistName, err)
		}
	}

	albumTitle := output.Format.Tags["album"]
	if albumTitle == "" {
		return fmt.Errorf("ingest: %s missing album tag", path)
	}

	dateTag := output.Format.Tags["date"]
	var releaseYear pgtype.Int4
	if len(dateTag) >= 4 {
		if year, err := strconv.Atoi(dateTag[:4]); err == nil {
			releaseYear = pgtype.Int4{Int32: int32(year), Valid: true}
		}
	}

	album, err := queries.GetAlbumByTitleAndArtist(ctx, sqlc.GetAlbumByTitleAndArtistParams{
		Title:    albumTitle,
		ArtistID: albumArtist.ID,
	})
	if err != nil {
		album, err = queries.CreateAlbum(ctx, sqlc.CreateAlbumParams{
			Title:        albumTitle,
			ArtistID:     albumArtist.ID,
			ReleaseYear:  releaseYear,
			CoverArtPath: "",
		})
		if err != nil {
			return fmt.Errorf("ingest: creating album %q: %w", albumTitle, err)
		}
	}

	var replayGain *float64
	if loudnormResult, err := loudnormAnalyze(path); err == nil {
		replayGain, _ = replayGainFromLoudnorm(loudnormResult)
	}

	var replayGainDB pgtype.Float8
	if replayGain != nil {
		replayGainDB = pgtype.Float8{Float64: *replayGain, Valid: true}
	}

	coverPath := filepath.Join(coverArtDir, album.ID.String()+".jpg")
	if err := extractCoverArt(path, coverPath); err != nil {
		fallbackCover, err := FetchCoverArtURL(ctx, albumArtistName, albumTitle)
		if err != nil {
			slog.Error("ingest: getting fallback cover art failed", "error", err)
		}
		if fallbackCover == "" {
			slog.Error("ingest: fallback cover art not found")
		}
		replacedStr := strings.Replace(fallbackCover, "100x100bb.jpg", "600x600bb.jpg", 1)
		if replacedStr != "" {
			if err := DownloadCoverArt(ctx, replacedStr, coverPath); err != nil {
				slog.Error("ingest: downloading fallback cover art failed", "error", err)
			} else if err := queries.UpdateAlbumCoverArt(ctx, sqlc.UpdateAlbumCoverArtParams{
				ID:           album.ID,
				CoverArtPath: coverPath,
			}); err != nil {
				slog.Error("ingest: saving fallback cover art path failed", "error", err)
			}
		}
	} else if err := queries.UpdateAlbumCoverArt(ctx, sqlc.UpdateAlbumCoverArtParams{
		ID:           album.ID,
		CoverArtPath: coverPath,
	}); err != nil {
		slog.Error("ingest: saving cover art path failed", "error", err)
	}

	var bitDepth, sampleRate, channels int32
	for _, s := range output.Streams {
		if s.CodecType == "audio" {
			bd, _ := strconv.Atoi(s.BitsPerRawSample)
			sr, _ := strconv.Atoi(s.SampleRate)
			bitDepth = int32(bd)
			sampleRate = int32(sr)
			channels = int32(s.Channels)
			break
		}
	}

	fingerprint, err := FingerprintFile(path)
	if err != nil {
		slog.Warn("ingest: fingerprinting failed", "path", path, "error", err)
		fingerprint = ""
	} else if fingerprint != "" {
		if existing, err := queries.GetTrackByFingerprint(ctx, fingerprint); err == nil {
			slog.Warn("ingest: possible duplicate track detected", "path", path, "duplicated_of", existing.Path)
		}
	}
	createdTrack, err := queries.CreateTrack(ctx, sqlc.CreateTrackParams{
		Title:             track.Title,
		AlbumID:           album.ID,
		ArtistID:          trackArtists[0].ID,
		Genre:             track.Genre,
		TrackNumber:       int32(track.TrackNumber),
		DiscNumber:        int32(track.DiscNumber),
		DurationSeconds:   int32(track.DurationSeconds),
		Path:              path,
		Format:            track.Format,
		ReplayGainTrackDb: replayGainDB,
		BitDepth:          bitDepth,
		SampleRate:        sampleRate,
		Channels:          channels,
		BitRate:           int32(track.BitRate),
		SizeBytes:         track.SizeBytes,
		Fingerprint:       fingerprint,
	})

	if err != nil {
		return fmt.Errorf("ingest: creating track %s: %w", path, err)
	}

	for i, a := range trackArtists {
		if err := queries.CreateTrackArtist(ctx, sqlc.CreateTrackArtistParams{
			TrackID:  createdTrack.ID,
			ArtistID: a.ID,
			Position: int32(i),
		}); err != nil {
			return fmt.Errorf("ingest: linking artist %q to track %s: %w", a.Name, path, err)
		}
	}

	return nil
}
