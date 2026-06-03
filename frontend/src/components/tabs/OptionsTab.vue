<script setup>
import { ref, watch, onMounted, nextTick, computed } from 'vue'
import { useDebounceFn } from '@vueuse/core'
import { useStore } from '../../composables/useStore'
import { FormatPreview, PreviewTree } from '../../../wailsjs/go/main/App'
import { RotateCcw, ArrowRight, Hash, AlertTriangle, Info, Sparkles, Settings2, FolderTree } from '@lucide/vue'
import Checkbox from '../Checkbox.vue'
import {
  FOLDER_PRESETS, FILE_PRESETS, CUSTOM_ID,
  matchFolderPreset, matchFilePreset,
} from '../../lib/presets'

const { state, persist, resetPrefs } = useStore()

const treeData = ref(null)
const treeLoading = ref(false)

async function computeTree() {
  if (!state.prefs.inputDir) return
  treeLoading.value = true
  try {
    treeData.value = await PreviewTree(state.prefs)
  } catch (e) {
    console.error('PreviewTree failed', e)
  } finally {
    treeLoading.value = false
  }
}

const debouncedComputeTree = useDebounceFn(computeTree, 800)

watch(
  () => [
    state.prefs.inputDir,
    state.prefs.outputDir,
    state.prefs.folderFmt,
    state.prefs.rawSplit,
    state.prefs.modTime,
    state.prefs.renameOnly,
  ],
  () => debouncedComputeTree(),
  { immediate: true },
)

const preview = ref({ folder: '', file: '', full: '' })
const folderInput = ref(null)
const fileInput = ref(null)
const lastFocused = ref('file')

const TOKENS = ['{date}', '{time}', '{datetime}', '{year}', '{month}', '{day}', '{camera}']

const RAW_SPLITS = [
  { value: '',            label: 'Disattivata' },
  { value: 'camera',      label: 'Fotocamera (marca + modello)' },
  { value: 'lens',        label: 'Obiettivo' },
  { value: 'iso',         label: 'ISO (a fasce)' },
  { value: 'camera_lens', label: 'Fotocamera → Obiettivo' },
]

const rawSplitExample = computed(() => {
  switch (state.prefs.rawSplit) {
    case 'camera':      return 'Sony_ILCE-7M4/'
    case 'lens':        return 'FE_24-70mm_F2.8_GM/'
    case 'iso':         return 'iso_200-800/'
    case 'camera_lens': return 'Sony_ILCE-7M4/FE_24-70mm/'
    default:            return ''
  }
})

/* ---------- Vista Semplice: preset ↔ pattern ---------- */

const folderPreset = computed({
  get: () => matchFolderPreset(state.prefs.folderFmt),
  set: (id) => {
    const p = FOLDER_PRESETS.find((x) => x.id === id)
    if (!p) return
    state.prefs.folderFmt = p.folderFmt
    // refreshPreview e persist sono gestiti dai watch su folderFmt
  },
})

const filePreset = computed({
  get: () => matchFilePreset(state.prefs.fileTpl),
  set: (id) => {
    const p = FILE_PRESETS.find((x) => x.id === id)
    if (!p) return
    state.prefs.fileTpl = p.fileTpl
  },
})

const isCustomLayout = computed(() =>
  folderPreset.value === CUSTOM_ID || filePreset.value === CUSTOM_ID
)

/* ---------- Validazioni (vista Avanzate) ---------- */

const folderWarning = computed(() => {
  const v = state.prefs.folderFmt || ''
  if (state.prefs.renameOnly) return ''
  if (!v.trim()) return { level: 'error', msg: 'Formato cartella vuoto.' }
  const hasDate = /2006|01|02/.test(v)
  if (!hasDate) {
    return { level: 'error', msg: 'Nessun riferimento alla data: tutte le foto finirebbero in una sola cartella.' }
  }
  return ''
})

