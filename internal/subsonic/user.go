package subsonic

import (
	"net/http"
)

// License types
type licenseSubsonicResponse struct {
	baseResponse
	License licenseEntry `json:"license" xml:"license"`
}

type licenseEntry struct {
	Valid bool `json:"valid" xml:"valid,attr"`
}

// MusicFolder types
type musicFoldersSubsonicResponse struct {
	baseResponse
	MusicFolders musicFoldersEntry `json:"musicFolders" xml:"musicFolders"`
}

type musicFoldersEntry struct {
	MusicFolder []musicFolderItem `json:"musicFolder" xml:"musicFolder"`
}

type musicFolderItem struct {
	ID   int    `json:"id" xml:"id,attr"`
	Name string `json:"name" xml:"name,attr"`
}

// OpenSubsonic extensions types
type extensionsSubsonicResponse struct {
	baseResponse
	OpenSubsonicExtensions []extensionItem `json:"openSubsonicExtensions" xml:"openSubsonicExtensions"`
}

type extensionItem struct {
	Name     string `json:"name" xml:"name,attr"`
	Versions []int  `json:"versions" xml:"versions"`
}

// User types
type userSubsonicResponse struct {
	baseResponse
	User userEntry `json:"user" xml:"user"`
}

type userEntry struct {
	Username          string `json:"username" xml:"username,attr"`
	AdminRole         bool   `json:"adminRole" xml:"adminRole,attr"`
	SettingsRole      bool   `json:"settingsRole" xml:"settingsRole,attr"`
	StreamRole        bool   `json:"streamRole" xml:"streamRole,attr"`
	DownloadRole      bool   `json:"downloadRole" xml:"downloadRole,attr"`
	PlaylistRole      bool   `json:"playlistRole" xml:"playlistRole,attr"`
	CoverArtRole      bool   `json:"coverArtRole" xml:"coverArtRole,attr"`
	UploadRole        bool   `json:"uploadRole" xml:"uploadRole,attr"`
	PodcastRole       bool   `json:"podcastRole" xml:"podcastRole,attr"`
	CommentRole       bool   `json:"commentRole" xml:"commentRole,attr"`
	JukeboxRole       bool   `json:"jukeboxRole" xml:"jukeboxRole,attr"`
	ShareRole         bool   `json:"shareRole" xml:"shareRole,attr"`
	ScrobblingEnabled bool   `json:"scrobblingEnabled" xml:"scrobblingEnabled,attr"`
}

func (h *Handler) GetLicenseHandler(w http.ResponseWriter, r *http.Request) {
	encodeResponse(w, r, licenseSubsonicResponse{
		baseResponse: newBaseResponse(),
		License:      licenseEntry{Valid: true},
	})
}

func (h *Handler) GetMusicFoldersHandler(w http.ResponseWriter, r *http.Request) {
	encodeResponse(w, r, musicFoldersSubsonicResponse{
		baseResponse: newBaseResponse(),
		MusicFolders: musicFoldersEntry{
			MusicFolder: []musicFolderItem{
				{ID: 1, Name: "Music"},
			},
		},
	})
}

func (h *Handler) GetOpenSubsonicExtensionsHandler(w http.ResponseWriter, r *http.Request) {
	encodeResponse(w, r, extensionsSubsonicResponse{
		baseResponse:           newBaseResponse(),
		OpenSubsonicExtensions: []extensionItem{},
	})
}

func (h *Handler) GetUserHandler(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	user, err := h.Queries.GetUserByUsername(r.Context(), username)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	userEntrie := userEntry{
		Username:          username,
		AdminRole:         user.IsAdmin,
		SettingsRole:      user.IsAdmin,
		StreamRole:        true,
		DownloadRole:      true,
		PlaylistRole:      true,
		CoverArtRole:      true,
		ScrobblingEnabled: true,
		UploadRole:        false,
		PodcastRole:       false,
		CommentRole:       false,
		JukeboxRole:       false,
		ShareRole:         false,
	}

	encodeResponse(w, r, userSubsonicResponse{
		baseResponse: newBaseResponse(),
		User:         userEntrie,
	})
}
