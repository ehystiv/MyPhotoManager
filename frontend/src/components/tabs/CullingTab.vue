<script setup>
import { ref, reactive, computed, watch, onMounted, onUnmounted } from 'vue'
import {
  Trash2, AlertTriangle, Check, ChevronLeft, ChevronRight,
  Images, RotateCcw, Play, RefreshCw,
} from '@lucide/vue'
import { useStore } from '../../composables/useStore'
import { PhotoData, PhotoMeta } from '../../../wailsjs/go/main/App'
import ConfirmDialog from '../ConfirmDialog.vue'
import { vAutoAnimate } from '@formkit/auto-animate/vue'

const { state, loadCulling, markPhoto, applyCulling, resetCulling } = useStore()

const confirmApplyOpen = ref(false)
const confirmResetOpen = ref(false)

const photos = computed(() => state.cullingPhotos)
const total = computed(() => photos.value.length)
const current = computed(() => photos.value[state.cullingIndex] || null)

// Cache delle immagini (path → data-URL). '' = in caricamento/non disponibile.
const srcCache = reactive({})

// Metadati EXIF della foto corrente.
const meta = ref(null)

async function loadMeta(path) {
  meta.value = null
  try {
    meta.value = await PhotoMeta(path)
  } catch (e) {
    console.error('PhotoMeta failed', e)
  }
}

async function loadAt(idx) {
  const p = photos.value[idx]
  if (!p || p.path in srcCache) return
  srcCache[p.path] = '' // segna come in caricamento, evita richieste doppie
  try {
    srcCache[p.path] = await PhotoData(p.path)
  } catch (e) {
    srcCache[p.path] = ''
    console.error('PhotoData failed', e)
  }
}

const currentSrc = computed(() => current.value ? srcCache[current.value.path] : '')

// Carica la foto corrente e precarica la successiva quando cambia l'indice.
watch(current, () => {
  if (!current.value) return
  loadAt(state.cullingIndex)
  loadAt(state.cullingIndex + 1)
  loadMeta(current.value.path)
}, { immediate: true })

const counts = computed(() => {
  const c = { delete: 0, review: 0, ok: 0, todo: 0 }
  for (const p of photos.value) {
    if (p.mark === 'delete') c.delete++
    else if (p.mark === 'review') c.review++
    else if (p.mark === 'ok') c.ok++
    else c.todo++
  }
  return c
})

const reviewed = computed(() => total.value - counts.value.todo)
const pct = computed(() => total.value ? Math.round((reviewed.value / total.value) * 100) : 0)
const hasMarks = computed(() => counts.value.delete + counts.value.review + counts.value.ok > 0)

const markLabels = {
  delete: { text: 'Elimina', cls: 'danger' },
  review: { text: 'Da correggere', cls: 'warning' },
  ok: { text: 'OK', cls: 'success' },
}

function go(delta) {
  const n = state.cullingIndex + delta
  if (n >= 0 && n < total.value) state.cullingIndex = n
}

function mark(m) {
  if (!current.value) return
  // Ri-cliccando la stessa categoria si rimuove la marcatura.
  const next = current.value.mark === m ? '' : m
  markPhoto(current.value.path, next)
  if (next && state.cullingIndex < total.value - 1) state.cullingIndex++
}

function onKey(e) {
  if (state.activeTab !== 'culling') return
  const tag = e.target?.tagName
  if (tag === 'INPUT' || tag === 'TEXTAREA') return
  switch (e.key) {
    case 'ArrowLeft':  e.preventDefault(); go(-1); break
    case 'ArrowRight': e.preventDefault(); go(1); break
    case '1': case 'd': e.preventDefault(); mark('delete'); break
    case '2': case 'f': e.preventDefault(); mark('review'); break
    case '3': case 'k': e.preventDefault(); mark('ok'); break
  }
}

function onApply() {
  if (state.prefs.dryRun) {
    applyCulling()
  } else {
    confirmApplyOpen.value = true
  }
}

onMounted(() => {
  loadCulling()
  window.addEventListener('keydown', onKey)
})
onUnmounted(() => window.removeEventListener('keydown', onKey))
</script>

