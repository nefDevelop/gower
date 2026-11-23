package models

import (
	"strings"
	"testing"
)

func TestConfigNormalizeTransitionDefault(t *testing.T) {
	var c Config
	c.Normalize()

	if c.Behavior.Transition.Type != "simple" {
		t.Errorf("expected default transition type 'simple', got %q", c.Behavior.Transition.Type)
	}
	if c.Behavior.Transition.Engine != "auto" {
		t.Errorf("expected default transition engine 'auto', got %q", c.Behavior.Transition.Engine)
	}
}

func TestTransitionConfigValidate(t *testing.T) {
	base := func() Config {
		c := Config{}
		c.Normalize()
		return c
	}

	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr string
	}{
		{
			name: "valid wave",
			mutate: func(c *Config) {
				c.Behavior.Transition = TransitionConfig{Type: "wave", Duration: 1.5, FPS: 60, Angle: 45}
			},
		},
		{
			name:    "unknown type",
			mutate:  func(c *Config) { c.Behavior.Transition.Type = "explode" },
			wantErr: "invalid behavior.transition.type",
		},
		{
			name:    "negative duration",
			mutate:  func(c *Config) { c.Behavior.Transition.Duration = -1 },
			wantErr: "behavior.transition.duration",
		},
		{
			name:    "too high fps",
			mutate:  func(c *Config) { c.Behavior.Transition.FPS = 500 },
			wantErr: "behavior.transition.fps",
		},
		{
			name:    "step out of range",
			mutate:  func(c *Config) { c.Behavior.Transition.Step = 999 },
			wantErr: "behavior.transition.step",
		},
		{
			name:    "angle out of range",
			mutate:  func(c *Config) { c.Behavior.Transition.Angle = 999 },
			wantErr: "behavior.transition.angle",
		},
		{
			name:   "roserlay engine",
			mutate: func(c *Config) { c.Behavior.Transition = TransitionConfig{Engine: "roserlay", Linger: 500} },
		},
		{
			name:   "auto engine",
			mutate: func(c *Config) { c.Behavior.Transition.Engine = "auto" },
		},
		{
			name:    "unknown engine",
			mutate:  func(c *Config) { c.Behavior.Transition.Engine = "magia" },
			wantErr: "behavior.transition.engine",
		},
		{
			name:    "linger out of range",
			mutate:  func(c *Config) { c.Behavior.Transition.Linger = 9000 },
			wantErr: "behavior.transition.linger",
		},
		{
			name:   "valid mask and feather",
			mutate: func(c *Config) { c.Behavior.Transition.Mask = "wipe"; c.Behavior.Transition.Feather = 0.2 },
		},
		{
			name:   "valid content mask",
			mutate: func(c *Config) { c.Behavior.Transition.Mask = "color" },
		},
		{
			name:   "valid saliency mask",
			mutate: func(c *Config) { c.Behavior.Transition.Mask = "saliency" },
		},
		{
			name:   "valid diff mask",
			mutate: func(c *Config) { c.Behavior.Transition.Mask = "diff" },
		},
		{
			name:   "valid layer",
			mutate: func(c *Config) { c.Behavior.Transition.Layer = "overlay" },
		},
		{
			name:    "invalid layer",
			mutate:  func(c *Config) { c.Behavior.Transition.Layer = "middle" },
			wantErr: "behavior.transition.layer",
		},
		{
			name:    "unknown mask",
			mutate:  func(c *Config) { c.Behavior.Transition.Mask = "checker" },
			wantErr: "behavior.transition.mask",
		},
		{
			name:    "feather out of range",
			mutate:  func(c *Config) { c.Behavior.Transition.Feather = 2 },
			wantErr: "behavior.transition.feather",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := base()
			tc.mutate(&c)
			err := c.Validate()
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}
