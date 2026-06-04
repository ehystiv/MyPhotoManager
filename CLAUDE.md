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

I metodi pubblici su `App` vengono esposti automaticamente al frontend. I binding JS/TS corrispondenti si trovano in `frontend/wailsjs/go/main/App.js` e `App.d.ts` — **vanno aggiornati manualmente** quando si aggiungono o rimuovono metodi, perché Wails li rigenera solo a `wails build/dev`.

### File Go

| File | Responsabilità |
|------|----------------|
| `main.go` | Entrypoint Wails, configurazione finestra macOS |
| `app.go` | Metodi esposti al frontend: preferenze, dialog, Organize, Dedupe, Watch, drop |
| `organizer.go` | Raccolta file, EXIF, hash, spostamento/copia/rinomina, deduplicazione, pulizia cartelle vuote |
| `culling.go` | Revisione foto: lista, marcatura (delete/review/ok), applicazione decisioni, lettura EXIF e dati immagine |
| `tree.go` | `PreviewTree`: calcola la struttura di cartelle di destinazione senza spostare file |
| `history.go` | Persistenza storico esecuzioni (`~/.myphoto/history.json`) |
| `watcher.go` | Loop di scansione periodica (ogni 10 s) che chiama `organizePhotos` |

### Struttura output cartelle

```
outputDir/
  raw/          ← tutti i formati RAW (arw, cr2, nef, dng…)
  jpg/          ← JPEG
  png/          ← PNG
  heic/         ← HEIC/HEIF
  webp/         ← WebP
  tiff/         ← TIFF/TIF
  bmp/          ← BMP
  senza_data/   ← file senza EXIF e senza ModTime fallback
  _da_correggere/ ← foto marcate "review" dal culling
```

Le cartelle gestite (`managedFolders`, costruita dinamicamente in `organizer.go`) sono escluse dalla raccolta dei sorgenti. "altri" è mantenuta per retrocompatibilità.

### Parallelismo

`organizePhotos` e `dedupePhotos` usano un pool di goroutine (`min(NumCPU, 8)`). Il pattern è:
- canale `jobs` bufferizzato per distribuire il lavoro
- `ioMu` serializza operazioni su filesystem (resolveConflict + mkdir + transfer)
- `mu` serializza scritture su log e stats
- `counter atomic.Int32` per il progresso senza lock

`PreviewTree` usa un semaforo (`chan struct{}{}` da 8) con goroutine + `sync.WaitGroup`.

### Persistenza (`~/.myphoto/`)

| File | Contenuto |
|------|-----------|
| `prefs.json` | Preferenze utente (`Prefs`) |
| `culling.json` | Marcature di revisione (map `path → mark`) |
| `history.json` | Storico esecuzioni (max 100 voci) |

### Eventi Wails (backend → frontend)

| Evento | Payload | Quando |
|--------|---------|--------|
| `organize:start` | — | inizio operazione |
| `progress:update` | `{current, total, filename, throughput, etaSec, elapsedSec}` | ogni file elaborato (throttle 50 ms) |
| `log:update` | stringa cumulativa | ogni scrittura sul log |
| `organize:done` | `OrganizeResult` | fine organizzazione |
| `dedupe:done` | `DedupeResult` | fine deduplicazione |
| `watch:status` | stringa | cambio stato watch |

### Frontend

Struttura in `frontend/src/`:

```
App.vue                    ← shell principale: tab nav, toolbar, drop overlay
components/
  tabs/                    ← una tab per funzionalità (OrganizzaTab, CullingTab, DedupeTab, WatchTab, LogTab, ResultsTab, OptionsTab)
  TabsNav.vue, Toolbar.vue, Titlebar.vue, StatusBar.vue  ← layout
  Checkbox.vue, ConfirmDialog.vue, DropOverlay.vue, DryRunBanner.vue, EmptyState.vue, PathDisplay.vue, Tooltip.vue
composables/
  useStore.js              ← stato globale reattivo (singleton), tutte le azioni e gli event listener Wails
  useShortcuts.js          ← scorciatoie da tastiera globali
  useLogParser.js          ← parsing del testo di log in voci strutturate
  useTheme.js              ← tema chiaro/scuro
lib/
  presets.js               ← preset di formato per FolderFmt/FileTpl
  utils.js                 ← formatBytes, helpers
style.css                  ← variabili CSS globali (colori, radius), Tailwind base
```

**`useStore.js`** è l'unico punto di stato globale: espone `state` reattivo, computed (`canRun`, `progPct`, `isUnsafeOrganize`) e tutte le azioni. I componenti non chiamano direttamente le API Wails — passano sempre da `useStore`.

### Preferenze utente

Il tipo `Prefs` in `app.go` è sia il formato di persistenza che il parametro passato a `Organize`, `BeginWatch` e `PreviewTree`. I default sono `FolderFmt: "2006_01_02"` e `FileTpl: "photo_{date}_{time}"`. Il campo `RawSplit` controlla la suddivisione dei RAW per estensione (vuoto = tutti in `raw/`).

### Culling (revisione foto)

Il sistema di culling opera sulla `cullingRoot` (outputDir se impostato, altrimenti inputDir). I formati supportati sono JPEG/PNG/WebP (nativi browser) e RAW (miniatura JPEG estratta dall'EXIF). Le marcature sono persistite in `culling.json` e applicate tramite `ApplyCulling(dryRun bool)`.
