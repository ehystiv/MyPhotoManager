# MyPhotoManager

App desktop macOS per organizzare, revisionare e deduplicare la propria libreria fotografica. Costruita con [Wails v2](https://wails.io) (Go + Vue 3).

## Funzionalità

- **Organizzazione automatica** — sposta o copia le foto in cartelle strutturate per data, leggendo i metadati EXIF. Supporta JPEG, PNG, HEIC, WebP, TIFF, BMP e tutti i principali formati RAW (ARW, CR2, NEF, DNG…).
- **Formato cartelle e nomi file personalizzabili** — pattern tipo `2006/01` per le cartelle e template tipo `photo_{date}_{time}` per i file.
- **Rilevamento duplicati** — scansiona e rimuove i duplicati tramite hash del contenuto.
- **Culling (revisione foto)** — interfaccia per marcare le foto come *da eliminare*, *da rivedere* o *ok*, con anteprima e metadati EXIF. Supporta miniature JPEG estratte dai file RAW.
- **Watch mode** — monitoraggio automatico della cartella sorgente ogni 10 secondi.
- **Anteprima struttura** — mostra la struttura di destinazione prima di eseguire qualsiasi spostamento.
- **Storico esecuzioni** — registro delle ultime 100 operazioni in `~/.myphoto/history.json`.
- **Modalità dry-run** — simula l'operazione senza toccare i file.

## Struttura cartelle di output

```
outputDir/
  raw/              ← tutti i formati RAW
  jpg/              ← JPEG
  png/              ← PNG
  heic/             ← HEIC/HEIF
  webp/             ← WebP
  tiff/             ← TIFF/TIF
  bmp/              ← BMP
  senza_data/       ← file senza EXIF e senza data di modifica
  _da_correggere/   ← foto marcate "review" dal culling
```

## Requisiti

- macOS (arm64)
- [Go 1.22+](https://go.dev)
- [Wails v2](https://wails.io/docs/gettingstarted/installation) (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)
- [Node.js](https://nodejs.org) (per il frontend Vue)

## Installazione e sviluppo

```bash
# Avvia in modalità sviluppo con hot-reload (frontend Vite + backend Go)
make dev

# Compila il binario con frontend embedded
make build

# Crea il bundle .app per macOS
make app

# Compila e installa in /Applications
make install

# Rimuove artefatti di build
make clean
```

Per lavorare solo sul frontend:

```bash
cd frontend && npm run dev
```

Per verificare la compilazione Go senza invocare Wails:

```bash
go build ./...
```

## Persistenza

I dati dell'app vengono salvati in `~/.myphoto/`:

| File | Contenuto |
|------|-----------|
| `prefs.json` | Preferenze utente |
| `culling.json` | Marcature di revisione (path → mark) |
| `history.json` | Storico delle ultime 100 esecuzioni |

## Architettura

Backend Go (`app.go`, `organizer.go`, `culling.go`, `tree.go`, `history.go`, `watcher.go`) esposto al frontend Vue 3 tramite i binding generati da Wails. La comunicazione avviene tramite chiamate dirette e un sistema di eventi (`wailsruntime.EventsEmit`).

Il parallelismo è gestito con goroutine pool (`min(NumCPU, 8)`), mutex per le operazioni sul filesystem e contatori atomici per il progresso.

## Autore

Stefano Bichicchi — bichicchi.stefano@gmail.com
