#!/bin/sh
# Uso: ./listar.sh /ruta/a/tus/wallpapers [light|dark]

dir="$1"
mode="${2:-dark}" # Si no se especifica modo, usa "dark" por defecto

if [ -z "$dir" ]; then
    echo "Error: Faltan argumentos." >&2
    echo "Uso: $0 <ruta_wallpapers> [light|dark]" >&2
    exit 1
fi

if [ ! -d "$dir" ]; then
    echo "La carpeta $dir no existe." >&2
    exit 1
fi

# Definir patrón de búsqueda según el modo (escapando corchetes para find)
if [ "$mode" = "light" ]; then
    pattern='*\[l\]*'
else
    pattern='*\[d\]*'
fi

# Listar archivos que coincidan con la etiqueta en el nombre
find "$dir" -maxdepth 1 -type f -iname "$pattern"
