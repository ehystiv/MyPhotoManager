<script setup>
import { ref, watch, onMounted, nextTick, computed } from 'vue'
import { useDebounceFn } from '@vueuse/core'
import { useStore } from '../../composables/useStore'
import { FormatPreview } from '../../../wailsjs/go/main/App'
import { RotateCcw, ArrowRight, Hash, AlertTriangle, Info } from '@lucide/vue'
import Checkbox from '../Checkbox.vue'

const { state, persist, resetPrefs } = useStore()

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

const folderWarning = computed(() => {
  const v = state.prefs.folderFmt || ''
  if (state.prefs.renameOnly) return ''
  if (!v.trim()) return { level: 'error', msg: 'Formato cartella vuoto.' }
  const hasDate = /2006|01|02/.test(v)
  if (!hasDate) {
    return { level: 'error', msg: 'Nessun placeholder data: tutte le foto finirebbero in una sola cartella.' }
  }
  return ''
})

const fileWarning = computed(() => {
  const v = state.prefs.fileTpl || ''
  if (!v.trim()) return { level: 'error', msg: 'Template nome file vuoto.' }
  const hasUniq = /\{date\}|\{time\}|\{datetime\}/.test(v)
  if (!hasUniq) {
    return { level: 'warn', msg: 'Nessun token data/ora: rischio di nomi duplicati (verrà aggiunto _1, _2…).' }
  }
  if (!/\{(date|time|datetime|year|month|day|camera)\}/.test(v)) {
    return { level: 'warn', msg: 'Nessun token: tutti i file avranno lo stesso nome.' }
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
          label="Dry-run"
          description="Solo anteprima, nessuna modifica reale ai file."
        />
        <Checkbox
          v-model="state.prefs.copyMode"
          @update:modelValue="onCheckChange"
          :disabled="state.prefs.renameOnly"
          disabled-reason="Non applicabile in modalità Rinomina in-place."
          label="Copia"
          description="Mantieni i file originali nella cartella sorgente."
        />
        <Checkbox
          v-model="state.prefs.renameOnly"
          @update:modelValue="onCheckChange"
          label="Rinomina in-place"
          description="Non spostare in sottocartelle: rinomina nella posizione attuale."
        />
        <Checkbox
          v-model="state.prefs.cleanDirs"
          @update:modelValue="onCheckChange"
          :disabled="state.prefs.renameOnly"
          disabled-reason="In rinomina in-place le cartelle non vengono toccate."
          label="Rimuovi cartelle vuote"
          description="Dopo lo spostamento, elimina le directory rimaste vuote."
        />
      </div>
    </section>

    <!-- Metadati -->
    <section>
      <header class="sec-head">
        <h3>Metadati e date</h3>
      </header>
      <div class="grid">
        <Checkbox
          v-model="state.prefs.stripMeta"
          @update:modelValue="onCheckChange"
          label="Rimuovi EXIF dai JPEG"
          description="Cancella metadati EXIF/XMP/IPTC nei file JPEG. RAW non toccati."
        />
        <Checkbox
          v-model="state.prefs.modTime"
          @update:modelValue="onCheckChange"
          label="Data da filesystem"
          description="Se EXIF mancante, usa la data di modifica del file."
        />
        <Checkbox
          v-model="state.prefs.checkDupes"
          @update:modelValue="onCheckChange"
          label="Salta duplicati"
          description="Confronto SHA-256: i file identici vengono ignorati."
        />
      </div>
    </section>

    <!-- Struttura -->
    <section>
      <header class="sec-head">
        <h3>Struttura cartelle e nomi</h3>
        <p>Pattern Go time per data: <code>2006</code>=anno, <code>01</code>=mese, <code>02</code>=giorno.</p>
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

      <!-- Token chip -->
      <div class="tokens">
        <span class="tokens-label">Token:</span>
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

    <!-- Opzioni avanzate -->
    <section>
      <header class="sec-head">
        <h3>Opzioni avanzate</h3>
        <p>Suddivisione extra applicata <strong>solo ai RAW</strong>: gli altri formati non hanno questi metadati.</p>
      </header>

      <div class="field">
        <label>Suddividi i RAW per metadato</label>
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
            Non applicabile in modalità Rinomina in-place.
          </div>
          <div v-else-if="state.prefs.rawSplit" class="hint" data-level="info">
            <Info :size="11" />
            I RAW privi di questo metadato finiranno in <code>sconosciuto/</code>.
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
.reset-sec {
  border-top: 1px solid hsl(var(--border));
  padding-top: 16px;
  margin-top: 4px;
}
</style>
