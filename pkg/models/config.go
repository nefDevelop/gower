package models

import (
	"fmt"
	"strconv"
	"strings"
)

type Config struct {
	Providers        ProvidersConfig         `json:"providers"`
	GenericProviders []GenericProviderConfig `json:"generic_providers"`
	Search           SearchConfig            `json:"search"`
	Behavior         BehaviorConfig          `json:"behavior"`
	Paths            PathsConfig             `json:"paths"`
	UI               UIConfig                `json:"ui"`
	Limits           LimitsConfig            `json:"limits"`
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
	Theme                 string           `json:"theme"`
	ChangeInterval        int              `json:"change_interval"`
	MultiMonitor          string           `json:"multi_monitor"`
	AutoDownload          bool             `json:"auto_download"`
	RespectDarkMode       bool             `json:"respect_dark_mode"`
	SaveFavoritesToFolder bool             `json:"save_favorites_to_folder"`
	FromFavorites         bool             `json:"from_favorites"`
	Transition            TransitionConfig `json:"transition"`
}

// TransitionConfig controla la animación al cambiar de fondo. Solo los
// backends awww/swww la aplican; el resto ignora estos valores.
type TransitionConfig struct {
	Type     string  `json:"type"`     // none|simple|fade|left|right|top|bottom|wipe|wave|grow|center|any|outer|random
	Duration float64 `json:"duration"` // segundos
	FPS      int     `json:"fps"`
	Step     int     `json:"step"`
	Angle    int     `json:"angle"`
	Pos      string  `json:"pos"`
	Bezier   string  `json:"bezier"`
	Wave     string  `json:"wave"`
	Engine   string  `json:"engine"` // awww (nativo) | roserlay (crossfade propio)
	Linger   int     `json:"linger"` // ms que roserlay mantiene el overlay tras on-complete
	Mask     string  `json:"mask"`   // crossfade|wipe|radial|random|color|edges|luminance|hue (vacío = derivar de type)
	Feather  float64 `json:"feather"`
	Invert   bool    `json:"invert"`
	Layer    string  `json:"layer"` // background|bottom|top|overlay (vacío = default de roserlay)
}

// transitionTypes son los valores aceptados por awww/swww.
var transitionTypes = map[string]bool{
	"none": true, "simple": true, "fade": true,
	"left": true, "right": true, "top": true, "bottom": true,
	"wipe": true, "wave": true, "grow": true, "center": true,
	"any": true, "outer": true, "random": true,
}

type PathsConfig struct {
	Wallpapers      string `json:"wallpapers"`
	IndexWallpapers bool   `json:"index_wallpapers"`
}

type UIConfig struct {
	ShowColors   bool `json:"show_colors"`
	ItemsPerPage int  `json:"items_per_page"`
}

type LimitsConfig struct {
	FeedSoftLimit    int `json:"feed_soft_limit"`
	FeedHardLimit    int `json:"feed_hard_limit"`
	RateLimitPeriod  int `json:"rate_limit_period"`
	LogRetentionDays int `json:"log_retention_days,omitempty"`
	AnalysisWorkers  int `json:"analysis_workers,omitempty"`
}

// Normalize rellena con valores por defecto los campos que, con cero, harían
// fallar al daemon o desactivarían límites sin avisar.
func (c *Config) Normalize() {
	if c.Behavior.ChangeInterval <= 0 {
		c.Behavior.ChangeInterval = 30
	}
	if c.Behavior.MultiMonitor == "" {
		c.Behavior.MultiMonitor = "clone"
	}
	if c.UI.ItemsPerPage <= 0 {
		c.UI.ItemsPerPage = 10
	}
	if c.Limits.FeedSoftLimit <= 0 {
		c.Limits.FeedSoftLimit = 400
	}
	if c.Limits.FeedHardLimit <= 0 {
		c.Limits.FeedHardLimit = 2000
	}
	if c.Limits.AnalysisWorkers <= 0 {
		c.Limits.AnalysisWorkers = 5
	}
	if c.Search.Tolerance <= 0 {
		c.Search.Tolerance = 0.05
	}
	if c.Behavior.Transition.Type == "" {
		// 'simple' es el default de awww/swww y reproduce el comportamiento
		// anterior, en el que no se pasaba ningún flag.
		c.Behavior.Transition.Type = "simple"
	}
	c.Behavior.Transition.Type = strings.ToLower(strings.TrimSpace(c.Behavior.Transition.Type))
	if c.Behavior.Transition.Engine == "" {
		c.Behavior.Transition.Engine = "auto"
	}
	c.Behavior.Transition.Engine = strings.ToLower(strings.TrimSpace(c.Behavior.Transition.Engine))
	if c.Behavior.Transition.Linger < 0 {
		c.Behavior.Transition.Linger = 0
	}
	c.Behavior.Transition.Mask = strings.ToLower(strings.TrimSpace(c.Behavior.Transition.Mask))
	c.Behavior.Transition.Layer = strings.ToLower(strings.TrimSpace(c.Behavior.Transition.Layer))
	if c.Behavior.Transition.Feather < 0 {
		c.Behavior.Transition.Feather = 0
	}
	if c.Behavior.Transition.Feather > 1 {
		c.Behavior.Transition.Feather = 1
	}
	if c.Behavior.Transition.Duration < 0 {
		c.Behavior.Transition.Duration = 0
	}
	if c.Behavior.Transition.FPS < 0 {
		c.Behavior.Transition.FPS = 0
	}
	if c.Behavior.Transition.Step < 0 {
		c.Behavior.Transition.Step = 0
	}
}

