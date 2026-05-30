import { reactive, computed, watch } from 'vue'
import {
  GetPreferences, SavePreferences,
  ScanPhotos, Organize, StopOperation,
  BeginWatch, StopCurrentWatch,
  ChooseInputDir, ChooseOutputDir,
  HandleDrop, Dedupe, OpenInFinder,
  ResetPreferences, ClearRecents,
  NotifyDesktop, ShowAbout,
} from '../../wailsjs/go/main/App'
import { EventsOn, EventsOff, OnFileDrop, OnFileDropOff } from '../../wailsjs/runtime/runtime'
import { toast } from 'vue-sonner'
import { formatBytes } from '../lib/utils'

const DEFAULTS = {
  inputDir: '',
  outputDir: '',
  dryRun: false,
  copyMode: false,
  stripMeta: false,
  modTime: false,
  checkDupes: false,
  renameOnly: false,
  cleanDirs: false,
  folderFmt: '2006_01_02',
  fileTpl: 'photo_{date}_{time}',
  recents: [],
  confirmedUnsafeOnce: false,
}

const state = reactive({
  prefs: { ...DEFAULTS },
  initialized: false,
  scanInfo: '',
  scanResult: null,
  scanning: false,
  logText: '',
  running: false,
  progress: { current: 0, total: 1, filename: '', throughput: 0, etaSec: 0, elapsedSec: 0 },
  watchActive: false,
  watchStatus: '',
  watchLastScanAt: null,
  stats: null,
  dedupeResult: null,
  lastOutDir: '',
  isDragOver: false,
  activeTab: localStorage.getItem('activeTab') || 'options',
  runStartedAt: 0,
})

watch(() => state.activeTab, (v) => localStorage.setItem('activeTab', v))

const hasInputDir = computed(() => !!state.prefs.inputDir)
const canRun = computed(() => hasInputDir.value && !state.running && !state.watchActive)
const progPct = computed(() =>
  state.progress.total > 0
    ? Math.round((state.progress.current / state.progress.total) * 100)
    : 0
)
const isUnsafeOrganize = computed(() =>
  !state.prefs.dryRun && !state.prefs.copyMode && !state.prefs.renameOnly
)

let saveTimer = null
async function persist() {
  clearTimeout(saveTimer)
  saveTimer = setTimeout(async () => {
    try {
      await SavePreferences({ ...state.prefs })
    } catch (e) {
      console.error('SavePreferences failed', e)
    }
  }, 250)
}

async function doScan(dir) {
  state.scanning = true
  state.scanInfo = 'Scansione in corso…'
  state.scanResult = null
  try {
    const r = await ScanPhotos(dir)
    state.scanResult = r
    if (!r || r.total === 0) {
      state.scanInfo = 'Nessuna foto trovata.'
      return
    }
    let msg = `${r.total} foto · ${r.raw} RAW · ${r.others} altri`
    if (r.noExif > 0) msg += ` · ${r.noExif} senza EXIF`
    if (r.totalBytes) msg += ` · ${formatBytes(r.totalBytes)}`
    state.scanInfo = msg
  } catch (e) {
    state.scanInfo = 'Errore scansione.'
    console.error(e)
  } finally {
    state.scanning = false
  }
}

async function chooseInput() {
  const dir = await ChooseInputDir()
  if (!dir) return
  state.prefs.inputDir = dir
  // Reload prefs to pick up updated Recents from backend.
  const fresh = await GetPreferences()
  state.prefs.recents = fresh.recents || []
  doScan(dir)
}

async function chooseOutput() {
  const dir = await ChooseOutputDir()
  if (!dir) return
  state.prefs.outputDir = dir
  await persist()
}

function clearOutput() {
  state.prefs.outputDir = ''
  persist()
}

async function start() {
  if (!state.prefs.inputDir) {
    toast.error('Seleziona prima una cartella di input')
    return
  }
  state.logText = ''
  state.stats = null
  state.dedupeResult = null
  await Organize({ ...state.prefs })
}

async function stop() {
  await StopOperation()
  if (state.watchActive) {
    await StopCurrentWatch()
    state.watchActive = false
  }
  state.running = false
}

async function toggleWatch(active) {
  state.watchActive = active
  if (active) {
    if (!state.prefs.inputDir) {
      toast.error('Seleziona prima una cartella di input')
      state.watchActive = false
      return
    }
    state.logText = ''
    state.stats = null
    state.running = true
    await BeginWatch({ ...state.prefs })
  } else {
    await StopCurrentWatch()
    state.running = false
  }
}

