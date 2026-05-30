<script setup>
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import {
  GetPreferences, SavePreferences,
  ChooseInputDir, ChooseOutputDir,
  ScanPhotos, Organize, StopOperation,
  BeginWatch, StopCurrentWatch,
  OpenInFinder, HandleDrop, Dedupe
} from '../wailsjs/go/main/App'
import { EventsOn, EventsOff, OnFileDrop, OnFileDropOff } from '../wailsjs/runtime/runtime'

// ── state ────────────────────────────────────────────────────────────────────

const prefs = ref({
  inputDir: '', outputDir: '',
  dryRun: false, copyMode: false, stripMeta: false, modTime: false,
  checkDupes: false, renameOnly: false, cleanDirs: false,
  folderFmt: '2006_01_02', fileTpl: 'photo_{date}_{time}'
})

const scanInfo    = ref('')
const logText     = ref('')
const running     = ref(false)
const progCurrent = ref(0)
const progTotal   = ref(1)
const progFile    = ref('')
const watchActive = ref(false)
const watchStatus = ref('')
const stats       = ref(null)
const lastOutDir  = ref('')
const logEl       = ref(null)
const saving      = ref(false)
const isDragOver  = ref(false)
const dedupeResult  = ref(null)
const dedupeConfirm = ref(false)

// ── computed ──────────────────────────────────────────────────────────────────

const progPct = computed(() =>
  progTotal.value > 0 ? Math.round((progCurrent.value / progTotal.value) * 100) : 0
)

const progInfo = computed(() =>
  running.value
    ? `${progCurrent.value} / ${progTotal.value}  —  ${progFile.value}`
    : ''
)

const inputLabel = computed(() =>
  prefs.value.inputDir || 'Nessuna cartella selezionata'
)

const outputLabel = computed(() =>
  prefs.value.outputDir || 'Stessa di input'
)

// ── lifecycle ─────────────────────────────────────────────────────────────────

onMounted(async () => {
  const p = await GetPreferences()
  prefs.value = p
  if (p.inputDir) doScan(p.inputDir)

  EventsOn('log:update',      (t)  => { logText.value = t; nextTick(scrollLog) })
  EventsOn('progress:update', (d)  => { progCurrent.value = d.current; progTotal.value = d.total; progFile.value = d.filename })
  EventsOn('organize:start',  ()   => { running.value = true; stats.value = null; dedupeResult.value = null; progCurrent.value = 0; progTotal.value = 1 })
  EventsOn('dedupe:done',     (r)  => { dedupeResult.value = r })
  EventsOn('organize:done',   (r)  => {
    running.value = false
    if (r && r.byYear && r.byYear.length > 0) {
      stats.value = r
      lastOutDir.value = prefs.value.outputDir || prefs.value.inputDir
    }
  })
  EventsOn('watch:status', (s) => { watchStatus.value = s })

  OnFileDrop(async (_x, _y, paths) => {
    isDragOver.value = false
    if (!paths || paths.length === 0) return
    const dir = await HandleDrop(paths)
    if (dir) {
      prefs.value.inputDir = dir
      doScan(dir)
    }
  }, false)
})

onUnmounted(() => {
  ['log:update','progress:update','organize:start','organize:done','watch:status','dedupe:done']
    .forEach(e => EventsOff(e))
  OnFileDropOff()
})

// ── methods ───────────────────────────────────────────────────────────────────

function scrollLog() {
  if (logEl.value) logEl.value.scrollTop = logEl.value.scrollHeight
}

async function doScan(dir) {
  scanInfo.value = 'Scansione in corso…'
  const r = await ScanPhotos(dir)
  if (!r || r.total === 0) { scanInfo.value = 'Nessuna foto trovata.'; return }
  let msg = `${r.total} foto  ·  ${r.raw} RAW  ·  ${r.others} altri`
  if (r.noExif > 0) msg += `  ·  ${r.noExif} senza EXIF`
  scanInfo.value = msg
}

async function chooseInput() {
  const dir = await ChooseInputDir()
  if (!dir) return
  prefs.value.inputDir = dir
  await save()
  doScan(dir)
}

