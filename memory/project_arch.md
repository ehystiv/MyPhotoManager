---
name: project-arch
description: Stack attuale di MyPhotoManager dopo migrazione da Fyne a Wails v2 + Vue 3
metadata:
  type: project
---

Migrazione da Fyne GUI a Wails v2 + Vue 3 completata su branch `refactor/web-ui`.

**Why:** Utente voleva libertà totale sul design UI tramite tecnologie web.

**How to apply:** Il progetto ora usa `wails build` (non `go build`) e `wails dev` per sviluppo. Ricorda che il comando wails va eseguito dalla root del progetto, non da sottocartelle.

## Struttura post-migrazione
- `main.go` — bootstrap Wails (embed `frontend/dist`, `options.App`)
- `app.go` — `App` struct con metodi esposti al frontend: `GetPreferences`, `SavePreferences`, `ChooseInputDir`, `ChooseOutputDir`, `ScanPhotos`, `Organize`, `StopOperation`, `BeginWatch`, `StopCurrentWatch`, `OpenInFinder`
- `organizer.go` — logica core invariata (no Fyne)
- `watcher.go` — file watcher invariato (no Fyne)
- `frontend/src/App.vue` — UI Vue 3 (Composition API, script setup)
- `frontend/wailsjs/` — binding auto-generati da Wails

## Preferenze
Salvate in `~/.myphoto/prefs.json` (JSON, non Fyne Preferences API).

## Build
- `wails build` → `build/bin/MyPhotoManager.app`
- `wails dev` → hot-reload con Vite

## Comunicazione Go ↔ Vue
- Chiamate JS→Go: binding in `wailsjs/go/main/App.js`
- Eventi Go→Vue: `wailsruntime.EventsEmit` / `EventsOn` in Vue
  - `log:update` — testo log completo
  - `progress:update` — `{current, total, filename}`
  - `organize:start` — inizio operazione
  - `organize:done` — `OrganizeResult` con `byYear[]`
  - `watch:status` — stringa stato watch

## wails CLI
Installato in `~/go/bin/wails` (v2.12.0). Non è in PATH di default — usare percorso assoluto o `make`.
