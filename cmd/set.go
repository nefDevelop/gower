// cmd/set.go
package cmd

import (
	"context"
	"fmt"
	"math/rand"
	"net/url"
	"os/exec"
	"strings"
	"time"

	"gower/internal/core"
	"gower/pkg/models"

	"github.com/spf13/cobra"
)

var (
	setID            string
	setURL           string
	setRandom        bool
	setTheme         string
	setFromFavorites bool
	setMultiMonitor  string
	setCommand       string
	setTargetMonitor string

	setTransitionType     string
	setTransitionDuration float64
	setTransitionFPS      int
	setTransitionStep     int
	setTransitionAngle    int
	setTransitionPos      string
	setTransitionBezier   string
	setTransitionWave     string
	setTransitionEngine   string
	setTransitionLinger   int
	setTransitionMask     string
	setTransitionFeather  float64
	setTransitionInvert   bool
	setTransitionLayer    string
)

var setCmd = &cobra.Command{
	Use:   "set [id|url|random]",
	Short: "Set wallpaper",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runSet,
}

func init() {
	setCmd.Flags().StringVar(&setID, "id", "",
		"wallpaper ID (e.g., wh_123456)")
	setCmd.Flags().StringVar(&setURL, "url", "",
		"direct wallpaper URL")
	setCmd.Flags().BoolVarP(&setRandom, "random", "r", false,
		"set random wallpaper")
	setCmd.Flags().StringVar(&setTheme, "theme", "",
		"theme filter [dark|light|auto]")
	setCmd.Flags().BoolVar(&setFromFavorites, "from-favorites", false,
		"random from favorites only")
	setCmd.Flags().StringVar(&setMultiMonitor, "multi-monitor", "",
		"multi-monitor mode [clone|distinct]")
	setCmd.Flags().StringVar(&setCommand, "command", "",
		"custom wallpaper command")
	setCmd.Flags().StringVar(&setTargetMonitor, "target-monitor", "",
		"set wallpaper on a specific monitor (e.g., 'eDP-1')")

	setRandomCmd.Flags().StringVar(&setTargetMonitor, "target-monitor", "", "set wallpaper on a specific monitor (e.g., 'eDP-1')")
	setRandomCmd.Flags().StringVar(&setTheme, "theme", "", "theme filter [dark|light|auto]")
	setRandomCmd.Flags().BoolVar(&setFromFavorites, "from-favorites", false, "random from favorites only")
	setRandomCmd.Flags().StringVar(&setMultiMonitor, "multi-monitor", "", "multi-monitor mode [clone|distinct]")

	// Flags persistentes: se heredan en set random y set undo.
	setCmd.PersistentFlags().StringVar(&setTransitionType, "transition-type", "", "transition effect for awww/swww (none|simple|fade|left|right|top|bottom|wipe|wave|grow|center|any|outer|random)")
	setCmd.PersistentFlags().Float64Var(&setTransitionDuration, "transition-duration", 0, "transition duration in seconds")
	setCmd.PersistentFlags().IntVar(&setTransitionFPS, "transition-fps", 0, "transition frame rate")
	setCmd.PersistentFlags().IntVar(&setTransitionStep, "transition-step", 0, "transition step (how fast it approaches the new image)")
	setCmd.PersistentFlags().IntVar(&setTransitionAngle, "transition-angle", 0, "wipe/wave angle in degrees")
	setCmd.PersistentFlags().StringVar(&setTransitionPos, "transition-pos", "", "grow/outer center (e.g. center, 0.5,0.5)")
	setCmd.PersistentFlags().StringVar(&setTransitionBezier, "transition-bezier", "", "fade bezier curve (e.g. .54,0,.34,.99)")
	setCmd.PersistentFlags().StringVar(&setTransitionWave, "transition-wave", "", "wave width and height (e.g. 20,20)")
	setCmd.PersistentFlags().StringVar(&setTransitionEngine, "transition-engine", "", "transition engine [auto|awww|roserlay]")
	setCmd.PersistentFlags().IntVar(&setTransitionLinger, "transition-linger", 0, "roserlay: ms to keep the overlay after committing the image")
	setCmd.PersistentFlags().StringVar(&setTransitionMask, "transition-mask", "", "roserlay mask [crossfade|wipe|radial|drops|random|color|edges|luminance|hue|saliency|diff]")
	setCmd.PersistentFlags().Float64Var(&setTransitionFeather, "transition-feather", 0, "roserlay mask feather (0..1)")
	setCmd.PersistentFlags().BoolVar(&setTransitionInvert, "transition-invert", false, "roserlay: invert reveal order")
	setCmd.PersistentFlags().StringVar(&setTransitionLayer, "transition-layer", "", "roserlay layer [background|bottom|top|overlay]")

	// Subcomandos
	setCmd.AddCommand(setRandomCmd)
	setCmd.AddCommand(setUndoCmd)

	rootCmd.AddCommand(setCmd)
}

