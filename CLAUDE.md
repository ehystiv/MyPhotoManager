# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
make dev        # avvia con hot-reload (wails dev) — frontend Vite + backend Go insieme
make build      # compila binario con frontend embedded
make app        # crea bundle .app macOS (darwin/arm64)
make install    # app + copia in /Applications
make clean      # rimuove build/bin, frontend/dist, frontend/node_modules

go build ./...  # verifica compilazione Go senza invocare Wails
```

Per lanciare solo il frontend:
```bash
cd frontend && npm run dev
```

Non esistono test automatizzati nel progetto.

## Architettura

App desktop macOS costruita con **Wails v2**: Go come backend, Vue 3 (Vite) come frontend. La comunicazione avviene tramite binding generati automaticamente e un sistema di eventi.

### Flusso dati

```
Frontend (Vue)  ──chiamata diretta──▶  app.go (metodi esportati su App)
                ◀──EventsEmit──────    app.go
```

I metodi pubblici su `App` (in `app.go`) vengono esposti automaticamente al frontend. I binding JS/TS corrispondenti si trovano in `frontend/wailsjs/go/main/App.js` e `App.d.ts` — **vanno aggiornati manualmente** quando si aggiungono o rimuovono metodi, perché Wails li rigenera solo a `wails build/dev`.

### File Go

| File | Responsabilità |
|------|----------------|
| `main.go` | Entrypoint Wails, configurazione finestra macOS |
| `app.go` | Metodi esposti al frontend: preferenze, dialog, Organize, Dedupe, Watch, drop |
| `organizer.go` | Tutta la logica di elaborazione foto: raccolta file, EXIF, hash, spostamento/copia/rinomina, deduplicazione, pulizia cartelle vuote |
| `watcher.go` | Loop di scansione periodica (ogni 10 s) che chiama `organizePhotos` |

### Struttura output cartelle

```
outputDir/
  raw/          ← file con estensioni RAW (arw, cr2, nef…)
  altri/        ← JPEG, PNG, HEIC…
  senza_data/   ← file senza EXIF e senza ModTime fallback
```

Queste tre cartelle (`managedFolders`) vengono escluse automaticamente dalla raccolta dei file sorgente per evitare di riprocessare file già organizzati.

### Parallelismo

`organizePhotos` e `dedupePhotos` usano un pool di goroutine (`min(NumCPU, 8)`). Il pattern è:
- canale `jobs` bufferizzato per distribuire il lavoro
- `ioMu` serializza operazioni su filesystem (resolveConflict + mkdir + transfer)
- `mu` serializza scritture su log e stats
- `counter atomic.Int32` per il progresso senza lock

### Eventi Wails (backend → frontend)

| Evento | Payload | Quando |
|--------|---------|--------|
| `organize:start` | — | inizio operazione |
| `progress:update` | `{current, total, filename}` | ogni file elaborato |
| `log:update` | stringa cumulativa | ogni scrittura sul log |
| `organize:done` | `OrganizeResult` | fine organizzazione |
| `dedupe:done` | `DedupeResult` | fine deduplicazione |
| `watch:status` | stringa | cambio stato watch |

### Frontend

Single-file component `frontend/src/App.vue` (script setup + template + style scoped). Non ci sono componenti separati. Le variabili CSS globali (colori, radius) sono in `frontend/src/style.css`.

### Preferenze utente

Salvate in `~/.myphoto/prefs.json` tramite `loadPrefs`/`savePrefs`. Il tipo `Prefs` in `app.go` è sia il formato di persistenza che il parametro passato a `Organize` e `BeginWatch`.
