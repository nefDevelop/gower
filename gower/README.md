# Gower - Wallpaper Manager CLI

[![Go Report Card](https://goreportcard.com/badge/github.com/user/gower)](https://goreportcard.com/report/github.com/user/gower)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**Gower** es una potente herramienta de línea de comandos para descubrir, gestionar y cambiar los fondos de pantalla de tu escritorio desde diversas fuentes en línea y locales.

![Gower Demo](https://user-images.githubusercontent.com/12345/67890.gif) <!-- Placeholder for a demo GIF -->

## ✨ Características

- **Múltiples Proveedores**: Descarga fondos de pantalla de varias fuentes:
  - Wallhaven
  - Reddit
  - NASA Picture of the Day
  - Bing Wallpaper of the Day
  - Proveedores genéricos configurables (JSON API)
- **Gestión de Historial (Feed)**: Lleva un registro de los fondos de pantalla que has visto, con opciones para buscar, filtrar y purgar.
- **Favoritos y Lista Negra**: Guarda tus fondos de pantalla preferidos y evita los que no te gustan.
- **Modo Demonio**: Ejecuta `gower` en segundo plano para cambiar tu fondo de pantalla automáticamente a intervalos definidos.
- **Configuración Avanzada**: Personaliza todo, desde las dimensiones de la imagen y los proveedores hasta los comandos para establecer el fondo de pantalla.
- **Eficiente y Respetuoso**: Límites de frecuencia para las APIs, gestión de energía para portátiles y almacenamiento en caché local.
- **Salida Flexible**: Muestra la información en formato de tabla o JSON, ideal para scripting.

## 📦 Instalación

### Desde el código fuente

Asegúrate de tener Go instalado (versión 1.18 o superior).

```bash
git clone https://github.com/user/gower.git
cd gower
go install .
```

### Binarios (Próximamente)

Se proporcionarán binarios precompilados para Linux, macOS y Windows en la sección de [Releases](https://github.com/user/gower/releases).

## 🚀 Uso Básico

Gower funciona a través de subcomandos. Aquí están los más importantes:

### 1. Actualizar el historial local (Feed)

Este comando busca nuevos fondos de pantalla de los proveedores habilitados y los añade a tu historial local (feed).

```bash
gower feed update
```
> **Nota**: La primera vez, es una buena idea ejecutar `gower feed update --force` para ignorar los límites de frecuencia y poblar tu historial.

### 2. Establecer un fondo de pantalla

Puedes establecer un fondo de pantalla aleatorio de tu historial:

```bash
gower set random
```

O establecer un fondo de pantalla específico por su ID:

```bash
gower set <WALLPAPER_ID>
```

### 3. Ver el historial

Muestra los fondos de pantalla en tu historial local.

```bash
gower feed show
```

Puedes paginar y filtrar los resultados:

```bash
gower feed show --page 2 --limit 10 --theme dark
```

### 4. Modo Demonio

Para cambiar automáticamente tu fondo de pantalla cada 30 minutos:

```bash
gower daemon --interval 30m
```

## ⚙️ Configuración

Gower busca un archivo de configuración `config.json` en las siguientes ubicaciones:
- `$XDG_CONFIG_HOME/gower/config.json`
- `$HOME/.config/gower/config.json`
- `$HOME/.gower.json`

Puedes especificar una ruta de configuración diferente con el flag `--config`.

Un archivo de configuración de ejemplo podría ser:

```json
{
  "providers": {
    "wallhaven": {
      "enabled": true,
      "apiKey": "TU_API_KEY_DE_WALLHAVEN"
    },
    "reddit": {
      "enabled": true,
      "subreddit": "wallpapers+wallpaper",
      "sort": "hot",
      "limit": 100
    },
    "nasa": { "enabled": true },
    "bing": { "enabled": true, "market": "en-US" }
  },
  "search": {
    "min_width": 1920,
    "min_height": 1080,
    "aspect_ratio": "16:9"
  },
  "behavior": {
    "theme": "any",
    "change_interval": 30,
    "wallpaper_command": "feh --bg-fill %s"
  },
  "paths": {
    "wallpapers": "/home/user/Pictures/Wallpapers"
  },
  "limits": {
    "feed_soft_limit": 200,
    "feed_hard_limit": 1000
  }
}
```

### Comando para establecer el fondo de pantalla

La opción `behavior.wallpaper_command` es crucial. Debes ajustarla a tu entorno de escritorio o gestor de ventanas. `%s` será reemplazado por la ruta del archivo de imagen.

- **feh**: `feh --bg-fill %s`
- **GNOME**: `gsettings set org.gnome.desktop.background picture-uri file://%s`
- **Sway/Wayland**: `swaymsg output * bg %s fill`

## 📜 Lista de Comandos

A continuación se muestra una lista de los principales comandos disponibles. Usa `gower <comando> --help` para más detalles.

- `gower feed`: Gestiona el historial de fondos de pantalla.
  - `update`: Busca nuevos fondos de pantalla.
  - `show`: Muestra el historial.
  - `purge`: Limpia el historial.
  - `stats`: Muestra estadísticas.
- `gower set`: Establece un fondo de pantalla.
  - `random`: Establece uno aleatorio.
  - `<ID>`: Establece uno específico.
- `gower daemon`: Inicia el demonio para cambio automático.
- `gower status`: Muestra el estado actual (fondo de pantalla, etc.).
- `gower favorites`: Gestiona tus fondos de pantalla favoritos.
  - `add <ID>`
  - `remove <ID>`
  - `list`
- `gower blacklist`: Gestiona la lista negra.
  - `add <ID>`
  - `remove <ID>`
  - `list`
- `gower cache`: Gestiona la caché local.
- `gower config`: Gestiona la configuración.

## 🛠️ Construir desde el código fuente

```bash
# Clona el repositorio
git clone https://github.com/user/gower.git
cd gower

# Instala dependencias y verifica el código
go mod tidy
go vet ./...

# Ejecuta los tests
go test ./...

# Construye el binario
go build -o gower .
```

## 📄 Licencia

Este proyecto está bajo la Licencia MIT. Ver el archivo [LICENSE](LICENSE) para más detalles.