async function chooseOutput() {
  const dir = await ChooseOutputDir()
  if (!dir) return
  prefs.value.outputDir = dir
  await save()
}

async function clearOutput() {
  prefs.value.outputDir = ''
  await save()
}

async function save() {
  if (saving.value) return
  saving.value = true
  await SavePreferences({ ...prefs.value })
  saving.value = false
}

async function start() {
  if (!prefs.value.inputDir) { alert('Seleziona prima una directory di input.'); return }
  logText.value = ''
  stats.value = null
  await Organize({ ...prefs.value })
}

async function stop() {
  await StopOperation()
  if (watchActive.value) {
    await StopCurrentWatch()
    watchActive.value = false
  }
  running.value = false
}

async function toggleWatch() {
  if (watchActive.value) {
    if (!prefs.value.inputDir) { alert('Seleziona prima una directory di input.'); watchActive.value = false; return }
    logText.value = ''
    stats.value = null
    running.value = true
    await BeginWatch({ ...prefs.value })
  } else {
    await StopCurrentWatch()
    running.value = false
  }
}

function openFinder() {
  if (lastOutDir.value) OpenInFinder(lastOutDir.value)
}

async function dedupePreview() {
  if (!prefs.value.inputDir) { alert('Seleziona prima una directory di input.'); return }
  logText.value = ''
  stats.value = null
  dedupeResult.value = null
  await Dedupe(prefs.value.inputDir, true)
}

async function dedupeRemove() {
  if (!prefs.value.inputDir) { alert('Seleziona prima una directory di input.'); return }
  dedupeConfirm.value = true
}

async function confirmDedupeRemove() {
  dedupeConfirm.value = false
  logText.value = ''
  stats.value = null
  dedupeResult.value = null
  await Dedupe(prefs.value.inputDir, false)
}

function formatBytes(b) {
  if (b >= 1 << 30) return (b / (1 << 30)).toFixed(1) + ' GB'
  if (b >= 1 << 20) return (b / (1 << 20)).toFixed(1) + ' MB'
  if (b >= 1 << 10) return Math.round(b / (1 << 10)) + ' KB'
  return b + ' B'
}

// debounced save on pref changes (text fields blur)
let saveTimer = null
function debouncedSave() {
  clearTimeout(saveTimer)
  saveTimer = setTimeout(save, 400)
}
</script>

