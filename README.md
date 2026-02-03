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

A continuación se presenta una lista detallada de todos los comandos y opciones disponibles.

### Comando Raíz

`gower` es el comando principal para interactuar con la aplicación.

```
gower [subcomando] [flags]
```

#### Opciones Globales (Persistentes)

Estas opciones se pueden usar con cualquier subcomando.

- `--verbose`, `-v`: Habilita la salida detallada.
- `--debug`: Habilita la salida de depuración para diagnósticos.
- `--quiet`, `-q`: Suprime toda la salida excepto los errores.
- `--json`: Formatea la salida como JSON en lugar de texto o tablas.
- `--table`: Formatea la salida en una tabla (comportamiento por defecto).
- `--no-color`: Desactiva la salida con colores.
- `--config <ruta>`: Especifica la ruta al archivo de configuración.
- `--dry-run`: Simula la ejecución de un comando sin realizar cambios reales en el sistema.

---

### Subcomandos

#### `gower explore`

Busca fondos de pantalla en los proveedores configurados.

- **Uso**: `gower explore [término_de_búsqueda] [flags]`
- **Flags**:
  - `--provider <nombre>`: Usa un proveedor específico para la búsqueda.
  - `--all`: Busca en todos los proveedores habilitados simultáneamente.
  - `--min-width <píxeles>`: Filtra por ancho mínimo de imagen.
  - `--min-height <píxeles>`: Filtra por altura mínima de imagen.
  - `--aspect-ratio <ratio>`: Filtra por proporción de aspecto (ej. "16:9").
  - `--color <hex>`: Busca imágenes por un color dominante (código hexadecimal).
  - `--page, -p <número>`: Solicita una página específica de resultados.
  - `--force-update`: Fuerza una nueva búsqueda en el proveedor, ignorando la caché.
  - `--save`: Guarda los resultados de la búsqueda directamente en el `feed.json`.

#### `gower feed`

Gestiona el historial local de fondos de pantalla (feed).

- `gower feed show`: Muestra el historial.
  - `--page, -p <número>`: Número de página a mostrar.
  - `--limit, -l <número>`: Cantidad de ítems por página.
  - `--theme <dark|light>`: Filtra por tema.
  - `--color <hex>`: Filtra por color.
- `gower feed update`: Sincroniza el feed desde las cachés de los proveedores o realiza una nueva búsqueda si es necesario.
  - `--force`: Ignora los límites de frecuencia para forzar la actualización.
- `gower feed purge`: Elimina todo el historial del feed.
  - `--force`: Confirma la eliminación sin preguntar.
- `gower feed stats`: Muestra estadísticas sobre el feed.
- `gower feed analyze`: Analiza los ítems del feed para extraer metadatos como colores (requiere descarga).
  - `--all`: Analiza todos los ítems, no solo los nuevos.
  - `--force`: Fuerza la regeneración de datos (ej. miniaturas).
- `gower feed random`: Obtiene un fondo de pantalla aleatorio del feed o de favoritos.
  - `--theme <dark|light>`: Filtra por tema.
  - `--from-favorites`: Elige un fondo de pantalla aleatorio de la lista de favoritos en lugar del feed.

#### `gower set`

Establece un fondo de pantalla.

- **Uso**: `gower set [ID|URL|random] [flags]`
- `gower set random`: Establece un fondo de pantalla aleatorio (equivalente a `gower set --random`).
- `gower set undo`: Revierte al fondo de pantalla anterior.
- **Flags**:
  - `--id <ID>`: ID del fondo de pantalla a establecer.
  - `--url <URL>`: URL directa de una imagen para establecer como fondo de pantalla.
  - `--random`, `-r`: Activa el modo aleatorio.
  - `--theme <dark|light|auto>`: Filtra por tema al buscar uno aleatorio.
  - `--from-favorites`: Elige uno aleatorio solo de los favoritos.
  - `--multi-monitor <clone|distinct>`: Define el comportamiento para múltiples monitores.
  - `--command <comando>`: Usa un comando personalizado para establecer el fondo de pantalla (ej. `feh --bg-fill %s`).
  - `--no-download`: No descarga la imagen, útil si ya existe localmente.

#### `gower favorites`

Gestiona la lista de fondos de pantalla favoritos.

- `gower favorites list`: Muestra la lista de favoritos.
  - `--page <número>`: Página a mostrar.
  - `--limit <número>`: Ítems por página.
  - `--color <hex>`: Filtra por color.
- `gower favorites add <ID>`: Añade un fondo de pantalla a favoritos.
  - `--notes <texto>`: Añade notas personales al favorito.
- `gower favorites remove <ID>`: Elimina un fondo de pantalla de favoritos.
- `gower favorites export --file <ruta>`: Exporta la lista de favoritos a un archivo JSON.
- `gower favorites import --file <ruta>`: Importa favoritos desde un archivo JSON.

#### `gower blacklist`

Gestiona la lista negra para excluir fondos de pantalla.

- `gower blacklist add <ID>`: Añade un fondo de pantalla a la lista negra.
- `gower blacklist remove <ID>`: Elimina un fondo de pantalla de la lista negra.
- `gower blacklist list`: Muestra todos los fondos de pantalla en la lista negra.

#### `gower config`

Gestiona la configuración de la aplicación.

- `gower config init`: Crea la estructura de configuración y el archivo `config.json` inicial.
- `gower config show`: Muestra la configuración actual en formato JSON.
- `gower config get <clave>`: Obtiene el valor de una clave de configuración (ej. `providers.reddit.limit`).
- `gower config set <clave=valor>`: Establece el valor de una clave de configuración.
- `gower config reset`: Restablece la configuración a sus valores por defecto.
- `gower config export [archivo]`: Exporta la configuración a un archivo o a la salida estándar.
- `gower config import <archivo>`: Importa una configuración desde un archivo.

#### `gower daemon`

Controla el demonio que cambia el fondo de pantalla automáticamente.

- `gower daemon start`: Inicia el demonio en segundo plano.
  - `--interval <minutos>`: Intervalo en minutos para cambiar el fondo de pantalla.
  - `--from-favorites`: Usa solo favoritos para los cambios.
  - `--theme <dark|light>`: Filtra por tema.
- `gower daemon stop`: Detiene el demonio.
  - `--force`: Fuerza la detención.
- `gower daemon status`: Muestra el estado actual del demonio (corriendo o detenido).
- `gower daemon pause`: Pausa temporalmente los cambios de fondo de pantalla.
- `gower daemon resume`: Reanuda los cambios de fondo de pantalla.

#### `gower status`

Muestra un resumen del estado general de la aplicación.

- **Flags**:
  - `--providers`: Muestra solo el estado de los proveedores.
  - `--storage`: Muestra solo el uso de almacenamiento.
  - `--daemon`: Muestra solo el estado del demonio.
  - `--system`: Muestra solo información del sistema y dependencias.
  - `--json`: Muestra toda la información en formato JSON.

#### `gower cache`

Gestiona la caché de la aplicación.

- `gower cache clean`: Limpia el contenido del directorio de caché (imágenes, miniaturas).
- `gower cache size`: Muestra el tamaño actual ocupado por la caché.

#### `gower export`

Exporta datos de la aplicación.

- `gower export all`: Exporta toda la configuración, feed y favoritos.
  - `--file <ruta.zip>`: Exporta todo a un único archivo ZIP.
  - `--include-images`: Incluye las imágenes descargadas en el ZIP.
- `gower export config --file <ruta>`: Exporta solo la configuración.
- `gower export feed --file <ruta>`: Exporta solo el feed.


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