async function dedupePreview() {
  if (!state.prefs.inputDir) {
    toast.error('Seleziona prima una cartella di input')
    return
  }
  state.logText = ''
  state.stats = null
  state.dedupeResult = null
  state.activeTab = 'log'
  await Dedupe(state.prefs.inputDir, true)
}

async function dedupeRemove() {
  state.logText = ''
  state.stats = null
  state.dedupeResult = null
  state.activeTab = 'log'
  await Dedupe(state.prefs.inputDir, false)
}

function openOutputInFinder() {
  if (state.lastOutDir) OpenInFinder(state.lastOutDir)
}

function setRecent(dir) {
  state.prefs.inputDir = dir
  persist()
  doScan(dir)
}

async function resetPrefs() {
  const p = await ResetPreferences()
  Object.assign(state.prefs, { ...DEFAULTS, ...p })
  toast.success('Preferenze ripristinate')
}

async function clearRecentsList() {
  await ClearRecents()
  state.prefs.recents = []
}

function bindEvents() {
  EventsOn('log:update', (t) => { state.logText = t })
  EventsOn('progress:update', (d) => {
    state.progress.current = d.current
    state.progress.total = d.total
    state.progress.filename = d.filename
    state.progress.throughput = d.throughput || 0
    state.progress.etaSec = d.etaSec || 0
    state.progress.elapsedSec = d.elapsedSec || 0
  })
  EventsOn('organize:start', () => {
    state.running = true
    state.stats = null
    state.dedupeResult = null
    state.runStartedAt = Date.now()
    state.progress.current = 0
    state.progress.total = 1
    state.progress.filename = ''
    state.progress.throughput = 0
    state.progress.etaSec = 0
  })
  EventsOn('dedupe:done', (r) => {
    state.dedupeResult = r
    if (r && r.removed > 0) {
      const action = r.dryRun ? 'da spostare nel cestino' : 'spostati nel cestino'
      const title = `${r.removed} duplicati ${action}`
      const description = `${r.groups} gruppi · liberati ${formatBytes(r.freed)}`
      toast.success(title, { description })
      maybeDesktopNotify(title, description)
    } else if (r) {
      toast.info('Nessun duplicato trovato')
    }
  })
  EventsOn('organize:done', (r) => {
    state.running = false
    if (r && r.byYear && r.byYear.length > 0) {
      state.stats = r
      state.lastOutDir = state.prefs.outputDir || state.prefs.inputDir
      const title = `${r.moved} foto organizzate`
      const description = `${r.raw} RAW · ${r.others} altri${r.dupes ? ` · ${r.dupes} duplicati saltati` : ''}`
      toast.success(title, {
        description,
        action: {
          label: 'Apri nel Finder',
          onClick: () => OpenInFinder(state.lastOutDir),
        },
      })
      maybeDesktopNotify(title, description)
      state.activeTab = 'results'
    }
  })
  EventsOn('watch:status', (s) => {
    state.watchStatus = s
    if (s && s.startsWith('Ultima scansione')) {
      state.watchLastScanAt = Date.now()
    }
  })

  OnFileDrop(async (_x, _y, paths) => {
    state.isDragOver = false
    if (!paths || paths.length === 0) return
    const dir = await HandleDrop(paths)
    if (dir) {
      state.prefs.inputDir = dir
      const fresh = await GetPreferences()
      state.prefs.recents = fresh.recents || []
      doScan(dir)
    }
  }, false)
}

function unbindEvents() {
  ;['log:update','progress:update','organize:start','organize:done','watch:status','dedupe:done']
    .forEach(e => EventsOff(e))
  OnFileDropOff()
}

async function init() {
  if (state.initialized) return
  const p = await GetPreferences()
  Object.assign(state.prefs, DEFAULTS, p)
  bindEvents()
  if (state.prefs.inputDir) doScan(state.prefs.inputDir)
  state.initialized = true
}

function maybeDesktopNotify(title, body) {
  // Notifica solo per operazioni "lunghe" (>=5s) e quando la finestra non è in focus.
  const elapsed = state.runStartedAt ? (Date.now() - state.runStartedAt) / 1000 : 0
  if (elapsed >= 5 && !document.hasFocus()) {
    NotifyDesktop(title, body)
  }
}

export function useStore() {
  return {
    state,
    hasInputDir,
    canRun,
    progPct,
    isUnsafeOrganize,
    init,
    unbindEvents,
    persist,
    doScan,
    chooseInput,
    chooseOutput,
    clearOutput,
    start,
    stop,
    toggleWatch,
    dedupePreview,
    dedupeRemove,
    openOutputInFinder,
    setRecent,
    resetPrefs,
    clearRecentsList,
    showAbout: ShowAbout,
  }
}