<template>
  <div class="root"
    @dragenter.prevent="isDragOver = true"
    @dragover.prevent
    @dragleave.self="isDragOver = false">

    <!-- Overlay drag & drop -->
    <Transition name="fade">
      <div v-if="isDragOver" class="drop-overlay" @dragleave="isDragOver = false">
        <div class="drop-hint">
          <span class="drop-icon">📁</span>
          <span>Rilascia la cartella qui</span>
        </div>
      </div>
    </Transition>

    <!-- macOS traffic lights spacer -->
    <div class="titlebar-spacer"></div>

    <div class="layout">
      <!-- ── LEFT PANEL ─────────────────────────────────────────── -->
      <div class="left">
        <div class="scroll-area">

          <!-- Cartelle -->
          <div class="card">
            <div class="card-title">Cartelle</div>

            <div class="field">
              <div class="field-label">Input</div>
              <div class="dir-row">
                <span class="dir-path" :class="{ empty: !prefs.inputDir }">{{ inputLabel }}</span>
                <button class="btn-secondary" @click="chooseInput">Scegli…</button>
              </div>
            </div>

            <div class="field">
              <div class="field-label">Output</div>
              <div class="dir-row">
                <span class="dir-path" :class="{ empty: !prefs.outputDir }">{{ outputLabel }}</span>
                <button v-if="prefs.outputDir" class="btn-ghost btn-clear" @click="clearOutput" title="Ripristina">✕</button>
                <button class="btn-secondary" @click="chooseOutput">Scegli…</button>
              </div>
            </div>

            <div class="scan-info" v-if="scanInfo">{{ scanInfo }}</div>
          </div>

          <!-- Opzioni -->
          <div class="card">
            <div class="card-title">Opzioni</div>
            <div class="options-grid">
              <label class="check-label">
                <input type="checkbox" v-model="prefs.dryRun" @change="save">
                <span>Dry-run <span class="muted">(anteprima, nessuna modifica)</span></span>
              </label>
              <label class="check-label" :class="{ disabled: prefs.renameOnly }">
                <input type="checkbox" v-model="prefs.copyMode" :disabled="prefs.renameOnly" @change="save">
                <span>Copia <span class="muted">(mantieni originali)</span></span>
              </label>
              <label class="check-label">
                <input type="checkbox" v-model="prefs.stripMeta" @change="save">
                <span>Rimuovi EXIF dai JPEG</span>
              </label>
              <label class="check-label">
                <input type="checkbox" v-model="prefs.modTime" @change="save">
                <span>Data da filesystem <span class="muted">se EXIF mancante</span></span>
              </label>
              <label class="check-label">
                <input type="checkbox" v-model="prefs.checkDupes" @change="save">
                <span>Salta duplicati <span class="muted">(SHA-256)</span></span>
              </label>
              <label class="check-label">
                <input type="checkbox" v-model="prefs.renameOnly" @change="save">
                <span>Rinomina in-place <span class="muted">(non sposta)</span></span>
              </label>
              <label class="check-label" :class="{ disabled: prefs.renameOnly }">
                <input type="checkbox" v-model="prefs.cleanDirs" :disabled="prefs.renameOnly" @change="save">
                <span>Rimuovi cartelle vuote</span>
              </label>
            </div>
          </div>

          <!-- Struttura -->
          <div class="card">
            <div class="card-title">Struttura</div>
            <div class="field">
              <div class="field-label">Cartelle</div>
              <input type="text" v-model="prefs.folderFmt" :disabled="prefs.renameOnly"
                placeholder="2006_01_02" @blur="save" @input="debouncedSave">
            </div>
            <div class="field">
              <div class="field-label">Nome file</div>
              <input type="text" v-model="prefs.fileTpl"
                placeholder="photo_{date}_{time}" @blur="save" @input="debouncedSave">
            </div>
            <div class="token-hint">
              Token: <code>{date}</code> <code>{time}</code> <code>{datetime}</code>
              <code>{year}</code> <code>{month}</code> <code>{day}</code> <code>{camera}</code>
            </div>
          </div>

          <!-- Watch -->
          <div class="card">
            <div class="card-title">Watch</div>
            <label class="check-label">
              <input type="checkbox" v-model="watchActive" @change="toggleWatch">
              <span>Monitora cartella automaticamente <span class="muted">(ogni 10 s)</span></span>
            </label>
            <div class="watch-status" v-if="watchStatus">{{ watchStatus }}</div>
          </div>

          <!-- Duplicati -->
          <div class="card">
            <div class="card-title">Duplicati</div>
            <div class="dedupe-desc muted">Confronto per hash SHA-256, indipendente dal nome file.</div>
            <template v-if="!dedupeConfirm">
              <div class="dedupe-btns">
                <button class="btn-secondary" @click="dedupePreview" :disabled="running || watchActive">Cerca</button>
                <button class="btn-danger btn-dedupe-rm" @click="dedupeRemove" :disabled="running || watchActive">Rimuovi</button>
              </div>
            </template>
            <template v-else>
              <div class="dedupe-confirm-msg muted">Eliminare definitivamente i duplicati? Un file per gruppo verrà mantenuto.</div>
              <div class="dedupe-btns">
                <button class="btn-danger" @click="confirmDedupeRemove">Conferma</button>
                <button class="btn-secondary" @click="dedupeConfirm = false">Annulla</button>
              </div>
            </template>
          </div>

        </div><!-- /scroll-area -->

        <!-- Action row -->
        <div class="action-row">
          <div class="progress-area">
            <div class="progress-track" v-if="running">
              <div class="progress-fill" :style="{ width: progPct + '%' }"></div>
            </div>
            <div class="progress-info muted" v-if="running">{{ progInfo }}</div>
          </div>
          <div class="btn-group">
            <button class="btn-danger" @click="stop" :disabled="!running && !watchActive">Stop</button>
            <button class="btn-primary" @click="start" :disabled="running || watchActive">▶  Avvia</button>
          </div>
        </div>
      </div>

      <!-- ── RIGHT PANEL ────────────────────────────────────────── -->
      <div class="right">
        <div class="card log-card">
          <div class="card-title">Log</div>
          <div class="log-body" ref="logEl">
            <pre class="log-pre">{{ logText || 'Nessuna operazione avviata.' }}</pre>
          </div>
        </div>

        <div class="card dedupe-card" v-if="dedupeResult">
          <div class="card-title">Risultato deduplicazione</div>
          <div class="dedupe-stats">
            <div class="dedupe-row"><span class="dedupe-label">Scansionate</span><span class="dedupe-val">{{ dedupeResult.scanned }} foto</span></div>
            <div class="dedupe-row"><span class="dedupe-label">Gruppi</span><span class="dedupe-val">{{ dedupeResult.groups }}</span></div>
            <div class="dedupe-row" v-if="dedupeResult.removed > 0">
              <span class="dedupe-label">{{ dedupeResult.dryRun ? 'Da rimuovere' : 'Rimossi' }}</span>
              <span class="dedupe-val">{{ dedupeResult.removed }} file · {{ formatBytes(dedupeResult.freed) }}</span>
            </div>
            <div class="dedupe-row" v-else><span class="dedupe-label muted">Nessun duplicato trovato</span></div>
          </div>
        </div>

        <div class="card stats-card" v-if="stats && stats.byYear && stats.byYear.length">
          <div class="card-title">Statistiche per anno</div>
          <div class="stats-list">
            <div class="stat-row" v-for="s in stats.byYear" :key="s.year">
              <span class="stat-year">{{ s.year }}</span>
              <span class="stat-sep">→</span>
              <span class="stat-count">{{ s.count }} foto</span>
            </div>
          </div>
          <button class="btn-secondary btn-finder" @click="openFinder">
            Apri cartella nel Finder
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* ── Layout ─────────────────────────────────────────────────────────────────── */
.root {
  display: flex;
  flex-direction: column;
  height: 100vh;
  overflow: hidden;
}

