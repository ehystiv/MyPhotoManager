APP_NAME  := MyPhotoManager
BINARY    := myphoto
FYNE      := ~/go/bin/fyne

.PHONY: all build app clean run install

## Default: crea il bundle .app
all: app

## Compila il binario grezzo (senza bundle)
build:
	go build -o $(BINARY) ./...

## Crea il bundle .app macOS (doppio clic, niente terminale)
app:
	$(FYNE) package -os darwin -name "$(APP_NAME)"

## Esegui direttamente (con terminale, utile per debug)
run:
	go run ./...

## Copia il .app in /Applications
install: app
	cp -r $(APP_NAME).app /Applications/$(APP_NAME).app
	@echo "✅ Installato in /Applications/$(APP_NAME).app"

## Rimuove artefatti di build
clean:
	rm -f $(BINARY)
	rm -rf $(APP_NAME).app
