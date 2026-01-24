#!/bin/sh
# Uso: ./indexar.sh /ruta/a/tus/wallpapers

dir="$1"

# Verificar que se pasó un directorio
if [ -z "$dir" ]; then
    echo "Error: Debes especificar la carpeta de wallpapers."
    echo "Uso: $0 <ruta>"
    exit 1
fi

# Verificar dependencias (ImageMagick)
if command -v magick >/dev/null 2>&1; then
    IM_CMD="magick"
elif command -v convert >/dev/null 2>&1; then
    IM_CMD="convert"
else
    echo "Error: Se requiere ImageMagick (comando 'magick' o 'convert') para analizar las imágenes."
    exit 1
fi

# 1. Crear directorios de clasificación
mkdir -p "$dir/light" "$dir/dark"

# 2. Limpiar clasificaciones anteriores
rm -f "$dir/light/"* "$dir/dark/"*

# 3. Buscar imágenes (jpg, png, webp) ignorando las carpetas light/dark para evitar bucles
find "$dir" -path "$dir/light" -prune -o -path "$dir/dark" -prune -o -type f \( -iname '*.jpg' -o -iname '*.jpeg' -o -iname '*.png' -o -iname '*.webp' \) -print | while read -r img; do
    
    # 4. Analizar brillo con ImageMagick (0.0 = negro, 1.0 = blanco)
    # Redimensionamos a 1x1 pixel y obtenemos la media de gris
    brightness=$($IM_CMD "$img" -colorspace gray -resize 1x1 -format "%[fx:mean]" info:)
    
    # Usar awk para comparar decimales (si brillo > 0.5 es light)
    type=$(echo "$brightness" | awk '{if ($1 > 0.5) print "light"; else print "dark"}')
    
    name=$(basename "$img")
    
    # 5. Crear enlace simbólico según el tipo detectado
    if [ "$type" = "light" ]; then
        ln -sf "$img" "$dir/light/$name"
        echo "-> Light: $name"
    elif [ "$type" = "dark" ]; then
        ln -sf "$img" "$dir/dark/$name"
        echo "-> Dark:  $name"
    fi
done
