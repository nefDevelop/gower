# Proyecto WallpaperWidget - Root Makefile
# Delega los comandos al Makefile de gower

.PHONY: all build build-linux build-windows build-all test lint clean help

all:
	$(MAKE) -C gower all

build:
	$(MAKE) -C gower build

build-linux:
	$(MAKE) -C gower build-linux

build-windows:
	$(MAKE) -C gower build-windows

build-all:
	$(MAKE) -C gower build-all

test:
	$(MAKE) -C gower test

lint:
	$(MAKE) -C gower lint

clean:
	$(MAKE) -C gower clean

help:
	@echo "WallpaperWidget project root"
	@echo "Este Makefile delega las tareas al proyecto principal 'gower'"
	@echo ""
	@$(MAKE) -s -C gower help
