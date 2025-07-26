package types

// Typed metadata structs for each event type

type StreamRequestToolMetadata struct {
	DefaultWorkingDir string `json:"default_working_dir"`
}

type StreamResponseToolMetadata struct {
	OK bool `json:"ok"`
}

// CacheInfoMetadata represents metadata for cache_info events
type CacheInfoMetadata struct {
	CacheEnabled bool   `json:"cache_enabled"`
	Model        string `json:"model,omitempty"`
}

type RoundStartMetadata struct {
	MaxRounds int `json:"max_rounds"`
}

type RoundEndMetadata struct {
	Round int `json:"round"`
}
