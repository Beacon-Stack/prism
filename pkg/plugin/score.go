package plugin

// ScoreBreakdown records how a release was evaluated against a quality profile.
// Each dimension is independently scored; Total is the sum.
type ScoreBreakdown struct {
	Total      int              `json:"total"`
	Dimensions []ScoreDimension `json:"dimensions"`
}

// ScoreDimension is one component of a ScoreBreakdown.
type ScoreDimension struct {
	Name    string `json:"name"`    // "resolution", "source", "codec", "hdr"
	Score   int    `json:"score"`   // points awarded for this dimension
	Max     int    `json:"max"`     // maximum possible for this dimension
	Matched bool   `json:"matched"` // did it meet the profile requirement?
	Got     string `json:"got"`     // what we found (e.g. "x264")
	Want    string `json:"want"`    // what the profile requires (e.g. "x265")
}
