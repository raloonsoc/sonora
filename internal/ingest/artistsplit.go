package ingest

import (
	"regexp"
	"strings"
)

var featuringPattern = regexp.MustCompile(`(?i)\s+(feat\.|ft\.|featuring)\s+`)

func SplitArtistNames(artist string) []string {
	var splitedArtists []string

	filteredName := featuringPattern.ReplaceAllString(artist, " & ")

	normalizeName := strings.ReplaceAll(filteredName, ",", " & ")

	namesSplited := strings.Split(normalizeName, "&")

	for _, name := range namesSplited {
		nameTrimmed := strings.TrimSpace(name)
		if nameTrimmed != "" {
			splitedArtists = append(splitedArtists, nameTrimmed)
		}
	}

	return splitedArtists
}