const fileWarning = computed(() => {
  const v = state.prefs.fileTpl || ''
  if (!v.trim()) return { level: 'error', msg: 'Nome file vuoto.' }
  const hasUniq = /\{date\}|\{time\}|\{datetime\}/.test(v)
  if (!hasUniq) {
    return { level: 'warn', msg: 'Nessun riferimento a data/ora: rischio di nomi duplicati (verrà aggiunto _1, _2…).' }
  }
  if (!/\{(date|time|datetime|year|month|day|camera)\}/.test(v)) {
    return { level: 'warn', msg: 'Nessun segnaposto: tutti i file avranno lo stesso nome.' }
  }
  return ''
})

async function refreshPreview() {
  try {
    preview.value = await FormatPreview(state.prefs.folderFmt, state.prefs.fileTpl)
  } catch {}
}
const debouncedRefresh = useDebounceFn(refreshPreview, 120)

watch(() => state.prefs.folderFmt, () => { debouncedRefresh(); persist() })
watch(() => state.prefs.fileTpl,   () => { debouncedRefresh(); persist() })

onMounted(refreshPreview)

function onCheckChange() {
  // Forza la persistenza istantanea dei checkbox (no debounce).
  persist()
}

function insertToken(token) {
  const target = lastFocused.value === 'folder' ? folderInput.value : fileInput.value
  const key = lastFocused.value === 'folder' ? 'folderFmt' : 'fileTpl'
  if (!target) {
    state.prefs[key] = (state.prefs[key] || '') + token
    return
  }
  const start = target.selectionStart ?? state.prefs[key].length
  const end = target.selectionEnd ?? state.prefs[key].length
  const v = state.prefs[key]
  state.prefs[key] = v.slice(0, start) + token + v.slice(end)
  nextTick(() => {
    target.focus()
    const pos = start + token.length
    target.setSelectionRange(pos, pos)
  })
}
</script>

