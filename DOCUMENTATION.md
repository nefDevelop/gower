# Documentación del Plugin: Wallpaper Widget

Este documento detalla la arquitectura, características y funcionamiento interno del plugin "Wallpaper Widget".

## 1. Descripción General

**Wallpaper Widget** es un complemento para el escritorio que permite a los usuarios descubrir, gestionar y cambiar sus fondos de pantalla de forma avanzada. Se integra con diversas fuentes en línea y ofrece un alto grado of personalización.

- **ID del Plugin**: `com.nef734.wallpaperwidget`
- **Componente Principal (UI)**: `ui/MainPanel.qml`

## 2. Arquitectura de Módulos

El plugin sigue una arquitectura modular y desacoplada, orquestada por un singleton central (`ui/Controller.qml`). Este controlador es responsable de inicializar y conectar todos los módulos del backend, actuando como la única fuente de verdad y punto de comunicación para la interfaz de usuario.

### Módulos Principales

- **`Controller` (Singleton)**: El cerebro del plugin. Instancia todos los demás módulos, inyecta las dependencias necesarias y expone la API pública que consume la UI.
- **`ProviderManager` (Singleton)**: Gestiona todas las fuentes de wallpapers en línea (Reddit, Wallhaven, NASA). Implementa lógicas de _rate limiting_ y _circuit breaker_ para un consumo de API robusto y respetuoso.
- **`StorageManager` (Singleton)**: Abstrae toda la lógica de persistencia de datos. Gestiona archivos JSON para el historial, favoritos, lista negra y configuración. Incluye mecanismos de seguridad como backups (`.bak`) y recuperación de datos corruptos.
- **`WallpaperChanger` (Singleton)**: Se encarga de la lógica de cambiar el fondo de pantalla. Es agnóstico a la plataforma, detectando el entorno de escritorio (KDE, GNOME, feh, etc.) para usar el comando adecuado. También gestiona el cambio automático temporizado y el soporte multimonitor.
- **`FileSystemManager`**: Gestiona las operaciones del sistema de archivos, como la descarga de wallpapers y thumbnails, y su almacenamiento en las rutas correctas.
- **`ColorEngine`**: Módulo para el análisis de imágenes. Detecta si `matugen` está disponible en el sistema para extraer paletas de colores de los wallpapers.
- **`Logger`**: Un sistema de logging interno para depuración.

## 3. Características Principales

### 3.1. Interfaz de Usuario (`ui/`)

La interfaz está construida con QML y se divide en cuatro pestañas principales:

1.  **Feed**: Muestra el historial de wallpapers descubiertos. Permite paginación, actualización y la selección aleatoria de un wallpaper del historial.
2.  **Explorar**: Contiene una barra de búsqueda para encontrar nuevos wallpapers a través de los proveedores configurados. Los resultados se muestran en una parrilla paginada.
3.  **Favoritos**: Muestra una colección de los wallpapers que el usuario ha marcado como favoritos.
4.  **Configuración**: Interfaz para ajustar todos los parámetros del widget, delegando en `SettingsView.qml`.

### 3.2. Proveedores de Wallpapers (`backend/ProviderManager.qml`)

El widget puede obtener imágenes de múltiples fuentes:

- **Wallhaven**: Búsqueda por palabra clave, con soporte para API Key, categorías, pureza y filtros de resolución.
- **Reddit**: Búsqueda en subreddits específicos.
- **NASA**: Obtiene imágenes del "Astronomy Picture of the Day" (APOD).

El sistema está diseñado para ser extensible a nuevos proveedores.

**Mecanismos de Resiliencia**:

- **Rate Limiting**: Monitoriza las cabeceras de respuesta de las APIs para evitar exceder los límites de peticiones, pausando futuras llamadas si es necesario.
- **Circuit Breaker**: Si una API falla repetidamente, el `ProviderManager` "abrirá el circuito" y dejará de hacer peticiones a esa API durante un tiempo para permitir su recuperación.

### 3.3. Gestión de Datos (`backend/StorageManager.qml`)

Todos los datos se almacenan localmente en la carpeta de configuración del plugin, dentro de `data_storage/`.

- **`feed.json`**: Almacena el historial de todos los wallpapers encontrados. Cada objeto contiene metadatos como el ID, la URL, la resolución, la paleta de colores (`pal`), la luminosidad (`lum`) y la fecha en que se añadió.
- **`favorites.json`**: Una lista simple con los IDs de los wallpapers marcados como favoritos.
- **`blacklist.json`**: Una lista de IDs de wallpapers que el usuario ha bloqueado.
- **`settings.json`**: Un objeto JSON que contiene toda la configuración del usuario. Ver sección "Configuración" para más detalles.