.titlebar-spacer {
  height: 28px;
  flex-shrink: 0;
  -webkit-app-region: drag;
}

.layout {
  display: flex;
  flex: 1;
  min-height: 0;
  gap: 0;
}

/* ── Left panel ──────────────────────────────────────────────────────────────── */
.left {
  width: 420px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  border-right: 1px solid var(--border);
  min-height: 0;
}

.scroll-area {
  flex: 1;
  overflow-y: auto;
  padding: 12px 12px 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

/* ── Right panel ─────────────────────────────────────────────────────────────── */
.right {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  padding: 12px;
  gap: 10px;
}

.log-card {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.log-body {
  flex: 1;
  overflow-y: auto;
  background: var(--bg);
  border-radius: 6px;
  padding: 10px;
  margin-top: 8px;
  min-height: 0;
}

.log-pre {
  font-family: 'SF Mono', 'Fira Code', monospace;
  font-size: 11.5px;
  line-height: 1.55;
  color: var(--text-muted);
  white-space: pre-wrap;
  word-break: break-all;
}

/* ── Card ────────────────────────────────────────────────────────────────────── */
.card {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 12px 14px;
}

.card-title {
  font-weight: 600;
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: .06em;
  color: var(--text-muted);
  margin-bottom: 10px;
}

/* ── Fields ──────────────────────────────────────────────────────────────────── */
.field {
  margin-bottom: 8px;
}

.field-label {
  font-size: 11.5px;
  color: var(--text-muted);
  margin-bottom: 4px;
}

.dir-row {
  display: flex;
  align-items: center;
  gap: 6px;
}

.dir-path {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12.5px;
}

.dir-path.empty {
  color: var(--text-muted);
  font-style: italic;
}

.scan-info {
  margin-top: 8px;
  font-size: 12px;
  color: var(--accent);
  font-style: italic;
}

/* ── Checkboxes ─────────────────────────────────────────────────────────────── */
.options-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px 12px;
}

.check-label {
  display: flex;
  align-items: flex-start;
  gap: 7px;
  cursor: pointer;
  line-height: 1.4;
}

.check-label.disabled {
  opacity: .4;
  pointer-events: none;
}

.check-label input { margin-top: 1px; }

/* ── Tokens ──────────────────────────────────────────────────────────────────── */
.token-hint {
  margin-top: 8px;
  font-size: 11px;
  color: var(--text-muted);
  line-height: 1.8;
}

.token-hint code {
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: 4px;
  padding: 1px 5px;
  font-size: 10.5px;
  margin-right: 2px;
}

/* ── Watch ───────────────────────────────────────────────────────────────────── */
.watch-status {
  margin-top: 8px;
  font-size: 12px;
  color: var(--success);
  font-style: italic;
}

/* ── Action row ──────────────────────────────────────────────────────────────── */
.action-row {
  border-top: 1px solid var(--border);
  padding: 10px 12px;
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
}

.progress-area {
  flex: 1;
  min-width: 0;
}

.progress-track {
  height: 4px;
  background: var(--border);
  border-radius: 2px;
  overflow: hidden;
  margin-bottom: 4px;
}

.progress-fill {
  height: 100%;
  background: var(--accent);
  border-radius: 2px;
  transition: width .15s;
}

.progress-info {
  font-size: 11px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.btn-group {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}

/* ── Stats ───────────────────────────────────────────────────────────────────── */
.stats-card { flex-shrink: 0; }

.stats-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin-bottom: 10px;
}

.stat-row {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
}

.stat-year  { font-weight: 600; min-width: 42px; }
.stat-sep   { color: var(--text-muted); }
.stat-count { color: var(--text-muted); }

.btn-finder { margin-top: 2px; width: 100%; }

/* ── Buttons ─────────────────────────────────────────────────────────────────── */
.btn-primary {
  background: var(--accent);
  color: #fff;
  font-weight: 600;
  padding: 7px 18px;
}
.btn-primary:hover:not(:disabled) { background: var(--accent-h); }

.btn-danger {
  background: var(--danger);
  color: #fff;
  font-weight: 600;
  padding: 7px 14px;
}
.btn-danger:hover:not(:disabled) { background: var(--danger-h); }

.btn-secondary {
  background: var(--border);
  color: var(--text);
  padding: 5px 12px;
  white-space: nowrap;
}
.btn-secondary:hover:not(:disabled) { background: #4b5563; }

.btn-ghost {
  background: transparent;
  color: var(--text-muted);
  padding: 4px 8px;
}
.btn-ghost:hover:not(:disabled) { color: var(--text); background: var(--border); }

.btn-clear {
  font-size: 11px;
  padding: 4px 7px;
}

/* ── Drag & drop overlay ─────────────────────────────────────────────────────── */
.drop-overlay {
  position: fixed;
  inset: 0;
  z-index: 100;
  background: rgba(59, 130, 246, .15);
  border: 2px dashed var(--accent);
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  pointer-events: all;
}

.drop-hint {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  color: var(--accent);
  font-size: 16px;
  font-weight: 600;
}

.drop-icon { font-size: 48px; line-height: 1; }

.fade-enter-active, .fade-leave-active { transition: opacity .15s; }
.fade-enter-from, .fade-leave-to { opacity: 0; }

/* ── Duplicati ───────────────────────────────────────────────────────────────── */
.dedupe-desc {
  font-size: 11.5px;
  margin-bottom: 10px;
}

.dedupe-confirm-msg {
  font-size: 11.5px;
  margin-bottom: 10px;
  line-height: 1.4;
}

.dedupe-btns {
  display: flex;
  gap: 8px;
}

.btn-dedupe-rm {
  padding: 5px 12px;
  font-size: 13px;
}

.dedupe-card { flex-shrink: 0; }

.dedupe-stats {
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.dedupe-row {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
}

.dedupe-label {
  color: var(--text-muted);
  min-width: 100px;
  font-size: 12px;
}

.dedupe-val {
  font-weight: 500;
}

/* ── Utilities ───────────────────────────────────────────────────────────────── */
.muted { color: var(--text-muted); }
</style>
