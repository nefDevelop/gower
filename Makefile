# Makefile para Gower - Wallpaper Manager CLI
# Soporta Linux y Windows (cmd.exe / PowerShell / bash)

# --- Variables ---
BINARY_NAME=gower
DIST_DIR=dist
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME?=$(shell date +"%Y-%m-%dT%H:%M:%S%z" 2>/dev/null || echo "unknown")

# Flags de compilación
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# Detección de OS para comandos de sistema
ifeq ($(OS),Windows_NT)
    # Windows (cmd.exe)
    EXE=.exe
    RM=del /F /Q
    RM_DIR=rmdir /S /Q
    MKDIR=mkdir
    # Función para normalizar rutas en Windows (convertir / a \)
    FIX_PATH=$(subst /,\,$(1))
    # Para evitar errores si el archivo no existe
    NULL_OUTPUT=2>NUL
    # Comandos condicionales
    RM_CMD=if exist $(BINARY_NAME)$(EXE) $(RM) $(BINARY_NAME)$(EXE)
    RM_DIST=if exist $(DIST_DIR) $(RM_DIR) $(DIST_DIR)
    MKDIR_LINUX=if not exist "$(DIST_DIR)\linux" $(MKDIR) "$(DIST_DIR)\linux"
    MKDIR_WINDOWS=if not exist "$(DIST_DIR)\windows" $(MKDIR) "$(DIST_DIR)\windows"
else
    # Linux / macOS / WSL
    EXE=
    RM=rm -f
    RM_DIR=rm -rf
    MKDIR=mkdir -p
    FIX_PATH=$(1)
    NULL_OUTPUT=2>/dev/null
    RM_CMD=$(RM) $(BINARY_NAME) $(BINARY_NAME).exe
    RM_DIST=$(RM_DIR) $(DIST_DIR)
    MKDIR_LINUX=$(MKDIR) $(DIST_DIR)/linux
    MKDIR_WINDOWS=$(MKDIR) $(DIST_DIR)/windows
endif

# --- Objetivos ---

.PHONY: all build build-linux build-windows build-all test test-integration lint generate release clean help

all: build

## build: Compila el binario para el sistema operativo actual
build:
	@echo "==> Construyendo $(BINARY_NAME)$(EXE) para $(OS)..."
	go build $(LDFLAGS) -o $(BINARY_NAME)$(EXE) .

## build-linux: Compila el binario para Linux (amd64)
build-linux:
	@echo "==> Construyendo para Linux (amd64)..."
	@$(MKDIR_LINUX) $(NULL_OUTPUT) || true
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(DIST_DIR)/linux/$(BINARY_NAME) .

## build-windows: Compila el binario para Windows (amd64)
build-windows:
	@echo "==> Construyendo para Windows (amd64)..."
	@$(MKDIR_WINDOWS) $(NULL_OUTPUT) || true
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(DIST_DIR)/windows/$(BINARY_NAME).exe .

## build-all: Compila para todas las plataformas soportadas
build-all: build-linux build-windows

## test: Ejecuta todos los tests unitarios
test:
	@echo "==> Ejecutando tests unitarios..."
	go test -v ./...

## test-integration: Ejecuta los tests de integración (usando tags)
test-integration:
	@echo "==> Ejecutando tests de integración..."
	go test -v -tags=integration ./...

## lint: Ejecuta el linter básico (go vet)
lint:
	@echo "==> Analizando código con go vet..."
	go vet ./...

## generate: Regenera los archivos generados (config paths)
generate:
	@echo "==> Regenerando código generado..."
	go generate ./...

## release: Crea el tag de versión y lo publica en todos los remotos
# El workflow .github/workflows/release.yml se dispara al pushear un tag v*,
# compila los binarios y publica la release. Uso: make release RELEASE_VERSION=v0.2.0
release:
	@if [ -z "$(strip $(RELEASE_VERSION))" ]; then \
		echo "==> ERROR: falta la versión. Uso: make release RELEASE_VERSION=v0.2.0"; \
		exit 1; \
	fi
	@case "$(RELEASE_VERSION)" in v*) ;; \
		*) echo "==> ERROR: la versión debe empezar por 'v' (ej. v0.2.0)"; exit 1 ;; \
	esac
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "==> ERROR: el árbol de trabajo tiene cambios sin commitear:"; \
		git status --short; \
		exit 1; \
	fi
	@if git rev-parse -q --verify "refs/tags/$(RELEASE_VERSION)" >/dev/null; then \
		echo "==> ERROR: el tag $(RELEASE_VERSION) ya existe"; exit 1; \
	fi
	@echo "==> Creando el tag $(RELEASE_VERSION) sobre $(shell git rev-parse --short HEAD)..."
	git tag -a "$(RELEASE_VERSION)" -m "$(RELEASE_VERSION)"
	@for remote in $$(git remote); do \
		echo "==> Publicando $(RELEASE_VERSION) en $(remote)..."; \
		git push "$(remote)" "$(RELEASE_VERSION)" || exit 1; \
	done
	@echo "==> Listo. La release se publica sola en cuanto el workflow se ejecuta."

## clean: Elimina los binarios y el directorio de distribución
clean:
	@echo "==> Limpiando artefactos..."
	@$(RM_CMD) $(NULL_OUTPUT) || true
	@$(RM_DIST) $(NULL_OUTPUT) || true

## help: Muestra esta ayuda
# Los comentarios van en la linea anterior al target, asi que se listan las
# lineas "## " y no los targets. Antes se buscaban los targets con un "## " en
# la misma linea, patron que no existe en este Makefile: el grep no encontraba
# nada y, como awk terminaba bien, la lista de respaldo nunca se imprimia.
help:
	@echo "Objetivos disponibles:"
	@echo ""
	@grep -E '^## [a-zA-Z_-]+:' $(MAKEFILE_LIST) | sed -e 's/^## //' | sort | \
		awk 'BEGIN {FS = ":"}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, substr($$0, index($$0, ":") + 1)}' 
