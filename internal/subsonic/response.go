package subsonic

import (
	"encoding/json"
	"encoding/xml"
	"net/http"
)

// baseResponse is embedded in every "...SubsonicResponse" struct passed to
// encodeResponse. The type/serverVersion/openSubsonic fields identify
// Sonora as an OpenSubsonic-conformant server — some clients (e.g. Feishin)
// rely on their presence to decide how to parse responses. XMLName/Xmlns
// only matter for the XML encoding path.
type baseResponse struct {
	XMLName xml.Name `json:"-" xml:"subsonic-response"`
	Xmlns   string   `json:"-" xml:"xmlns,attr"`
	Status  string   `json:"status" xml:"status,attr"`
	Version string   `json:"version" xml:"version,attr"`
	Type    string   `json:"type" xml:"type,attr"`
	// ServerVersion identifies Sonora's own release (OpenSubsonic field).
	// ServerAPIVersion is the same API version as Version, under the
	// legacy Subsonic attribute name some clients (e.g. Amperfy) still
	// read to negotiate the API version before streaming.
	ServerVersion    string `json:"serverVersion" xml:"serverVersion,attr"`
	ServerAPIVersion string `json:"serverApiVersion" xml:"serverApiVersion,attr"`
	OpenSubsonic     bool   `json:"openSubsonic" xml:"openSubsonic,attr"`
}

func newBaseResponse() baseResponse {
	return baseResponse{
		Xmlns:            "http://subsonic.org/restapi",
		Status:           "ok",
		Version:          "1.16.1",
		Type:             "sonora",
		ServerVersion:    "0.1.0",
		ServerAPIVersion: "1.16.1",
		OpenSubsonic:     true,
	}
}

// encodeResponse writes resp (a struct embedding baseResponse) to w.
// JSON is used when the client asked for it via f=json (the OpenSubsonic
// default in practice); XML otherwise — XML is the original Subsonic
// protocol's default when f is omitted, and clients like Amperfy rely on
// that default. The "subsonic-response" wrapping key is added here for
// JSON; for XML it comes from baseResponse's XMLName.
func encodeResponse(w http.ResponseWriter, r *http.Request, resp any) {
	if r.URL.Query().Get("f") == "json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"subsonic-response": resp})
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	w.Write([]byte(xml.Header))
	xml.NewEncoder(w).Encode(resp)
}
