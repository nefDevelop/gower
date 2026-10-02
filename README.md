# Gower - Wallpaper Manager CLI

[![Go Report Card](https://goreportcard.com/badge/github.com/nefDevelop/gower)](https://goreportcard.com/report/github.com/nefDevelop/gower)
[![CI](https://github.com/nefDevelop/gower/actions/workflows/ci.yml/badge.svg)](https://github.com/nefDevelop/gower/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/nefDevelop/gower.svg)](https://github.com/nefDevelop/gower/releases/latest)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**Gower** is a powerful CLI tool to discover, manage, and change desktop wallpapers from online and local sources.
**Gower** es una potente herramienta de línea de comandos para descubrir, gestionar y cambiar los fondos de pantalla de tu escritorio desde diversas fuentes.

> [!WARNING]
> **In development / En desarrollo**: Gower is in active development. Bugs, breaking changes, and instability may occur. Use at your own risk.

## ✨ Features / Características

- **Multi-Provider**: Download from Wallhaven, Reddit, NASA APOD, Bing, Unsplash, and custom JSON APIs
  **Múltiples Proveedores**: Descarga desde Wallhaven, Reddit, NASA, Bing, Unsplash y APIs configurables
- **Feed History**: Track wallpapers you've seen with search, filter, and purge
- **Favorites & Blacklist**: Save preferred wallpapers and avoid unwanted ones
- **Daemon Mode**: Auto-change wallpapers on a schedule, optionally as a systemd user service
- **Smart Dark Mode**: Detects system theme (dark/light) and selects matching wallpapers
- **Rate Limiting**: Respectful API usage with configurable limits
- **Multi-Monitor**: Supports clone and distinct wallpaper modes per monitor
- **Local Indexing**: Optionally index a local wallpaper folder into the feed
- **Flexible Output**: Table or JSON output for scripting

## 📦 Installation / Instalación

### Prebuilt binaries / Binarios precompilados

Download from [releases/latest](https://github.com/nefDevelop/gower/releases/latest):

| Platform | Asset |
|----------|-------|
| Linux amd64 | `gower` |
| Windows amd64 | `gower.exe` |

```bash
# Linux
chmod +x gower && sudo mv gower /usr/local/bin/
```

### From source / Desde el código fuente

```bash
# Prerequisites / Requisitos: Go 1.25+
git clone https://github.com/nefDevelop/gower
cd gower
make build
```

## 🖥️ Supported desktops / Escritorios soportados

Gower detects your session automatically and picks the right mechanism. You can
check what it found with `gower status`.

| Desktop | Mechanism | Requires |
|---------|-----------|----------|
| Niri | `awww`, `swww` or `swaybg` | one of the three |
| Sway | `swaymsg` (per output) or `swaybg` | — |
| GNOME | `gsettings` | — |
| KDE / Plasma | `dbus-send` talking to PlasmaShell | — |
| Hyprland | `dms` or `quickshell` | one of the two |
| Any X11 | `feh` or `nitrogen` | one of the two |
| Windows | built-in (no external tool) | — |

On Wayland compositors, running more than one wallpaper daemon at once causes
flickering. Gower prefers whichever daemon is already running; see
[Troubleshooting](#-solución-de-problemas).

## 🚀 Quick Start / Inicio rápido

```bash
# Init config / Inicializar configuración
gower config init

# Explore wallpapers / Explorar fondos de pantalla
gower explore

# Sync feed / Sincronizar historial
gower feed update

# Set a random wallpaper / Establecer fondo aleatorio
gower set random

# Watch a directory for new wallpapers / Vigilar directorio
gower feed watch

# Show feed statistics / Estadísticas del feed
gower stats

# Start daemon / Iniciar demonio
gower daemon start
```

## 📖 Commands / Comandos

| Command | Description | Descripción |
|---------|-------------|-------------|
| `explore` | Search across providers | Buscar en proveedores |
| `download` | Download a wallpaper by ID or URL | Descargar un fondo por ID o URL |
| `wallpaper` | Show details, or remove from feed/disk | Ver detalles o quitar del feed/disco |
| `set` | Set the wallpaper | Establecer el fondo |
| `set random` | Set a random wallpaper | Establecer un fondo aleatorio |
| `set undo` | Revert to the previous wallpaper | Volver al fondo anterior |
| `feed show` | Show feed history | Listar el historial |
| `feed update` | Sync feed from provider caches | Sincronizar desde las cachés |
| `feed watch` | Watch a directory for new wallpapers | Vigilar un directorio |
| `feed analyze` | Extract metadata and colors | Extraer metadatos y colores |
| `feed random` | Random wallpaper from feed or favorites | Fondo aleatorio del feed o favoritos |
| `feed purge` | Purge feed history | Purgar el historial |
| `feed stats` | Feed statistics | Estadísticas del feed |
| `feed get colors` | Feed color palette | Paleta de colores del feed |
| `favorites list` | List favorites | Listar favoritos |
| `favorites add` | Add a favorite | Añadir favorito |
| `favorites remove` | Remove a favorite | Quitar favorito |
| `favorites analyze` | Analyze favorites | Analizar favoritos |
| `favorites get colors` | Favorites color palette | Paleta de colores de favoritos |
| `blacklist add` | Blacklist a wallpaper | Añadir a la lista negra |
| `blacklist remove` | Un-blacklist a wallpaper | Quitar de la lista negra |
| `blacklist list` | List blacklisted wallpapers | Listar la lista negra |
| `config init` | Init configuration | Inicializar configuración |
| `config show` | Show current config | Mostrar configuración |
| `config get` | Read a config value | Leer un valor |
| `config set` | Write a config value | Cambiar un valor |
| `config reset` | Reset to defaults | Restablecer valores |
| `config update` | Migrate config to the current structure | Migrar la configuración |
| `config provider` | Manage generic providers | Gestionar proveedores genéricos |
| `export` | Export config, feed, favorites or all | Exportar datos |
| `import` | Import config or favorites | Importar datos |
| `daemon start` | Start the daemon | Iniciar el demonio |
| `daemon stop` / `pause` / `resume` / `status` | Control the daemon | Controlar el demonio |
| `daemon install` / `uninstall` | systemd user service | Servicio systemd de usuario |
| `system cache` | Cache size, clean, prune | Tamaño y limpieza de caché |
| `system storage` | Verify and repair data files | Verificar y reparar datos |
| `status` | Show system status | Mostrar estado del sistema |
| `stats` | Detailed feed statistics | Estadísticas detalladas |
| `version` | Show version | Mostrar versión |

Run `gower <command> --help` for flags. / Usa `gower <comando> --help` para ver las opciones.

## 🛠️ Configuration / Configuración

Resolved in this order, first match wins:

1. `--config <path>`
2. `$XDG_CONFIG_HOME/gower/config.json`
3. `$HOME/.config/gower/config.json` (the usual case on Linux)
4. `%AppData%\gower\config.json` (Windows)
5. `~/.gower/config.json` (legacy, only if it already exists)

Key settings / Ajustes principales:

| Key | Default | Description |
|-----|---------|-------------|
| `providers.wallhaven.enabled` | `true` | Enable Wallhaven provider |
| `providers.reddit.subreddit` | `wallpapers` | Reddit subreddits (comma-separated) |
| `providers.unsplash.enabled` | `false` | Unsplash provider |
| `providers.unsplash.api_key` | `""` | Unsplash API key (required) |
| `search.min_width` | `1920` | Minimum wallpaper width |
| `search.min_height` | `1080` | Minimum wallpaper height |
| `search.aspect_ratio` | `16:9` | Preferred aspect ratio |
| `behavior.theme` | `""` | Force a theme instead of auto-detecting |
| `behavior.change_interval` | `30` | Daemon change interval (minutes) |
| `behavior.multi_monitor` | `clone` | Multi-monitor mode (clone/distinct) |
| `behavior.respect_dark_mode` | `true` | Pick wallpapers matching the system theme |
| `paths.index_wallpapers` | `false` | Index a local folder into the feed |
| `paths.wallpapers` | `""` | Folder to index |
| `ui.items_per_page` | `10` | Default page size for `list` commands |
| `ui.show_colors` | `true` | Include the color in `favorites list` |
| `limits.feed_soft_limit` | `400` | Feed size before cleanup kicks in |
| `limits.feed_hard_limit` | `2000` | Hard feed cap |
| `limits.analysis_workers` | `5` | Concurrent analysis workers |

`gower config get <key>` and `gower config set <key> <value>` work on any of
them. / Funcionan con cualquiera de ellas.

## 🏗️ Architecture / Arquitectura

```
gower/
├── cmd/              # CLI commands (Cobra), one file per command group
├── internal/
│   ├── core/         # Controller facade, services, wallpaper changer, colors
│   ├── providers/    # Wallhaven, Reddit, NASA, Bing, Unsplash, Generic
│   ├── utils/        # Logger, HTTP clients, rate limiter, file security
│   └── genconfig/    # AST generator for config_paths_gen.go
├── pkg/models/       # Shared types (Config, Wallpaper) + generated config paths
└── main.go           # Entry point
```

## 🧪 Development / Desarrollo

```bash
make test             # Unit tests
make lint             # go vet
golangci-lint run     # Full lint (needs golangci-lint v2.5.0, the version CI pins)
make generate         # Regenerate config code after changing models
make build-all        # Cross-compile for Linux and Windows
make release RELEASE_VERSION=vX.Y.Z   # Tag and publish, which triggers the release workflow
```

CI runs the suite with `-shuffle=on` on purpose: the tests share package-level
state, and shuffling makes any order dependence fail the build instead of
lying in wait.

## 🔍 Solución de Problemas

### Parpadeo o el fondo no cambia (Wayland/Niri)
En entornos Wayland como Niri, tener múltiples demonios de fondo (ej. `swww` y `awww`) ejecutándose simultáneamente puede causar conflictos, haciendo que el fondo parpadee o vuelva al anterior inmediatamente.

Gower prioriza los gestores en este orden para Niri:
1. **Demonio ya activo**: si `awww-daemon` está corriendo, usa `awww`. Si `swww-daemon` está corriendo, usa `swww`.
2. **Herramientas instaladas**: si ninguno está activo, intenta iniciar `awww` (si está instalado) y luego `swww`.

**Solución**: asegúrate de que solo un demonio de fondo esté activo. Puedes verificarlo con `gower status`, que mostrará una advertencia si detecta múltiples demonios.

### `gower status` reports a desktop you don't recognize
Gower falls back to whichever tool it finds first when it can't identify the
session. Check that you are running inside the graphical session, not over a
plain SSH shell.

---

## 📄 License / Licencia

[MIT](LICENSE)