<template>
  <div class="culling-tab">
    <header class="head">
      <div class="head-text">
        <h2>Revisione foto</h2>
        <p>Scorri le foto e decidi: elimina, da correggere, o tieni così. Niente viene spostato finché non premi «Applica».</p>
      </div>
      <button class="btn btn-ghost btn-sm" @click="loadCulling" :disabled="state.cullingLoading" title="Ricarica le foto">
        <RefreshCw :size="13" :class="{ spin: state.cullingLoading }" /> Aggiorna
      </button>
    </header>

    <!-- Stato vuoto -->
    <div v-if="!total" class="empty">
      <Images :size="34" />
      <div class="empty-title">
        {{ state.cullingLoading ? 'Caricamento…' : 'Nessuna foto da rivedere' }}
      </div>
      <div class="empty-sub" v-if="!state.cullingLoading">
        Vengono mostrate le foto JPEG, PNG e WEBP nella cartella di output.
      </div>
    </div>

    <template v-else>
      <!-- Avanzamento -->
      <div class="progress-head">
        <span class="counter">{{ state.cullingIndex + 1 }} / {{ total }}</span>
        <div class="bar"><div class="bar-fill" :style="{ width: pct + '%' }" /></div>
        <span class="counter muted">{{ reviewed }} riviste</span>
      </div>

      <!-- Visualizzatore -->
      <div class="viewer">
        <button class="navbtn" @click="go(-1)" :disabled="state.cullingIndex === 0" title="Precedente (←)">
          <ChevronLeft :size="20" />
        </button>

        <div class="photo-frame">
          <img v-if="currentSrc" :key="current.path" :src="currentSrc" :alt="current.name" />
          <div v-else class="loading-ph"><RefreshCw :size="22" class="spin" /></div>
          <span
            v-if="current?.mark"
            class="mark-badge"
            :class="`badge-${markLabels[current.mark].cls}`"
          >{{ markLabels[current.mark].text }}</span>
        </div>

        <button class="navbtn" @click="go(1)" :disabled="state.cullingIndex >= total - 1" title="Successiva (→)">
          <ChevronRight :size="20" />
        </button>
      </div>

      <div class="filename" v-if="current">{{ current.rel }}</div>

      <!-- Metadati EXIF -->
      <div v-if="meta" class="exif-strip">
        <span v-if="meta.date"             class="exif-chip" title="Data e ora di scatto">{{ meta.date }}</span>
        <span v-if="meta.camera"           class="exif-chip" title="Fotocamera (marca + modello)">{{ meta.camera }}</span>
        <span v-if="meta.focal"            class="exif-chip" title="Lunghezza focale">{{ meta.focal }}</span>
        <span v-if="meta.aperture"         class="exif-chip" title="Apertura diaframma">{{ meta.aperture }}</span>
        <span v-if="meta.shutter"          class="exif-chip" title="Tempo di esposizione">{{ meta.shutter }}s</span>
        <span v-if="meta.iso"              class="exif-chip" title="Sensibilità ISO">ISO {{ meta.iso }}</span>
        <span v-if="meta.bias"             class="exif-chip" title="Compensazione esposizione">{{ meta.bias }}</span>
        <span v-if="meta.program"          class="exif-chip" title="Modalità di esposizione">{{ meta.program }}</span>
        <span v-if="meta.metering"         class="exif-chip" title="Modalità di misurazione esposizione">{{ meta.metering }}</span>
        <span v-if="meta.maxAp"            class="exif-chip" title="Apertura massima dell'obiettivo">{{ meta.maxAp }}</span>
        <span v-if="meta.width && meta.height" class="exif-chip" title="Dimensioni in pixel">{{ meta.width }}×{{ meta.height }}</span>
        <span v-if="meta.lens"             class="exif-chip exif-lens" :title="'Obiettivo: ' + meta.lens">{{ meta.lens }}</span>
        <span v-if="meta.gps"              class="exif-chip" :title="'Coordinate GPS: ' + meta.gps">GPS</span>
        <span v-if="meta.flash"            class="exif-chip" title="Flash scattato">Flash</span>
      </div>

      <!-- Azioni di marcatura -->
      <div class="actions">
        <button class="btn btn-danger btn-lg" :class="{ active: current?.mark === 'delete' }" @click="mark('delete')">
          <Trash2 :size="15" /> Elimina <kbd class="hint-kbd">1</kbd>
        </button>
        <button class="btn btn-warning btn-lg" :class="{ active: current?.mark === 'review' }" @click="mark('review')">
          <AlertTriangle :size="15" /> Da correggere <kbd class="hint-kbd">2</kbd>
        </button>
        <button class="btn btn-success btn-lg" :class="{ active: current?.mark === 'ok' }" @click="mark('ok')">
          <Check :size="15" /> OK <kbd class="hint-kbd">3</kbd>
        </button>
      </div>

      <!-- Riepilogo + applica -->
      <div class="summary">
        <div class="tags">
          <span class="tag tag-danger">{{ counts.delete }} da eliminare</span>
          <span class="tag tag-warning">{{ counts.review }} da correggere</span>
          <span class="tag tag-success">{{ counts.ok }} ok</span>
          <span class="tag">{{ counts.todo }} da vedere</span>
        </div>
        <div class="summary-actions">
          <button class="btn btn-ghost btn-md" @click="confirmResetOpen = true" :disabled="!hasMarks">
            <RotateCcw :size="13" /> Azzera
          </button>
          <button class="btn btn-primary btn-md" @click="onApply" :disabled="!hasMarks">
            <Play :size="13" /> {{ state.prefs.dryRun ? 'Anteprima' : 'Applica' }}
          </button>
        </div>
      </div>

      <!-- Esito ultima applicazione -->
      <div v-auto-animate>
        <div v-if="state.cullingResult" class="result">
          <div class="row">
            <span class="label">Eliminate</span>
            <span class="val">{{ state.cullingResult.deleted }}</span>
          </div>
          <div class="row">
            <span class="label">Spostate in «da correggere»</span>
            <span class="val">{{ state.cullingResult.moved }}</span>
          </div>
          <div class="row">
            <span class="label">Tenute</span>
            <span class="val">{{ state.cullingResult.kept }}</span>
          </div>
          <div class="row" v-if="state.cullingResult.errors">
            <span class="label">Errori</span>
            <span class="val danger">{{ state.cullingResult.errors }}</span>
          </div>
          <div v-if="state.cullingResult.dryRun" class="hint">
            <AlertTriangle :size="12" />
            Anteprima — i file non sono stati toccati. Disattiva «Prova senza modificare» per applicare davvero.
          </div>
        </div>
      </div>
    </template>

    <ConfirmDialog
      v-model:open="confirmApplyOpen"
      title="Applicare le decisioni?"
      description="Le foto «Elimina» finiranno nel cestino di sistema, quelle «Da correggere» saranno spostate nella cartella _da_correggere. Le foto «OK» restano dove sono."
      confirm-text="Applica"
      :destructive="true"
      @confirm="applyCulling"
    />
    <ConfirmDialog
      v-model:open="confirmResetOpen"
      title="Azzerare tutte le marcature?"
      description="Le decisioni prese verranno dimenticate. I file non vengono toccati."
      confirm-text="Azzera"
      @confirm="resetCulling"
    />
  </div>