var setRandomCmd = &cobra.Command{
	Use:   "random",
	Short: "Set random wallpaper",
	RunE:  runSetRandom,
}

var setUndoCmd = &cobra.Command{
	Use:   "undo",
	Short: "Revert to the previous wallpaper",
	RunE:  runSetUndo,
}

func runSet(cmd *cobra.Command, args []string) error {
	if err := ensureConfig(); err != nil {
		return err
	}

	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}
	controller := core.NewController(cfg)

	var wallpaper *models.Wallpaper
	var wallpapers []models.Wallpaper
	var targetMonitors []core.Monitor
	var targetMonitorIndex = -1

	// 1. Determine target wallpaper
	if setID != "" && setURL != "" {
		return fmt.Errorf("--id and --url cannot be used together")
	}
	if setMultiMonitor != "" && setMultiMonitor != "clone" && setMultiMonitor != "distinct" {
		return fmt.Errorf("invalid --multi-monitor %q: expected clone or distinct", setMultiMonitor)
	}
	if setTheme != "" && setTheme != "dark" && setTheme != "light" && setTheme != "auto" {
		return fmt.Errorf("invalid --theme %q: expected dark, light or auto", setTheme)
	}

	if len(args) > 0 {
		input := args[0]
		if input == "random" {
			return runSetRandom(cmd, args)
		}
		if u, err := url.Parse(input); err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" {
			wallpaper = &models.Wallpaper{
				ID:     "manual_url",
				URL:    input,
				Source: "manual",
			}
		} else {
			wp, err := controller.GetWallpaper(input)
			if err != nil {
				return fmt.Errorf("error: %w", err)
			}
			wallpaper = wp
		}
	} else if setURL != "" {
		u, err := url.Parse(setURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return fmt.Errorf("invalid --url %q: expected an http(s) URL", setURL)
		}
		wallpaper = &models.Wallpaper{
			ID:     "manual_url",
			URL:    setURL,
			Source: "manual",
		}
	} else if setID != "" {
		wp, err := controller.GetWallpaper(setID)
		if err != nil {
			return fmt.Errorf("error: %w", err)
		}
		wallpaper = wp
	} else if setRandom {
		return runSetRandom(cmd, args)
	} else {
		return cmd.Help()
	}

	if wallpaper != nil {
		wallpapers = []models.Wallpaper{*wallpaper}
	}

	if setTargetMonitor != "" {
		changer := core.NewWallpaperChanger("", false) // RespectDarkMode doesn't matter for detection
		allMonitors, err := changer.DetectMonitors()
		if err != nil {
			return fmt.Errorf("error detecting monitors for target-monitor: %w", err)
		}

		found := false
		for i, mon := range allMonitors {
			if mon.ID == setTargetMonitor || mon.Name == setTargetMonitor {
				targetMonitors = []core.Monitor{mon}
				targetMonitorIndex = i
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("monitor '%s' not found. Use 'gower status --monitors' to see available monitors", setTargetMonitor)
		}
	} else {
		// If no target monitor is specified, pass an empty slice, which applyWallpapers will interpret as "all"
		targetMonitors = []core.Monitor{}
	}

	return applyWallpapers(cmd, controller, wallpapers, targetMonitors, cfg, targetMonitorIndex)
}

