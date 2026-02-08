package models

type Config struct {
	Providers        ProvidersConfig         `json:"providers"`
	GenericProviders []GenericProviderConfig `json:"generic_providers"`
	Search           SearchConfig            `json:"search"`
	Behavior         BehaviorConfig          `json:"behavior"`
	Power            PowerConfig             `json:"power"`
	Paths            PathsConfig             `json:"paths"`
	UI               UIConfig                `json:"ui"`
	Limits           LimitsConfig            `json:"limits"`
	DryRun           bool                    `json:"dry_run,omitempty"`
}

type ProvidersConfig struct {
	Wallhaven WallhavenConfig `json:"wallhaven"`
	Reddit    RedditConfig    `json:"reddit"`
	Nasa      NasaConfig      `json:"nasa"`
	Bing      BingConfig      `json:"bing"`
	Unsplash  UnsplashConfig  `json:"unsplash"`
}

type WallhavenConfig struct {
	Enabled   bool            `json:"enabled"`
	APIKey    string          `json:"api_key"`
	RateLimit RateLimitConfig `json:"rate_limit"`
}

type UnsplashConfig struct {
	Enabled bool   `json:"enabled"`
	APIKey  string `json:"api_key"`
}

type RedditConfig struct {
	Enabled   bool   `json:"enabled"`
	Subreddit string `json:"subreddit"`
	Sort      string `json:"sort"`
	Limit     int    `json:"limit"`
}

type NasaConfig struct {
	Enabled bool   `json:"enabled"`
	APIKey  string `json:"api_key"`
}

type BingConfig struct {
	Enabled bool   `json:"enabled"`
	Market  string `json:"market"` // e.g. "en-US", "es-ES"
}

type GenericProviderConfig struct {
	Name            string          `json:"name"`
	Enabled         bool            `json:"enabled"`
	APIURL          string          `json:"api_url,omitempty"`
	APIKey          string          `json:"api_key,omitempty"`
	ResponseMapping ResponseMapping `json:"-"`
}

type ResponseMapping struct {
	ResultsPath   string `json:"results_path"`
	URLPath       string `json:"url_path"`
	ThumbnailPath string `json:"thumbnail_path"`
	IDPath        string `json:"id_path"`
	DimensionPath string `json:"dimension_path"`
}

type RateLimitConfig struct {
	Requests   int `json:"requests"`
	PerSeconds int `json:"per_seconds"`
}

type SearchConfig struct {
	MinWidth    int     `json:"min_width"`
	MinHeight   int     `json:"min_height"`
	AspectRatio string  `json:"aspect_ratio"`
	Tolerance   float64 `json:"tolerance"`
}

type BehaviorConfig struct {
	Theme                 string `json:"theme"`
	ChangeInterval        int    `json:"change_interval"`
	MultiMonitor          string `json:"multi_monitor"`
	AutoDownload          bool   `json:"auto_download"`
	RespectDarkMode       bool   `json:"respect_dark_mode"`
	SaveFavoritesToFolder bool   `json:"save_favorites_to_folder"`
	FromFavorites         bool   `json:"from_favorites"`
}

type PowerConfig struct {
	BatteryMultiplier   int  `json:"battery_multiplier"`
	PauseOnLowBattery   bool `json:"pause_on_low_battery"`
	LowBatteryThreshold int  `json:"low_battery_threshold"`
}

type PathsConfig struct {
	Wallpapers      string `json:"wallpapers"`
	UseSystemDir    bool   `json:"use_system_dir"`
	IndexWallpapers bool   `json:"index_wallpapers"`
}

type UIConfig struct {
	ShowColors   bool `json:"show_colors"`
	ItemsPerPage int  `json:"items_per_page"`
	ImagePreview bool `json:"image_preview"`
}

type LimitsConfig struct {
	FeedSoftLimit     int `json:"feed_soft_limit"`
	FeedHardLimit     int `json:"feed_hard_limit"`
	RateLimitRequests int `json:"rate_limit_requests"`
	RateLimitPeriod   int `json:"rate_limit_period"`
}