<template>
  <div class="options-tab">
    <!-- Toggle vista -->
    <div class="mode-switch" role="tablist" aria-label="Livello di dettaglio">
      <button
        class="seg"
        :class="{ active: state.optionsMode === 'simple' }"
        @click="state.optionsMode = 'simple'"
        role="tab"
        :aria-selected="state.optionsMode === 'simple'"
      >
        <Sparkles :size="13" /> Semplice
      </button>
      <button
        class="seg"
        :class="{ active: state.optionsMode === 'advanced' }"
        @click="state.optionsMode = 'advanced'"
        role="tab"
        :aria-selected="state.optionsMode === 'advanced'"
      >
        <Settings2 :size="13" /> Avanzate
      </button>
    </div>

    <!-- ============ VISTA SEMPLICE ============ -->
    <template v-if="state.optionsMode === 'simple'">
      <section>
        <header class="sec-head">
          <h3>Organizza le foto</h3>
          <p>Scegli come raggruppare le foto in cartelle.</p>
        </header>
        <div class="field">
          <label>Raggruppa per</label>
          <select class="input select" v-model="folderPreset" :disabled="state.prefs.renameOnly">
            <option v-for="o in FOLDER_PRESETS" :key="o.id" :value="o.id">
              {{ o.label }} — {{ o.example }}{{ o.recommended ? '  ✓ consigliato' : '' }}
            </option>
            <option v-if="folderPreset === CUSTOM_ID" :value="CUSTOM_ID" disabled>
              Personalizzato (impostato nelle opzioni avanzate)
            </option>
          </select>
        </div>
      </section>

      <section>
        <header class="sec-head">
          <h3>Nome dei file</h3>
          <p>Come rinominare ogni foto.</p>
        </header>
        <div class="field">
          <label>Formato nome</label>
          <select class="input select" v-model="filePreset">
            <option v-for="o in FILE_PRESETS" :key="o.id" :value="o.id">
              {{ o.label }} — {{ o.example }}{{ o.recommended ? '  ✓ consigliato' : '' }}
            </option>
            <option v-if="filePreset === CUSTOM_ID" :value="CUSTOM_ID" disabled>
              Personalizzato (impostato nelle opzioni avanzate)
            </option>
          </select>
        </div>
      </section>

      <section>
        <header class="sec-head">
          <h3>Sicurezza</h3>
        </header>
        <div class="grid">
          <Checkbox
            v-model="state.prefs.dryRun"
            @update:modelValue="onCheckChange"
            label="Prova senza modificare i file"
            description="Mostra cosa accadrebbe, senza toccare nulla. Consigliato la prima volta."
          />
          <Checkbox
            v-model="state.prefs.copyMode"
            @update:modelValue="onCheckChange"
            :disabled="state.prefs.renameOnly"
            disabled-reason="Non applicabile con «Rinomina senza spostare»."
            label="Tieni una copia degli originali"
            description="I file restano anche nella cartella di partenza."
          />
          <Checkbox
            v-model="state.prefs.checkDupes"
            @update:modelValue="onCheckChange"
            label="Ignora le foto doppie"
            description="Salta le foto con lo stesso identico contenuto."
          />
        </div>
      </section>

      <!-- Anteprima live -->
      <div class="preview">
        <span class="preview-label">Anteprima</span>
        <div class="preview-path">
          <ArrowRight :size="13" />
          <code>
            <span class="dim">{{ state.prefs.outputDir || state.prefs.inputDir || '/output' }}/</span>
            <span class="hl">jpg/</span>
            <span class="hl">{{ preview.folder }}/</span>
            <span class="strong">{{ preview.file }}</span>
          </code>
        </div>
      </div>

      <div v-if="isCustomLayout" class="custom-note">
        <Info :size="12" />
        Stai usando un formato personalizzato. Modificalo nelle
        <button class="link" @click="state.optionsMode = 'advanced'">opzioni avanzate</button>.
      </div>
    </template>

    <!-- ============ VISTA AVANZATE ============ -->
    <template v-else>
      <!-- Modalità -->
      <section>
        <header class="sec-head">
          <h3>Modalità</h3>
          <p>Come elaborare i file di origine.</p>
        </header>
        <div class="grid">
          <Checkbox
            v-model="state.prefs.dryRun"
            @update:modelValue="onCheckChange"
            label="Prova senza modifiche"
            description="Solo anteprima, nessuna modifica reale ai file."
          />
          <Checkbox
            v-model="state.prefs.copyMode"
            @update:modelValue="onCheckChange"
            :disabled="state.prefs.renameOnly"
            disabled-reason="Non applicabile con «Rinomina senza spostare»."
            label="Tieni una copia degli originali"
            description="Mantieni i file originali nella cartella sorgente."
          />
          <Checkbox
            v-model="state.prefs.renameOnly"
            @update:modelValue="onCheckChange"
            label="Rinomina senza spostare"
            description="Non spostare in sottocartelle: rinomina nella posizione attuale."
          />
          <Checkbox
            v-model="state.prefs.cleanDirs"
            @update:modelValue="onCheckChange"
            :disabled="state.prefs.renameOnly"
            disabled-reason="Con «Rinomina senza spostare» le cartelle non vengono toccate."
            label="Elimina le cartelle rimaste vuote"
            description="Dopo lo spostamento, rimuovi le cartelle svuotate."
          />
        </div>
      </section>

      <!-- Metadati -->
      <section>
        <header class="sec-head">
          <h3>Informazioni e date</h3>
        </header>
        <div class="grid">
          <Checkbox
            v-model="state.prefs.stripMeta"
            @update:modelValue="onCheckChange"
            label="Cancella le informazioni della foto"
            description="Rimuove data, luogo e fotocamera dai file JPEG. I RAW non vengono toccati."
          />
          <Checkbox
            v-model="state.prefs.modTime"
            @update:modelValue="onCheckChange"
            label="Usa la data del file se manca quella della foto"
            description="Quando la foto non ha una data, usa quella di modifica del file."
          />
          <Checkbox
            v-model="state.prefs.checkDupes"
            @update:modelValue="onCheckChange"
            label="Ignora le foto doppie"
            description="Salta le foto con lo stesso identico contenuto."
          />
        </div>
      </section>

      <!-- Struttura -->
      <section>
        <header class="sec-head">
          <h3>Struttura cartelle e nomi</h3>
          <p>Formato avanzato per esperti — <code>2006</code>=anno, <code>01</code>=mese, <code>02</code>=giorno.</p>
        </header>

        <div class="field">
          <label>Cartelle</label>
          <input
            ref="folderInput"
            type="text"
            v-model="state.prefs.folderFmt"
            class="input mono"
            :class="{ 'has-warn': folderWarning?.level === 'warn', 'has-err': folderWarning?.level === 'error' }"
            :disabled="state.prefs.renameOnly"
            placeholder="2006_01_02"
            @focus="lastFocused = 'folder'"
          />
          <Transition name="hint">
            <div v-if="folderWarning" class="hint" :data-level="folderWarning.level" role="alert">
              <component :is="folderWarning.level === 'error' ? AlertTriangle : Info" :size="11" />
              {{ folderWarning.msg }}
            </div>
          </Transition>
        </div>

        <div class="field">
          <label>Nome file</label>
          <input
            ref="fileInput"
            type="text"
            v-model="state.prefs.fileTpl"
            class="input mono"
            :class="{ 'has-warn': fileWarning?.level === 'warn', 'has-err': fileWarning?.level === 'error' }"
            placeholder="photo_{date}_{time}"
            @focus="lastFocused = 'file'"
          />
          <Transition name="hint">
            <div v-if="fileWarning" class="hint" :data-level="fileWarning.level" role="alert">
              <component :is="fileWarning.level === 'error' ? AlertTriangle : Info" :size="11" />
              {{ fileWarning.msg }}
            </div>
          </Transition>
        </div>

        <!-- Segnaposto -->
        <div class="tokens">
          <span class="tokens-label">Segnaposto:</span>
          <button
            v-for="t in TOKENS"
            :key="t"
            class="chip chip-interactive"
            @click="insertToken(t)"
            :title="`Inserisci ${t} nel campo selezionato`"
          >
            <Hash :size="9" />
            {{ t.slice(1, -1) }}
          </button>
        </div>

        <!-- Anteprima live -->
        <div class="preview">
          <span class="preview-label">Anteprima</span>
          <div class="preview-path">
            <ArrowRight :size="13" />
            <code>
              <span class="dim">{{ state.prefs.outputDir || state.prefs.inputDir || '/output' }}/</span>
              <span class="hl">jpg/</span>
              <span class="hl">{{ preview.folder }}/</span>
              <span class="strong">{{ preview.file }}</span>
            </code>
          </div>
        </div>
      </section>

      <!-- Opzioni avanzate RAW -->
      <section>
        <header class="sec-head">
          <h3>Suddivisione RAW</h3>
          <p>Suddivisione extra applicata <strong>solo ai RAW</strong>: gli altri formati non hanno queste informazioni.</p>
        </header>

        <div class="field">
          <label>Suddividi i RAW per</label>
          <select
            class="input select"
            v-model="state.prefs.rawSplit"
            @change="onCheckChange"
            :disabled="state.prefs.renameOnly"
          >
            <option v-for="o in RAW_SPLITS" :key="o.value" :value="o.value">{{ o.label }}</option>
          </select>

          <Transition name="hint">
            <div v-if="state.prefs.renameOnly" class="hint" data-level="info">
              <Info :size="11" />
              Non applicabile con «Rinomina senza spostare».
            </div>
            <div v-else-if="state.prefs.rawSplit" class="hint" data-level="info">
              <Info :size="11" />
              I RAW privi di questa informazione finiranno in <code>sconosciuto/</code>.
            </div>
          </Transition>
        </div>

        <!-- Anteprima struttura RAW -->
        <div class="preview" v-if="state.prefs.rawSplit && !state.prefs.renameOnly">
          <span class="preview-label">Anteprima RAW</span>
          <div class="preview-path">
            <ArrowRight :size="13" />
            <code>
              <span class="dim">{{ state.prefs.outputDir || state.prefs.inputDir || '/output' }}/</span>
              <span class="hl">raw/</span>
              <span class="hl">{{ rawSplitExample }}</span>
              <span class="hl">{{ preview.folder }}/</span>
              <span class="strong">{{ preview.file }}</span>
            </code>
          </div>
        </div>
      </section>

      <!-- Reset -->
      <section class="reset-sec">
        <button class="btn btn-ghost btn-sm" @click="resetPrefs">
          <RotateCcw :size="12" /> Ripristina default
        </button>
      </section>
    </template>

    <!-- Struttura di destinazione (comune a entrambe le viste) -->
    <section v-if="state.prefs.inputDir" class="tree-section">
      <header class="sec-head">
        <h3>
          <FolderTree :size="12" class="tree-title-icon" />
          Struttura di destinazione
          <span v-if="treeLoading" class="tree-loading-dot" title="Calcolo in corso…" />
        </h3>
        <p>Stima le cartelle che verranno create organizzando le tue foto.</p>
      </header>

      <div v-if="treeData" class="tree">
        <div v-for="cat in treeData.categories" :key="cat.name" class="tree-cat">
          <div class="tree-cat-row">
            <span class="tree-cat-name">{{ cat.name }}/</span>
            <span class="tree-cat-count">{{ cat.count }}</span>
          </div>
          <div v-for="folder in cat.folders" :key="folder.path" class="tree-folder-row">
            <span class="tree-folder-indent">↳</span>
            <span class="tree-folder-name">{{ folder.path }}/</span>
            <span class="tree-folder-count">{{ folder.count }}</span>
          </div>
        </div>
        <div v-if="treeData.truncated" class="tree-note">
          Stima basata sulle prime {{ treeData.scanned }} foto analizzate.
        </div>
        <div v-if="!treeData.categories?.length" class="tree-empty">
          Nessuna foto trovata nella cartella di partenza.
        </div>
      </div>
    </section>
  </div>