</template>

<style scoped>
.culling-tab {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 20px 24px;
  width: 100%;
  min-height: 0;
  overflow-y: auto;
}
.head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}
.head h2 {
  font-size: 16px;
  font-weight: 600;
  color: hsl(var(--text));
  margin: 0 0 6px 0;
}
.head p {
  font-size: 12.5px;
  color: hsl(var(--muted));
  margin: 0;
  line-height: 1.5;
}
.spin { animation: spin 1s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }

/* Stato vuoto */
.empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: hsl(var(--muted));
  text-align: center;
  padding: 40px 0;
}
.empty-title { font-size: 14px; font-weight: 600; color: hsl(var(--text)); }
.empty-sub { font-size: 12px; }

/* Avanzamento */
.progress-head {
  display: flex;
  align-items: center;
  gap: 12px;
}
.counter {
  font-size: 12px;
  font-variant-numeric: tabular-nums;
  color: hsl(var(--text));
  white-space: nowrap;
}
.counter.muted { color: hsl(var(--muted)); }
.bar {
  flex: 1;
  height: 6px;
  border-radius: 999px;
  background: hsl(var(--surface));
  border: 1px solid hsl(var(--border));
  overflow: hidden;
}
.bar-fill {
  height: 100%;
  background: hsl(var(--accent));
  transition: width .2s ease;
}

