APP_NAME := MyPhotoManager
BINARY   := myphoto
WAILS    := ~/go/bin/wails

.PHONY: all dev build app clean install

## Default: crea il bundle .app
all: app

## Avvia in modalità sviluppo con hot-reload
dev:
	$(WAILS) dev

## Compila il binario con frontend integrato
build:
	$(WAILS) build

## Crea il bundle .app macOS
app:
	$(WAILS) build -platform darwin/arm64 -o $(BINARY)

## Copia il .app in /Applications
install: app
	cp -r build/bin/$(APP_NAME).app /Applications/$(APP_NAME).app
	@echo "Installato in /Applications/$(APP_NAME).app"

## Rimuove artefatti di build
clean:
	rm -rf build/bin
	rm -rf frontend/dist
	rm -rf frontend/node_modules