</template>

<style scoped>
.options-tab {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 24px;
  padding: 20px 24px;
  max-width: 760px;
  width: 100%;
  margin: 0 auto;
  min-height: 0;
  overflow-y: auto;
}

/* Segmented control Semplice / Avanzate */
.mode-switch {
  display: inline-flex;
  align-self: flex-end;
  gap: 2px;
  padding: 3px;
  background: hsl(var(--subtle));
  border: 1px solid hsl(var(--border));
  border-radius: 8px;
}
.seg {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  height: 28px;
  padding: 0 12px;
  border: none;
  background: transparent;
  color: hsl(var(--muted));
  font-size: 12px;
  font-weight: 500;
  border-radius: 6px;
  cursor: pointer;
  transition: all .12s;
}
.seg:hover { color: hsl(var(--text)); }
.seg.active {
  background: hsl(var(--bg));
  color: hsl(var(--text));
  box-shadow: 0 1px 2px rgba(0,0,0,.06);
}

section {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.sec-head h3 {
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: .06em;
  color: hsl(var(--muted));
  margin: 0;
}
.sec-head p {
  font-size: 11.5px;
  color: hsl(var(--muted));
  margin: 4px 0 0;
}
.sec-head p code {
  background: hsl(var(--subtle));
  border: 1px solid hsl(var(--border));
  border-radius: 3px;
  padding: 0 4px;
  font-size: 10.5px;
  font-family: var(--font-mono, monospace);
}
.grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 4px 24px;
}
.field {
  display: flex;
  flex-direction: column;
  gap: 5px;
}
.field label {
  font-size: 11.5px;
  color: hsl(var(--muted));
  font-weight: 500;
}
.input.mono {
  font-family: var(--font-mono, monospace);
  font-size: 12.5px;
}
.input.has-warn { border-color: hsl(var(--warning) / .6); }
.input.has-warn:focus { border-color: hsl(var(--warning)); }
.input.has-err  { border-color: hsl(var(--danger)  / .6); }
.input.has-err:focus  { border-color: hsl(var(--danger)); }

