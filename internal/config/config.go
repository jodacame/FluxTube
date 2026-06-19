// Package config holds persisted settings, the video library and rules, backed
// by a single bbolt database.
package config

// Settings is the full, persisted configuration. It can also be seeded from
// environment variables at startup.
type Settings struct {
	Cache     CacheSettings     `json:"cache"`
	Quality   QualitySettings   `json:"quality"`
	Net       NetSettings       `json:"net"`
	YouTube   YouTubeSettings   `json:"youtube"`
	Discovery DiscoverySettings `json:"discovery"`
	Limits    LimitSettings     `json:"limits"`
	Music     MusicSettings     `json:"music"`
	APIToken  string            `json:"apiToken"`
}

// MusicSettings controls the intelligent music handling.
type MusicSettings struct {
	// AutoSave persists the audio (and saves to the library as music)
	// automatically whenever a played video is detected as music.
	AutoSave bool `json:"autoSave"`
	// Dir is the persistent directory where music audio files are stored.
	Dir string `json:"dir"`
}

// CacheSettings controls the ephemeral segment cache.
type CacheSettings struct {
	Mode           string `json:"mode"` // "ephemeral" | "cache"
	MaxSizeMB      int    `json:"maxSizeMB"`
	SegmentSeconds int    `json:"segmentSeconds"`
	Path           string `json:"path"`
}

// QualitySettings controls default track selection.
type QualitySettings struct {
	DefaultMaxHeight int      `json:"defaultMaxHeight"`
	PreferAudioLang  string   `json:"preferAudioLang"`
	PreferSubLang    string   `json:"preferSubLang"`
	AutoSubLangs     []string `json:"autoSubLangs"`
}

// NetSettings controls the HTTP listener.
type NetSettings struct {
	ListenHost string `json:"listenHost"`
	ListenPort int    `json:"listenPort"`
}

// YouTubeSettings controls extraction behaviour.
type YouTubeSettings struct {
	CookiesFile  string `json:"cookiesFile"`
	ExtractorArg string `json:"extractorArg"`
}

// DiscoverySettings controls the discovery provider.
type DiscoverySettings struct {
	Provider         string `json:"provider"` // "yt-dlp" | "invidious"
	InvidiousBaseURL string `json:"invidiousBaseUrl"`
	CacheSeconds     int    `json:"cacheSeconds"`
}

// LimitSettings controls runtime limits.
type LimitSettings struct {
	MaxSessions    int `json:"maxSessions"`
	MaxFFmpeg      int `json:"maxFFmpeg"`
	IdleTimeoutSec int `json:"idleTimeoutSec"`
}

// Defaults returns a fully-populated default configuration.
func Defaults() Settings {
	return Settings{
		Cache: CacheSettings{
			Mode:           "ephemeral",
			MaxSizeMB:      2048,
			SegmentSeconds: 6,
			Path:           "/cache",
		},
		Quality: QualitySettings{
			DefaultMaxHeight: 1080,
			AutoSubLangs:     []string{"en", "es"},
		},
		Net: NetSettings{
			ListenHost: "0.0.0.0",
			ListenPort: 7002,
		},
		Discovery: DiscoverySettings{
			Provider:     "yt-dlp",
			CacheSeconds: 600,
		},
		Limits: LimitSettings{
			MaxSessions:    8,
			MaxFFmpeg:      4,
			IdleTimeoutSec: 180,
		},
		Music: MusicSettings{AutoSave: true, Dir: "/config/music"},
	}
}

// Entry is a video saved in the library with cheap catalog metadata. Live
// playback state is computed at request time, not persisted.
type Entry struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Channel   string `json:"channel"`
	ChannelID string `json:"channelId"`
	Thumbnail string `json:"thumbnail"`
	Duration  int    `json:"duration"`
	AddedAt   int64  `json:"addedAt"`
	Kind      string `json:"kind"` // "video" | "music"
}