func runSetRandom(cmd *cobra.Command, args []string) error {
	if err := ensureConfig(); err != nil {
		return err
	}

	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}
	controller := core.NewController(cfg)

	// Auto-detect theme if "auto" or if empty and config enabled. Se usa una
	// variable local para no mutar el flag global entre invocaciones.
	targetTheme := setTheme
	if targetTheme == "" {
		targetTheme = cfg.Behavior.Theme
	}
	if targetTheme == "auto" || (targetTheme == "" && cfg.Behavior.RespectDarkMode) {
		if core.IsSystemInDarkMode() {
			targetTheme = "dark"
		} else {
			targetTheme = "light"
		}
		if config.Debug {
			cmd.Printf("   [DEBUG] Auto-selecting theme: %s\n", targetTheme)
		}
	}

	var wallpapers []models.Wallpaper
	var numWallpapers = 1 // Default to 1 wallpaper
	var monitors []core.Monitor
	var targetMonitorIndex = -1

	mmMode := setMultiMonitor
	if mmMode == "" && cfg != nil {
		mmMode = cfg.Behavior.MultiMonitor
	}

	if setTargetMonitor != "" {
		changer := core.NewWallpaperChanger("", false)
		allMonitors, err := changer.DetectMonitors()
		if err != nil {
			return fmt.Errorf("error detecting monitors for target-monitor: %w", err)
		}

		found := false
		for i, mon := range allMonitors {
			if mon.ID == setTargetMonitor || mon.Name == setTargetMonitor {
				monitors = []core.Monitor{mon}
				targetMonitorIndex = i
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("monitor '%s' not found. Use 'gower status --monitors' to see available monitors", setTargetMonitor)
		}
	} else if mmMode == "distinct" {
		respectDark := true
		if cfg != nil {
			respectDark = cfg.Behavior.RespectDarkMode
		}
		changer := core.NewWallpaperChanger("", respectDark)
		detectedMonitors, err := changer.DetectMonitors()
		if err != nil {
			cmd.Printf("Warning: Could not detect monitors for distinct mode, falling back to single wallpaper: %v\n", err)
			numWallpapers = 1
			monitors = []core.Monitor{{ID: "default", Name: "default", Primary: true}} // Default single monitor
		} else {
			monitors = detectedMonitors
			numWallpapers = len(monitors)
			cmd.Printf("Detected %d monitors for distinct mode.\n", numWallpapers)
		}
	} else {
		// For clone mode or single monitor, we still need a monitor slice for applyWallpapers
		monitors = []core.Monitor{{ID: "default", Name: "default", Primary: true}}
	}

	for i := 0; i < numWallpapers; i++ {
		var wallpaper models.Wallpaper
		if setFromFavorites {
			favorites, err := loadFavorites()
			if err != nil {
				return fmt.Errorf("error loading favorites: %w", err)
			}
			if len(favorites) == 0 {
				return fmt.Errorf("no favorites found")
			}
			fav := favorites[rand.Intn(len(favorites))] //nolint:gosec // G404: Cryptographically secure random is not required for wallpaper selection.
			wallpaper = fav.Wallpaper
		} else {
			var err error
			wallpaper, err = controller.GetRandomFromFeed(targetTheme)
			if err != nil {
				return fmt.Errorf("error getting random wallpaper %d: %w", i+1, err)
			}
		}
		wallpapers = append(wallpapers, wallpaper)
	}

	return applyWallpapers(cmd, controller, wallpapers, monitors, cfg, targetMonitorIndex)
}

func runSetUndo(cmd *cobra.Command, args []string) error {
	if err := ensureConfig(); err != nil {
		return err
	}

	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}
	controller := core.NewController(cfg)

	state, err := loadState()
	if err != nil {
		return fmt.Errorf("could not load state: %w", err)
	}

	var wallpapers []models.Wallpaper

	// Try to use PreviousWallpapers if available (multi-monitor support)
	if len(state.PreviousWallpapers) > 0 {
		for _, id := range state.PreviousWallpapers {
			wp, err := controller.GetWallpaper(id)
			if err != nil {
				cmd.Printf("Warning: could not retrieve previous wallpaper '%s': %v\n", id, err)
				continue
			}
			wallpapers = append(wallpapers, *wp)
		}
	} else if state.PreviousWallpaperID != "" {
		wp, err := controller.GetWallpaper(state.PreviousWallpaperID)
		if err != nil {
			return fmt.Errorf("could not get previous wallpaper '%s': %w", state.PreviousWallpaperID, err)
		}
		wallpapers = append(wallpapers, *wp)
	}

	if len(wallpapers) == 0 {
		return fmt.Errorf("no previous wallpaper found in state")
	}

	// When we undo, we don't want to update the state again in the same way,
	// so we call a slightly different application function or pass a flag.
	// For simplicity, we'll just apply it without a state change.
	return applyWallpapers(cmd, controller, wallpapers, []core.Monitor{}, cfg, -1)
}