// Validate comprueba los valores que consumen el CLI y el daemon. Un valor
// inválido debe rechazarse al escribirlo, no provocar un panic más tarde.
func (c *Config) Validate() error {
	switch strings.ToLower(c.Behavior.Theme) {
	case "", "auto", "dark", "light":
	default:
		return fmt.Errorf("invalid behavior.theme %q: expected auto, dark or light", c.Behavior.Theme)
	}

	switch c.Behavior.MultiMonitor {
	case "", "clone", "distinct":
	default:
		return fmt.Errorf("invalid behavior.multi_monitor %q: expected clone or distinct", c.Behavior.MultiMonitor)
	}

	if c.Behavior.ChangeInterval <= 0 {
		return fmt.Errorf("behavior.change_interval must be greater than zero")
	}
	if c.Search.MinWidth < 0 || c.Search.MinHeight < 0 {
		return fmt.Errorf("search minimum dimensions must be positive")
	}
	if c.UI.ItemsPerPage < 0 {
		return fmt.Errorf("ui.items_per_page must be positive")
	}
	if c.Limits.FeedSoftLimit < 0 || c.Limits.FeedHardLimit < 0 {
		return fmt.Errorf("limits must be positive")
	}
	if c.Limits.FeedHardLimit > 0 && c.Limits.FeedSoftLimit > c.Limits.FeedHardLimit {
		return fmt.Errorf("limits.feed_soft_limit cannot exceed limits.feed_hard_limit")
	}
	if c.Limits.AnalysisWorkers < 0 {
		return fmt.Errorf("limits.analysis_workers must be positive")
	}

	if c.Search.AspectRatio != "" {
		parts := strings.Split(strings.TrimSpace(c.Search.AspectRatio), ":")
		if len(parts) != 2 {
			return fmt.Errorf("invalid search.aspect_ratio %q: expected W:H", c.Search.AspectRatio)
		}
		w, errW := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		h, errH := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if errW != nil || errH != nil || w <= 0 || h <= 0 {
			return fmt.Errorf("invalid search.aspect_ratio %q: values must be positive numbers", c.Search.AspectRatio)
		}
	}

	for _, p := range c.GenericProviders {
		if p.Name == "" {
			return fmt.Errorf("generic provider without name")
		}
		if p.Enabled && p.APIURL == "" {
			return fmt.Errorf("generic provider %q is enabled but has no api_url", p.Name)
		}
	}

	if err := c.Behavior.Transition.validate(); err != nil {
		return err
	}

	return nil
}

func (t TransitionConfig) validate() error {
	if t.Type != "" && !transitionTypes[t.Type] {
		return fmt.Errorf("invalid behavior.transition.type %q: expected one of none, simple, fade, left, right, top, bottom, wipe, wave, grow, center, any, outer, random", t.Type)
	}
	if t.Duration < 0 || t.Duration > 60 {
		return fmt.Errorf("behavior.transition.duration must be between 0 and 60 seconds")
	}
	if t.FPS < 0 || t.FPS > 240 {
		return fmt.Errorf("behavior.transition.fps must be between 0 and 240")
	}
	if t.Step < 0 || t.Step > 255 {
		return fmt.Errorf("behavior.transition.step must be between 0 and 255")
	}
	if t.Angle < 0 || t.Angle > 360 {
		return fmt.Errorf("behavior.transition.angle must be between 0 and 360 degrees")
	}
	switch t.Engine {
	case "", "auto", "awww", "roserlay":
	default:
		return fmt.Errorf("invalid behavior.transition.engine %q: expected auto, awww or roserlay", t.Engine)
	}
	if t.Linger < 0 || t.Linger > 5000 {
		return fmt.Errorf("behavior.transition.linger must be between 0 and 5000 ms")
	}
	switch t.Mask {
	case "", "crossfade", "wipe", "radial", "drops", "random", "color", "edges", "luminance", "hue", "saliency", "diff":
	default:
		return fmt.Errorf("invalid behavior.transition.mask %q: expected crossfade, wipe, radial, drops, random, color, edges, luminance, hue, saliency or diff", t.Mask)
	}
	if t.Feather < 0 || t.Feather > 1 {
		return fmt.Errorf("behavior.transition.feather must be between 0 and 1")
	}
	switch t.Layer {
	case "", "background", "bottom", "top", "overlay":
	default:
		return fmt.Errorf("invalid behavior.transition.layer %q: expected background, bottom, top or overlay", t.Layer)
	}
	return nil
}
