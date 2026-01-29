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