func applyWallpapers(cmd *cobra.Command, controller *core.Controller, wallpapers []models.Wallpaper, monitors []core.Monitor, cfg *models.Config, targetIndex int) error {
	if len(wallpapers) == 0 {
		return fmt.Errorf("no wallpapers provided to apply")
	}

	if config.DryRun {
		for _, wp := range wallpapers {
			if !config.Quiet {
				cmd.Printf("[DRY-RUN] Se aplicaría %s (Source: %s, URL: %s)\n", wp.ID, wp.Source, wp.URL)
			}
		}
		return nil
	}

	localPaths := make([]string, len(wallpapers))
	for i, wp := range wallpapers {
		if !config.Quiet {
			cmd.Printf("Preparing wallpaper: %s (Source: %s)\n", wp.ID, wp.Source)
		}
		if config.Debug {
			lum := controller.ColorManager.GetLuminance(wp.Color)
			cmd.Printf("   [DEBUG] Color: %s | Luminance: %.2f | Dark: %v\n", wp.Color, lum, lum < 100)
		}
		var err error
		localPaths[i], err = controller.DownloadWallpaper(wp)
		if err != nil {
			return fmt.Errorf("error downloading wallpaper %s: %w", wp.ID, err)
		}

		// Update path in feed and favorites
		_ = controller.UpdateWallpaperPath(wp.ID, localPaths[i])
	}

	// Determine command to run, prioritizing the flag, then auto-detection.
	customCmdTpl := setCommand

	if customCmdTpl != "" {
		if !config.Quiet {
			cmd.Printf("Running custom command for %d wallpaper(s)\n", len(localPaths))
		}
		for _, localPath := range localPaths {
			finalCmd := strings.ReplaceAll(customCmdTpl, "%s", shellQuote(localPath))
			if !config.Quiet {
				cmd.Printf("Running custom command: %s\n", finalCmd)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			err := exec.CommandContext(ctx, "sh", "-c", finalCmd).Run() //nolint:gosec // User provides the command template.
			cancel()
			if err != nil {
				return fmt.Errorf("error running custom wallpaper command: %w", err)
			}
		}
	} else {
		// Fallback to existing auto-detection logic
		respectDark := true
		if cfg != nil {
			respectDark = cfg.Behavior.RespectDarkMode
		}
		changer := core.NewWallpaperChanger("", respectDark)
		if config.Debug {
			cmd.Printf("   [DEBUG] Desktop Environment detected: %s\n", changer.Env)
		}
		changer.Transition = resolveTransition(cfg, cmd)
		if transitionExplicit(cmd, changer.Transition) && !changer.SupportsTransitions() {
			if !config.Quiet {
				cmd.Printf("Warning: transition effects are only supported by awww/swww; ignoring them on '%s'.\n", changer.Env)
			}
		}

		mmMode := setMultiMonitor
		if mmMode == "" && cfg != nil {
			mmMode = cfg.Behavior.MultiMonitor
		}

		// Call the new SetWallpapers (plural) function
		if err := changer.SetWallpapers(localPaths, monitors, mmMode); err != nil {
			return fmt.Errorf("error setting wallpaper(s): %w", err)
		}
	}

	// Update state - for multi-monitor, we'll just store the ID of the first wallpaper for simplicity
	// or the last one if we want to track the "main" one. Let's store the first one.
	state, err := loadState()
	if err != nil {
		cmd.Printf("Warning: could not load state to update it: %v\n", err)
	} else {
		var ids []string
		for _, wp := range wallpapers {
			ids = append(ids, wp.ID)
		}

		var newCurrentWallpapers []string

		if targetIndex >= 0 {
			// Partial update: preserve existing state and update only the target index
			newCurrentWallpapers = make([]string, len(state.CurrentWallpapers))
			copy(newCurrentWallpapers, state.CurrentWallpapers)

			// Ensure capacity if targetIndex is beyond current length
			if targetIndex >= len(newCurrentWallpapers) {
				grown := make([]string, targetIndex+1)
				copy(grown, newCurrentWallpapers)
				newCurrentWallpapers = grown
			}

			if len(ids) > 0 {
				newCurrentWallpapers[targetIndex] = ids[0]
			}
		} else {
			// Full update (clone or distinct set)
			newCurrentWallpapers = ids
		}

		changed := false
		if len(state.CurrentWallpapers) != len(newCurrentWallpapers) {
			changed = true
		} else {
			for i, id := range newCurrentWallpapers {
				if state.CurrentWallpapers[i] != id {
					changed = true
					break
				}
			}
		}

		if changed {
			state.PreviousWallpaperID = state.CurrentWallpaperID
			state.PreviousWallpapers = state.CurrentWallpapers

			// Update singular ID only if we are updating the first monitor or doing a full update
			if len(newCurrentWallpapers) > 0 {
				if targetIndex == -1 || targetIndex == 0 {
					state.CurrentWallpaperID = newCurrentWallpapers[0]
				}
			}
			state.CurrentWallpapers = newCurrentWallpapers
		}
		if err := saveState(state); err != nil {
			cmd.Printf("Warning: could not save state: %v\n", err)
		}
	}

	if !config.Quiet {
		cmd.Println("Wallpaper(s) set successfully.")
	}
	return nil
}

// shellQuote escapa un argumento para pasarlo a sh -c sin permitir inyección.
func shellQuote(s string) string {
	if s != "" && !strings.ContainsAny(s, " \t\n\"'\\$`;&|<>()*?!") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// resolveTransition aplica la precedencia flag > config.
func resolveTransition(cfg *models.Config, cmd *cobra.Command) models.TransitionConfig {
	t := cfg.Behavior.Transition
	if cmd == nil {
		return t
	}
	if f := cmd.Flags().Lookup("transition-type"); f != nil && f.Changed {
		t.Type = setTransitionType
	}
	if f := cmd.Flags().Lookup("transition-duration"); f != nil && f.Changed {
		t.Duration = setTransitionDuration
	}
	if f := cmd.Flags().Lookup("transition-fps"); f != nil && f.Changed {
		t.FPS = setTransitionFPS
	}
	if f := cmd.Flags().Lookup("transition-step"); f != nil && f.Changed {
		t.Step = setTransitionStep
	}
	if f := cmd.Flags().Lookup("transition-angle"); f != nil && f.Changed {
		t.Angle = setTransitionAngle
	}
	if f := cmd.Flags().Lookup("transition-pos"); f != nil && f.Changed {
		t.Pos = setTransitionPos
	}
	if f := cmd.Flags().Lookup("transition-bezier"); f != nil && f.Changed {
		t.Bezier = setTransitionBezier
	}
	if f := cmd.Flags().Lookup("transition-wave"); f != nil && f.Changed {
		t.Wave = setTransitionWave
	}
	if f := cmd.Flags().Lookup("transition-engine"); f != nil && f.Changed {
		t.Engine = setTransitionEngine
	}
	if f := cmd.Flags().Lookup("transition-linger"); f != nil && f.Changed {
		t.Linger = setTransitionLinger
	}
	if f := cmd.Flags().Lookup("transition-mask"); f != nil && f.Changed {
		t.Mask = setTransitionMask
	}
	if f := cmd.Flags().Lookup("transition-feather"); f != nil && f.Changed {
		t.Feather = setTransitionFeather
	}
	if f := cmd.Flags().Lookup("transition-invert"); f != nil && f.Changed {
		t.Invert = setTransitionInvert
	}
	if f := cmd.Flags().Lookup("transition-layer"); f != nil && f.Changed {
		t.Layer = setTransitionLayer
	}
	return t
}

// transitionExplicit indica si el usuario pidió una transición distinta de la
// implícita por defecto. Evita avisar en cada `set` sobre escritorios sin
// soporte cuando no se ha configurado nada especial.
func transitionExplicit(cmd *cobra.Command, t models.TransitionConfig) bool {
	if cmd != nil {
		for _, name := range []string{"transition-type", "transition-duration", "transition-fps", "transition-step", "transition-angle", "transition-pos", "transition-bezier", "transition-wave", "transition-engine", "transition-linger", "transition-mask", "transition-feather", "transition-invert", "transition-layer"} {
			if f := cmd.Flags().Lookup(name); f != nil && f.Changed {
				return true
			}
		}
	}
	return t.Type != "" && t.Type != "simple" && t.Type != "none"
}
