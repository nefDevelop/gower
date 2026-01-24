#!/bin/sh
# Uso: ./listar.sh /ruta/a/tus/wallpapers [light|dark]

dir="$1"
mode="${2:-dark}" # Si no se especifica modo, usa "dark" por defecto

if [ -z "$dir" ]; then
    echo "Error: Faltan argumentos."
    echo "Uso: $0 <ruta_wallpapers> [light|dark]"
    exit 1
fi

target_dir="$dir/$mode"

# Verificar si la carpeta del modo existe (si se ha indexado antes)
if [ ! -d "$target_dir" ]; then
    echo "La carpeta $target_dir no existe. ¿Has ejecutado el indexador?"
    exit 1
fi

# Listar todos los archivos o enlaces simbólicos en esa carpeta
find "$target_dir" -type l -o -type f