.input.select {
  cursor: pointer;
  appearance: none;
  -webkit-appearance: none;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 24 24' fill='none' stroke='%23888' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpolyline points='6 9 12 15 18 9'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 10px center;
  padding-right: 30px;
}
.input.select:disabled { cursor: not-allowed; opacity: .55; }

.hint {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 11px;
  margin-top: 2px;
  line-height: 1.4;
}
.hint[data-level="warn"]  { color: hsl(var(--warning)); }
.hint[data-level="error"] { color: hsl(var(--danger)); }
.hint-enter-active, .hint-leave-active { transition: opacity .15s, transform .15s; }
.hint-enter-from, .hint-leave-to { opacity: 0; transform: translateY(-2px); }
.tokens {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 4px;
  margin-top: 2px;
}
.tokens-label {
  font-size: 11px;
  color: hsl(var(--muted));
  margin-right: 4px;
}
.preview {
  margin-top: 8px;
  padding: 10px 12px;
  background: hsl(var(--subtle));
  border: 1px solid hsl(var(--border));
  border-radius: 8px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.preview-label {
  font-size: 10.5px;
  color: hsl(var(--muted));
  text-transform: uppercase;
  letter-spacing: .08em;
  font-weight: 600;
}
.preview-path {
  display: flex;
  align-items: center;
  gap: 6px;
  color: hsl(var(--muted));
}
.preview-path code {
  font-family: var(--font-mono, monospace);
  font-size: 12px;
  word-break: break-all;
  line-height: 1.5;
}
.dim { color: hsl(var(--muted)); }
.hl { color: hsl(var(--text)); }
.strong { color: hsl(var(--accent)); font-weight: 600; }

.custom-note {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11.5px;
  color: hsl(var(--muted));
}
.link {
  background: none;
  border: none;
  padding: 0;
  color: hsl(var(--accent));
  font-size: inherit;
  cursor: pointer;
  text-decoration: underline;
}

.reset-sec {
  border-top: 1px solid hsl(var(--border));
  padding-top: 16px;
  margin-top: 4px;
}

/* Struttura di destinazione */
.tree-section {
  border-top: 1px solid hsl(var(--border));
  padding-top: 16px;
}
.sec-head h3 {
  display: flex;
  align-items: center;
  gap: 5px;
}
.tree-title-icon { color: hsl(var(--muted)); flex-shrink: 0; }
.tree-loading-dot {
  display: inline-block;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: hsl(var(--accent));
  animation: pulse-dot 1s ease-in-out infinite;
  margin-left: 2px;
}
@keyframes pulse-dot {
  0%, 100% { opacity: .4; transform: scale(.85); }
  50%       { opacity: 1;  transform: scale(1.1); }
}
.tree {
  background: hsl(var(--subtle));
  border: 1px solid hsl(var(--border));
  border-radius: 8px;
  padding: 10px 14px;
  display: flex;
  flex-direction: column;
  gap: 2px;
  font-variant-numeric: tabular-nums;
}
.tree-cat { margin-bottom: 4px; }
.tree-cat:last-child { margin-bottom: 0; }
.tree-cat-row {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 600;
  color: hsl(var(--text));
}
.tree-cat-name { flex: 1; font-family: var(--font-mono, monospace); }
.tree-cat-count {
  font-size: 11px;
  color: hsl(var(--muted));
  background: hsl(var(--bg));
  border: 1px solid hsl(var(--border));
  border-radius: 999px;
  padding: 1px 7px;
}
.tree-folder-row {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11.5px;
  color: hsl(var(--muted));
  padding-left: 6px;
}
.tree-folder-indent { color: hsl(var(--border)); font-size: 12px; }
.tree-folder-name { flex: 1; font-family: var(--font-mono, monospace); }
.tree-folder-count { font-size: 11px; }
.tree-note {
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px solid hsl(var(--border));
  font-size: 11px;
  color: hsl(var(--muted));
  font-style: italic;
}
.tree-empty { font-size: 12px; color: hsl(var(--muted)); }
</style>
