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
    # Install specific for Windows
    INSTALL_COPY_CMD=@echo "On Windows, please copy $(BINARY_NAME)$(EXE) from the current directory to a location in your PATH."
    INSTALL_CHMOD_CMD=
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
    # Install specific for Linux/macOS
    INSTALL_TARGET_DIR=/usr/local/bin
    INSTALL_COPY_CMD=sudo cp $(BINARY_NAME) $(INSTALL_TARGET_DIR)
    INSTALL_CHMOD_CMD=sudo chmod +x $(INSTALL_TARGET_DIR)/$(BINARY_NAME)
endif

# --- Objetivos ---

.PHONY: all build build-linux build-windows build-all test test-integration lint clean install help

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

## install: Instala el binario en el sistema
install: build
ifeq ($(OS),Windows_NT)
	@$(INSTALL_COPY_CMD)
else
	@echo "==> Instalando $(BINARY_NAME) en $(INSTALL_TARGET_DIR)..."
	$(INSTALL_COPY_CMD)
	$(INSTALL_CHMOD_CMD)
	@echo "==> Instalación completada. Puede que necesites ejecutar 'hash -r' o abrir una nueva terminal."
endif

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

## clean: Elimina los binarios y el directorio de distribución
clean:
	@echo "==> Limpiando artefactos..."
	@$(RM_CMD) $(NULL_OUTPUT) || true
	@$(RM_DIST) $(NULL_OUTPUT) || true

## help: Muestra esta ayuda
help:
	@echo "Objetivos disponibles:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' || \
	(echo "  build           - Compila para el OS actual" && \
	 echo "  build-linux     - Compila para Linux" && \
	 echo "  build-windows   - Compila para Windows" && \
	 echo "  install         - Instala el binario en el sistema" && \
	 echo "  test            - Ejecuta tests" && \
	 echo "  clean           - Limpia binarios")