### 3.4. Cambio de Fondos de Pantalla (`backend/WallpaperChanger.qml`)

- **Detección de Entorno**: Intenta detectar automáticamente el gestor de ventanas o entorno de escritorio para usar el comando correcto. Soporta:
  - `feh`
  - `nitrogen`
  - `gsettings` (GNOME)
  - `dbus` (KDE Plasma)
  - `swaymsg` (Sway)
  - `hyprctl` (Hyprland)
- **Cambio Automático**: Un temporizador interno permite rotar el fondo de pantalla a intervalos definidos por el usuario (ej. cada 30 minutos).
- **Soporte Multimonitor**: Detecta múltiples pantallas a través de `xrandr`. Puede configurarse para:
  - `all_same`: Usar el mismo wallpaper en todas las pantallas.
  - `distinct`: Usar un wallpaper diferente para cada pantalla.
- **Selección por Tema**: Si los archivos locales han sido indexados con `wallpaperIndex.sh`, puede seleccionar wallpapers oscuros (`[d]`) o claros (`[l]`) para que coincidan con el tema del sistema.

## 4. Scripts Auxiliares

El proyecto incluye scripts de shell para facilitar el desarrollo y el uso:

- **`historyMaker.sh`**: Un script **para desarrolladores**. Genera un archivo `feed.json` con 3000 entradas falsas para probar la UI sin necesidad de consumir APIs. **No es para uso del usuario final**.
- **`wallpaperIndex.sh`**: Una utilidad **para el usuario**. Este script analiza una carpeta de imágenes locales, determina si son predominantemente "claras" u "oscuras" usando ImageMagick, y les añade una etiqueta `[l]` o `[d]` al nombre del archivo. Esto permite al `WallpaperChanger` seleccionar wallpapers que se ajusten al tema del sistema.

## 5. Almacenamiento y Estructura de Datos

### `settings.json`

A continuación se muestra un resumen de las opciones configurables, inferidas de `StorageManager.qml`:

```json
{
  "providers": {
    "wallhaven": { "enabled": true, "apiKey": "", "categories": "100", "purity": "100" },
    "reddit": { "enabled": true, "subreddit": "wallpapers", "sort": "hot" },
    "nasa": { "enabled": false, "apiKey": "DEMO_KEY" }
  },
  "search": {
    "min_width": 1920,
    "min_height": 1080,
    "aspect_ratio": "16:9",
    "tolerance": 0.1
  },
  "behavior": {
    "theme": "dark", // "light" o "dark"
    "change_interval": 30, // en minutos
    "multi_monitor": "clone", // "clone" o "distinct"
    "wallpaper_command": "feh --bg-fill", // Comando personalizado
    "auto_download_thumbs": true
  },
  "paths": {
    "wallpapers": "/ruta/a/guardar/wallpapers",
    "thumbs": "thumbs/" // Relativo a la carpeta de datos
  }
}
```

### `feed.json` (Ejemplo de un item)

```json
{
  "id": "wh_123456",
  "src": 0, // 0: wallhaven, 1: reddit, 2: nasa
  "url": "https://...",
  "url_thumb": "https://...",
  "res": [3840, 2160],
  "pal": ["#1a1b1e", "#3a3b3e", "#5a5b5e"],
  "lum": "d", // "d" para dark, "l" para light
  "added": 1672531200, // Timestamp
  "viewed": false,
  "query": "landscape",
  "provider": "wallhaven",
  "title": "A beautiful landscape"
}
```

## 6. Límites y Estilo de Código

- **Estilo**: El código es asíncrono y se basa en `Promise`. La comunicación entre módulos se realiza mediante señales y funciones de callback.
- **Dependencias Externas**:
  - `matugen` (opcional, para análisis de color).
  - `ImageMagick` (opcional, para el script `wallpaperIndex.sh`).
  - Un comando para cambiar wallpapers (`feh`, `nitrogen`, etc.).
- **Limitaciones**:
  - La purga del historial tiene un límite "suave" (por defecto 400) y un límite "duro" (por defecto 2000), para evitar que el archivo `feed.json` crezca indefinidamente.
  - La detección de entorno de escritorio depende de comandos estándar que deben estar en el `PATH` del sistema.
