import sys
import os
import subprocess
import json
import argparse
import random
import time

# --- Constantes y Rutas ---
SCRIPT_DIR = os.path.dirname(__file__)
CONFIG_FILE = os.path.join(SCRIPT_DIR, 'config.json')
STATUS_FILE = os.path.join(SCRIPT_DIR, 'status.json') # New status file
SUPPORTED_EXTENSIONS = ['.jpg', '.jpeg', '.png', '.webp']

# --- Lógica Principal del Gestor ---

def load_config():
    """Carga la configuración desde config.json."""
    if not os.path.exists(CONFIG_FILE):
        # Create a default config if it doesn't exist
        default_config = {"current_mode": "dark", "timer_minutes": 30, "wallpaper_folder": ""}
        save_config(default_config)
        return default_config
    with open(CONFIG_FILE, 'r', encoding='utf-8') as f:
        return json.load(f)

def save_config(config_data):
    """Guarda la configuración en config.json."""
    with open(CONFIG_FILE, 'w', encoding='utf-8') as f:
        json.dump(config_data, f, indent=4)

def save_status(status_data):
    """Guarda el estado actual en status.json."""
    with open(STATUS_FILE, 'w', encoding='utf-8') as f:
        json.dump(status_data, f, indent=4)

def get_wallpaper_type(image_path):
    """Ejecuta matugen y devuelve 'light' o 'dark'."""
    try:
        result = subprocess.run(['matugen', '-j', image_path], capture_output=True, text=True, check=True, encoding='utf-8')
        data = json.loads(result.stdout)
        return data.get('palette', {}).get('type')
    except Exception as e:
        print(f"Error procesando {os.path.basename(image_path)}: {e}", file=sys.stderr)
        return None

def index_wallpapers(wallpaper_dir):
    """Analiza un directorio de fondos, los clasifica y crea symlinks."""
    print(f"Iniciando análisis y clasificación en: {wallpaper_dir}", file=sys.stderr)
    
    light_dir = os.path.join(wallpaper_dir, 'light')
    dark_dir = os.path.join(wallpaper_dir, 'dark')
    os.makedirs(light_dir, exist_ok=True)
    os.makedirs(dark_dir, exist_ok=True)

    for d in [light_dir, dark_dir]:
        for f in os.listdir(d):
            os.remove(os.path.join(d, f))

    for root, _, files in os.walk(wallpaper_dir):
        if root in [light_dir, dark_dir]:
            continue

        for file in files:
            if any(file.lower().endswith(ext) for ext in SUPPORTED_EXTENSIONS):
                image_path = os.path.join(root, file)
                print(f"  -> Analizando: {os.path.basename(image_path)}...", end='', flush=True, file=sys.stderr)
                
                wp_type = get_wallpaper_type(image_path)
                
                if wp_type == 'light':
                    target_dir = light_dir
                elif wp_type == 'dark':
                    target_dir = dark_dir
                else:
                    print(" FALLÓ (tipo no reconocido)", file=sys.stderr)
                    continue
                
                symlink_path = os.path.join(target_dir, file)
                try:
                    src_path = os.path.relpath(image_path, target_dir)
                    os.symlink(src_path, symlink_path)
                    print(f" OK ({wp_type})", file=sys.stderr)
                except FileExistsError:
                    print(f" OK ({wp_type}, ya existía)", file=sys.stderr)
                except Exception as e:
                    print(f" FALLÓ (error de symlink: {e})", file=sys.stderr)

    print(f"\nAnálisis completado. Symlinks creados en subcarpetas 'light' y 'dark'.", file=sys.stderr)


# --- Ejecución Principal ---

if __name__ == '__main__':
    parser = argparse.ArgumentParser(description="""
    Asistente para el demonio de fondos de pantalla 'awwww'. 
    Este script ayuda a clasificar fondos y a controlar el demonio.
    """)
    subparsers = parser.add_subparsers(dest='command', required=True)

    # Comando 'index'
    parser_index = subparsers.add_parser('index', help='Analiza y clasifica los fondos en una carpeta.')
    parser_index.add_argument('path', help='Ruta a la carpeta de fondos de pantalla.')

    # Comando 'daemon'
    parser_daemon = subparsers.add_parser('daemon', help='Inicia el demonio awwww con la configuración guardada.')
    
    # Comando 'next'
    parser_next = subparsers.add_parser('next', help='Pasa al siguiente fondo de pantalla.')

    # Comando 'toggle'
    parser_toggle = subparsers.add_parser('toggle', help='Cambia entre el modo claro y oscuro.')

    # Comando 'status' (para la UI)
    parser_status = subparsers.add_parser('status', help='Devuelve la configuración actual en JSON.')
    
    # Comando 'set-config' (para la UI)
    parser_set_config = subparsers.add_parser('set-config', help='Establece nueva configuración desde un string JSON.')
    parser_set_config.add_argument('json_config', help='String en formato JSON con la nueva configuración.')


    args = parser.parse_args()

    try:
        config = load_config()

        if args.command == 'index':
            config['wallpaper_folder'] = os.path.abspath(args.path)
            save_config(config)
            index_wallpapers(config['wallpaper_folder'])
            print("¡Listo! Ahora puedes iniciar el demonio con 'python manager.py daemon'", file=sys.stderr)
            save_status(config) # Save status after index
            
        elif args.command == 'daemon':
            folder = config.get('wallpaper_folder')
            if not folder:
                raise ValueError("La carpeta de fondos no está configurada. Ejecuta 'index' primero.", file=sys.stderr)
            
            mode = config.get('current_mode', 'dark')
            mode_folder = os.path.join(folder, mode)
            timer = config.get('timer_minutes', 30)

            print(f"Iniciando demonio 'awwww' para la carpeta: {mode_folder}", file=sys.stderr)
            os.execvp('awwww', ['awwww', '-d', mode_folder, '-i', str(timer * 60)])

        elif args.command == 'next':
            subprocess.run(['awwww', 'next'], check=True)
            save_status(config) # Save status after next
            
        elif args.command == 'toggle':
            config['current_mode'] = 'light' if config.get('current_mode') == 'dark' else 'dark'
            save_config(config)
            
            folder = config.get('wallpaper_folder')
            mode_folder = os.path.join(folder, config['current_mode'])
            
            print(f"Cambiando el demonio a la carpeta: {mode_folder}", file=sys.stderr)
            subprocess.run(['awwww', 'set', mode_folder], check=True)
            save_status(config) # Save status after toggle

        elif args.command == 'status':
            save_status(config) # Always save status to file for QML to read

        elif args.command == 'set-config':
            new_config = json.loads(args.json_config)
            config.update(new_config)
            save_config(config)
            save_status(config) # Save status after set-config


    except FileNotFoundError:
        # This catch is for config.json missing *initially*. load_config() now creates it.
        # So this block is mostly for other FileNotFoundError
        error_msg = {"error": "Config file not found after initial creation attempt."}
        save_status(error_msg)
        print(json.dumps(error_msg), file=sys.stderr)
        sys.exit(1)

    except ValueError as e:
        error_msg = {"error": str(e)}
        save_status(error_msg)
        print(json.dumps(error_msg), file=sys.stderr)
        sys.exit(1)
        
    except Exception as e:
        error_msg = {"error": str(e)}
        save_status(error_msg)
        print(json.dumps(error_msg), file=sys.stderr)
        sys.exit(1)