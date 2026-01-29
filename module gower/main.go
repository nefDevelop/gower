// main.go - Estructura principal

import (
	"flag"
	"fmt"
	"os"
)

type Config struct {
	Verbose    bool
	Debug      bool
	Quiet      bool
	JSONOutput bool
	ConfigFile string
}

type Command interface {
	Execute() error
	Help() string
}

// --- Definiciones de Comandos (structs) ---

type FeedCommand struct{}
func (c *FeedCommand) Execute() error { fmt.Println("Ejecutando el comando 'feed'..."); return nil }
func (c *FeedCommand) Help() string   { return "Muestra el feed de wallpapers." }

type ExploreCommand struct{}
func (c *ExploreCommand) Execute() error { fmt.Println("Ejecutando el comando 'explore'..."); return nil }
func (c *ExploreCommand) Help() string   { return "Explora nuevos wallpapers." }

type FavoritesCommand struct{}
func (c *FavoritesCommand) Execute() error { fmt.Println("Ejecutando el comando 'favorites'..."); return nil }
func (c *FavoritesCommand) Help() string   { return "Muestra tus wallpapers favoritos." }

type SetCommand struct{}
func (c *SetCommand) Execute() error { fmt.Println("Ejecutando el comando 'set'..."); return nil }
func (c *SetCommand) Help() string   { return "Establece un nuevo wallpaper." }

type BlacklistCommand struct{}
func (c *BlacklistCommand) Execute() error { fmt.Println("Ejecutando el comando 'blacklist'..."); return nil }
func (c *BlacklistCommand) Help() string   { return "Añade un wallpaper a la lista negra." }

type ConfigCommand struct{}
func (c *ConfigCommand) Execute() error { fmt.Println("Ejecutando el comando 'config'..."); return nil }
func (c *ConfigCommand) Help() string   { return "Gestiona la configuración." }

type DaemonCommand struct{}
func (c *DaemonCommand) Execute() error { fmt.Println("Ejecutando el comando 'daemon'..."); return nil }
func (c *DaemonCommand) Help() string   { return "Inicia el demonio en segundo plano." }

type InteractiveCommand struct{}
func (c *InteractiveCommand) Execute() error { fmt.Println("Ejecutando el comando 'interactive'..."); return nil }
func (c *InteractiveCommand) Help() string   { return "Inicia el modo interactivo." }


// Comandos disponibles
var commands = map[string]Command{
	"feed":        &FeedCommand{},
	"explore":     &ExploreCommand{},
	"favorites":   &FavoritesCommand{},
	"set":         &SetCommand{},
	"blacklist":   &BlacklistCommand{},
	"config":      &ConfigCommand{},
	"daemon":      &DaemonCommand{},
	"interactive": &InteractiveCommand{},
}

func main() {
	// Definir flags globales
	var config Config
	flag.BoolVar(&config.Verbose, "v", false, "Habilita la salida detallada.")
	flag.BoolVar(&config.Debug, "debug", false, "Habilita la salida de depuración.")
	flag.BoolVar(&config.Quiet, "q", false, "Suprime toda la salida excepto los errores.")
	flag.BoolVar(&config.JSONOutput, "json", false, "Formatea la salida como JSON.")
	flag.StringVar(&config.ConfigFile, "config", "", "Ruta al archivo de configuración.")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Uso: %s [flags] <comando> [argumentos del comando]\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "\nComandos disponibles:")
		for name, cmd := range commands {
			fmt.Fprintf(os.Stderr, "  %s: %s\n", name, cmd.Help())
		}
		fmt.Fprintln(os.Stderr, "\nFlags globales:")
		flag.PrintDefaults()
	}

	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	cmdName := args[0]
	cmd, exists := commands[cmdName]

	if !exists {
		fmt.Fprintf(os.Stderr, "Error: Comando desconocido '%s'\n", cmdName)
		flag.Usage()
		os.Exit(1)
	}

	// Aquí podrías pasar la config y los argumentos restantes al comando
	// Por ejemplo: cmd.Init(config, args[1:])

	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error ejecutando el comando '%s': %v\n", cmdName, err)
		os.Exit(1)
	}
}

