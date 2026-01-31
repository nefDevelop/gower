# Makefile para el proyecto Gower

# Variables
BINARY_NAME=gower
GO_FLAGS=-v

# --- Objetivos Principales ---

.PHONY: all build install test test-integration clean help

all: build

# Construye el binario para el sistema operativo actual
build:
	@echo "Construyendo el binario $(BINARY_NAME)..."
	go build -o $(BINARY_NAME) .

# Instala el binario en el GOPATH
install:
	@echo "Instalando $(BINARY_NAME)..."
	go install .

# Ejecuta los tests unitarios (todos excepto los de integración)
test:
	@echo "Ejecutando tests unitarios..."
	go test $(GO_FLAGS) ./...

# Ejecuta solo los tests de integración
test-integration:
	@echo "Ejecutando tests de integración..."
	go test $(GO_FLAGS) -tags=integration ./...

# Limpia los artefactos de compilación
clean:
	@echo "Limpiando..."
	@if [ -f $(BINARY_NAME) ]; then rm $(BINARY_NAME); fi

# Muestra esta ayuda
help:
	@echo "Objetivos disponibles:"
	@echo "  make build           - Construye el binario"
	@echo "  make install         - Instala el binario"
	@echo "  make test            - Ejecuta los tests unitarios"
	@echo "  make test-integration- Ejecuta los tests de integración"
	@echo "  make clean           - Limpia los binarios compilados"
	@echo "  make help            - Muestra esta ayuda"
