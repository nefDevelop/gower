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

# Limpiar sistema antiguo de carpetas si existe
if [ -d "$dir/light" ]; then rm -rf "$dir/light"; fi
if [ -d "$dir/dark" ]; then rm -rf "$dir/dark"; fi

# Buscar imágenes en el directorio raíz
find "$dir" -maxdepth 1 -type f \( -iname '*.jpg' -o -iname '*.jpeg' -o -iname '*.png' -o -iname '*.webp' \) | while read -r img; do
    
    filename=$(basename "$img")
    
    # Analizar brillo
    brightness=$($IM_CMD "$img" -colorspace gray -resize 1x1 -format "%[fx:mean]" info:)
    
    # Determinar etiqueta
    is_light=$(echo "$brightness" | awk '{if ($1 > 0.5) print 1; else print 0}')
    
    if [ "$is_light" -eq 1 ]; then
        tag="[l]"
        wrong_tag="[d]"
    else
        tag="[d]"
        wrong_tag="[l]"
    fi
    
    # Verificar si necesita renombrado (falta etiqueta correcta o tiene la incorrecta)
    if ! echo "$filename" | grep -F -q "$tag" || echo "$filename" | grep -F -q "$wrong_tag"; then
        extension="${filename##*.}"
        filename_no_ext="${filename%.*}"
        
        # Limpiar nombre de etiquetas anteriores
        clean_name=$(echo "$filename_no_ext" | sed 's/ *\[[ld]\] *//g' | sed 's/^ *//;s/ *$//')
        
        if [ -z "$clean_name" ]; then clean_name="wallpaper"; fi
        
        new_name="${clean_name} ${tag}.${extension}"
        
        # Renombrar
        mv "$img" "$dir/$new_name"
        echo "Renombrado: $filename -> $new_name"
    fi
done
