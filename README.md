# Gower - Wallpaper Manager CLI

[![Go Report Card](https://goreportcard.com/badge/github.com/user/gower)](https://goreportcard.com/report/github.com/user/gower)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**Gower** is a powerful CLI tool to discover, manage, and change desktop wallpapers from online and local sources.
**Gower** es una potente herramienta de línea de comandos para descubrir, gestionar y cambiar los fondos de pantalla de tu escritorio desde diversas fuentes.

> [!WARNING]
> **In development / En desarrollo**: Gower is in active development. Bugs, breaking changes, and instability may occur. Use at your own risk.

## ✨ Features / Características

- **Multi-Provider**: Download from Wallhaven, Reddit, NASA APOD, Bing, Unsplash, and custom JSON APIs
  **Múltiples Proveedores**: Descarga desde Wallhaven, Reddit, NASA, Bing, Unsplash y APIs configurables
- **Feed History**: Track wallpapers you've seen with search, filter, and purge
- **Favorites & Blacklist**: Save preferred wallpapers and avoid unwanted ones
- **Daemon Mode**: Auto-change wallpapers on a schedule in the background
- **Smart Dark Mode**: Detects system theme (dark/light) and selects matching wallpapers
- **Rate Limiting**: Respectful API usage with configurable limits
- **Multi-Monitor**: Supports clone and distinct wallpaper modes per monitor
- **Flexible Output**: Table or JSON output for scripting

## 📦 Installation / Instalación

### From source / Desde el código fuente

```bash
# Prerequisites / Requisitos: Go 1.25+
git clone https://github.com/user/gower
cd gower
make build
```

### Binary / Binario

```bash
make build-all
# Binaries in ./dist/
```

## 🚀 Quick Start / Inicio rápido

```bash
# Init config / Inicializar configuración
gower config init

# Explore wallpapers / Explorar fondos de pantalla
gower explore

# Sync feed / Sincronizar historial
gower feed sync

# Set a random wallpaper / Establecer fondo aleatorio
gower set random

# Start daemon / Iniciar demonio
gower daemon start

# Watch directory for new wallpapers / Vigilar directorio
gower feed watch

# Show feed statistics / Estadísticas del feed
gower stats
```

## 📖 Commands / Comandos

| Command | Description | Descripción |
|---------|-------------|-------------|
| `config init` | Init configuration | Inicializar configuración |
| `config show` | Show current config | Mostrar configuración |
| `config set` | Set a config value | Cambiar un valor |
| `explore` | Search providers | Buscar en proveedores |
| `feed sync` | Sync feed from providers | Sincronizar historial |
| `feed list` | List feed items | Listar historial |
| `set` | Set wallpaper | Establecer fondo |
| `favorites` | Manage favorites | Gestionar favoritos |
| `blacklist` | Manage blacklist | Gestionar lista negra |
| `daemon` | Background service | Servicio en segundo plano |
| `daemon install` | Install systemd user service | Instalar servicio systemd |
| `daemon uninstall` | Remove systemd user service | Eliminar servicio systemd |
| `status` | Show system status | Mostrar estado del sistema |
| `export/import` | Data portability | Portabilidad de datos |
| `feed watch` | Watch directory for new wallpapers | Vigilar directorio por cambios |
| `stats` | Show detailed feed statistics | Estadísticas detalladas del feed |

## 🛠️ Configuration / Configuración

Configuration is stored in `$XDG_CONFIG_HOME/gower/config.json` (Linux) or `~/.gower/config.json` (legacy).

Key settings / Ajustes principales:

| Key | Default | Description |
|-----|---------|-------------|
| `providers.wallhaven.enabled` | `true` | Enable Wallhaven provider |
| `providers.reddit.subreddit` | `wallpapers` | Reddit subreddits (comma-separated) |
| `search.min_width` | `1920` | Minimum wallpaper width |
| `search.min_height` | `1080` | Minimum wallpaper height |
| `behavior.change_interval` | `30` | Daemon change interval (minutes) |
| `behavior.multi_monitor` | `clone` | Multi-monitor mode (clone/distinct) |
| `limits.analysis_workers` | `5` | Concurrent analysis workers |
| `providers.unsplash.enabled` | `false` | Enable Unsplash provider |
| `providers.unsplash.api_key` | `""` | Unsplash API key (required) |

## 🏗️ Architecture / Arquitectura

```
gower/
├── cmd/              # CLI commands (Cobra)
├── internal/
│   ├── core/         # Core logic (Controller, Services, Color, Wallpaper)
│   ├── providers/    # Wallhaven, Reddit, NASA, Bing, Unsplash, Generic
│   └── utils/        # Logger, HTTP client, Rate limiter, File security
├── pkg/models/       # Shared types (Config, Wallpaper, FeedStats)
└── main.go           # Entry point
```

## 🧪 Development / Desarrollo

```bash
make test           # Run tests
make lint           # Go vet
golangci-lint run   # Full lint (requires golangci-lint v2)
go generate ./...   # Regenerate config code (after model changes)
```

## 📄 License / Licencia

MIT