/* Visualizzatore */
.viewer {
  flex: 1;
  min-height: 0;
  display: flex;
  align-items: stretch;
  gap: 10px;
}
.navbtn {
  flex-shrink: 0;
  align-self: center;
  width: 38px;
  height: 38px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  border: 1px solid hsl(var(--border));
  background: hsl(var(--surface));
  color: hsl(var(--text));
  cursor: pointer;
}
.navbtn:hover:not(:disabled) { background: hsl(var(--bg)); border-color: hsl(var(--accent)); }
.navbtn:disabled { opacity: .35; cursor: default; }
.photo-frame {
  position: relative;
  flex: 1;
  min-height: 0;
  background: hsl(var(--surface));
  border: 1px solid hsl(var(--border));
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}
.photo-frame img {
  width: 100%;
  height: 100%;
  object-fit: contain;
  display: block;
}
.loading-ph {
  display: flex;
  align-items: center;
  justify-content: center;
  color: hsl(var(--muted));
}
.mark-badge {
  position: absolute;
  top: 10px;
  left: 10px;
  padding: 3px 10px;
  border-radius: 6px;
  font-size: 11px;
  font-weight: 600;
  color: #fff;
}
.badge-danger  { background: hsl(var(--danger)); }
.badge-warning { background: hsl(var(--warning)); }
.badge-success { background: hsl(var(--success)); }

.filename {
  text-align: center;
  font-size: 11.5px;
  color: hsl(var(--muted));
  font-variant-numeric: tabular-nums;
  word-break: break-all;
}

/* Azioni */
.actions {
  display: flex;
  gap: 10px;
  justify-content: center;
}
.actions .btn { position: relative; }
.actions .btn.active {
  outline: 2px solid hsl(var(--text));
  outline-offset: 2px;
}
.hint-kbd {
  font-size: 9.5px;
  opacity: .7;
  padding: 1px 4px;
  border-radius: 3px;
  background: rgba(255,255,255,.2);
  margin-left: 2px;
}

/* Riepilogo */
.summary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}
.tags { display: flex; gap: 6px; flex-wrap: wrap; }
.tag {
  font-size: 11px;
  padding: 3px 9px;
  border-radius: 999px;
  background: hsl(var(--surface));
  border: 1px solid hsl(var(--border));
  color: hsl(var(--muted));
  font-variant-numeric: tabular-nums;
}
.tag-danger  { color: hsl(var(--danger));  border-color: hsl(var(--danger) / .35);  background: hsl(var(--danger) / .08); }
.tag-warning { color: hsl(var(--warning)); border-color: hsl(var(--warning) / .35); background: hsl(var(--warning) / .08); }
.tag-success { color: hsl(var(--success)); border-color: hsl(var(--success) / .35); background: hsl(var(--success) / .08); }
.summary-actions { display: flex; gap: 8px; }

/* Esito */
.result {
  background: hsl(var(--surface));
  border: 1px solid hsl(var(--border));
  border-radius: 10px;
  padding: 14px 16px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  font-size: 13px;
}
.label { color: hsl(var(--muted)); font-size: 12px; }
.val { color: hsl(var(--text)); font-weight: 500; font-variant-numeric: tabular-nums; }
.val.danger { color: hsl(var(--danger)); }
.hint {
  margin-top: 4px;
  padding: 8px 10px;
  background: hsl(var(--warning) / .1);
  border: 1px solid hsl(var(--warning) / .3);
  border-radius: 6px;
  font-size: 11.5px;
  color: hsl(var(--warning));
  display: flex;
  align-items: center;
  gap: 6px;
}

/* Strip metadati EXIF */
.exif-strip {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
  justify-content: center;
}
.exif-chip {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 999px;
  background: hsl(var(--surface));
  border: 1px solid hsl(var(--border));
  color: hsl(var(--muted));
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}
.exif-lens {
  max-width: 220px;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
