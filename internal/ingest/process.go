package ingest

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"

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

	artist, err := queries.GetArtistByName(ctx, artistName)
	if err != nil {
		artist, err = queries.CreateArtist(ctx, artistName)
		if err != nil {
			return fmt.Errorf("ingest: creating artist %q: %w", artistName, err)
		}
	}

	albumTitle := output.Format.Tags["album"]
	if albumTitle == "" {
		return fmt.Errorf("ingest: %s missing album tag", path)
	}

	album, err := queries.GetAlbumByTitleAndArtist(ctx, sqlc.GetAlbumByTitleAndArtistParams{
		Title:    albumTitle,
		ArtistID: artist.ID,
	})
	if err != nil {
		album, err = queries.CreateAlbum(ctx, sqlc.CreateAlbumParams{
			Title:        albumTitle,
			ArtistID:     artist.ID,
			ReleaseYear:  pgtype.Int4{},
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
	if err := extractCoverArt(path, coverPath); err == nil {
		queries.UpdateAlbumCoverArt(ctx, sqlc.UpdateAlbumCoverArtParams{
			ID:           album.ID,
			CoverArtPath: coverPath,
		})
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

	_, err = queries.CreateTrack(ctx, sqlc.CreateTrackParams{
		Title:             track.Title,
		AlbumID:           album.ID,
		ArtistID:          artist.ID,
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
	})

	if err != nil {
		return fmt.Errorf("ingest: creating track %s: %w", path, err)
	}
	return nil
}
