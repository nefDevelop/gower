# Makefile para el proyecto Gower

# Variables
BINARY_NAME=gower
GO_FLAGS=-v
DIST_DIR=dist

# --- Objetivos Principales ---

.PHONY: all build build-linux build-windows build-all install test test-integration clean help

all: build

# Construye el binario para el sistema operativo actual
build:
	@echo "Construyendo el binario $(BINARY_NAME) para el SO actual..."
	go build -o $(BINARY_NAME) .

# Construye el binario para Linux
build-linux:
	@echo "Construyendo para Linux (amd64)..."
	@mkdir -p $(DIST_DIR)/linux
	GOOS=linux GOARCH=amd64 go build -o $(DIST_DIR)/linux/$(BINARY_NAME) .

# Construye el binario para Windows
build-windows:
	@echo "Construyendo para Windows (amd64)..."
	@mkdir -p $(DIST_DIR)/windows
	GOOS=windows GOARCH=amd64 go build -o $(DIST_DIR)/windows/$(BINARY_NAME).exe .

# Construye para todas las plataformas soportadas
build-all: build-linux build-windows

# Instala el binario en el GOPATH
install:
	@echo "Instalando $(BINARY_NAME)..."
	go install .

# Ejecuta los tests unitarios
test:
	@echo "Ejecutando tests unitarios..."
	go test $(GO_FLAGS) ./...

# Ejecuta los tests de integración
test-integration:
	@echo "Ejecutando tests de integración..."
	go test $(GO_FLAGS) -tags=integration ./...

# Limpia los artefactos de compilación
clean:
	@echo "Limpiando..."
	@if [ -f $(BINARY_NAME) ]; then rm $(BINARY_NAME); fi
	@if [ -f $(BINARY_NAME).exe ]; then rm $(BINARY_NAME).exe; fi
	@if [ -d $(DIST_DIR) ]; then rm -r $(DIST_DIR); fi

# Muestra esta ayuda
help:
	@echo "Objetivos disponibles:"
	@echo "  make build           - Construye el binario para el SO actual"
	@echo "  make build-linux     - Construye el binario para Linux"
	@echo "  make build-windows   - Construye el binario para Windows"
	@echo "  make build-all       - Construye para todas las plataformas"
	@echo "  make install         - Instala el binario en GOPATH"
	@echo "  make test            - Ejecuta los tests unitarios"
	@echo "  make test-integration- Ejecuta los tests de integración"
	@echo "  make clean           - Limpia los binarios compilados"
	@echo "  make help            - Muestra esta ayuda"
