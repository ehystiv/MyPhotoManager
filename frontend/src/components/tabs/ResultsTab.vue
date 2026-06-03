<script setup>
import { computed, ref, onMounted, watch } from 'vue'
import { useStore } from '../../composables/useStore'
import {
  FolderOpen, Image, Layers, SkipForward, Copy as CopyIcon, Trash2, BarChart3, FolderTree, History,
} from '@lucide/vue'
import { vAutoAnimate } from '@formkit/auto-animate/vue'
import { GetHistory } from '../../../wailsjs/go/main/App'

const { state, openOutputInFinder } = useStore()

const stats = computed(() => state.stats)
const history = ref([])

async function loadHistory() {
  try {
    history.value = await GetHistory()
  } catch (e) {
    console.error('GetHistory failed', e)
  }
}

onMounted(loadHistory)
watch(() => state.stats, (v) => { if (v?.moved > 0) loadHistory() })

const totals = computed(() => {
  const s = stats.value
  if (!s) return []
  return [
    { key: 'moved',    label: 'Organizzate', value: s.moved,    icon: Image,    color: 'accent' },
    { key: 'raw',      label: 'RAW',         value: s.raw,      icon: Layers,   color: 'text' },
    { key: 'others',   label: 'Altri',       value: s.others,   icon: Image,    color: 'text' },
    { key: 'skipped',  label: 'Senza data',  value: s.skipped,  icon: SkipForward, color: 'warning' },
    { key: 'dupes',    label: 'Duplicati',   value: s.dupes,    icon: CopyIcon, color: 'muted' },
    { key: 'cleaned',  label: 'Cartelle vuote', value: s.cleaned, icon: Trash2, color: 'muted' },
    { key: 'migrated', label: 'Migrati',     value: s.migrated, icon: FolderTree, color: 'muted' },
  ].filter(t => t.value > 0)
})

const maxYear = computed(() => {
  if (!stats.value?.byYear?.length) return 1
  return Math.max(...stats.value.byYear.map(y => y.count))
})

const maxCat = computed(() => {
  if (!stats.value?.byCategory?.length) return 1
  return Math.max(...stats.value.byCategory.map(c => c.count))
})

function formatRunDate(dateStr) {
  const d = new Date(dateStr)
  const now = new Date()
  const diffDays = (now - d) / 1000 / 60 / 60 / 24
  const time = d.toLocaleTimeString('it-IT', { hour: '2-digit', minute: '2-digit' })
  if (diffDays < 1) return `Oggi ${time}`
  if (diffDays < 2) return `Ieri ${time}`
  return d.toLocaleDateString('it-IT', { day: '2-digit', month: 'short' }) + ` ${time}`
}

function basename(path) {
  return path.split('/').filter(Boolean).pop() || path
}
</script>

<template>
  <div class="results-tab" v-if="stats || history.length">
    <!-- Risultato corrente -->
    <template v-if="stats">
      <header class="head">
        <h2>Risultato</h2>
        <button class="btn btn-secondary btn-md" @click="openOutputInFinder">
          <FolderOpen :size="13" /> Apri nel Finder
        </button>
      </header>

      <div class="totals" v-auto-animate>
        <div v-for="t in totals" :key="t.key" class="total-card" :data-color="t.color">
          <component :is="t.icon" :size="14" class="total-icon" />
          <div class="total-num">{{ t.value }}</div>
          <div class="total-label">{{ t.label }}</div>
        </div>
      </div>

      <div class="years" v-if="stats.byCategory?.length">
        <div class="years-head">
          <FolderTree :size="13" />
          <span>Per tipo</span>
        </div>
        <div class="years-list" v-auto-animate>
          <div v-for="c in stats.byCategory" :key="c.category" class="year-row">
            <span class="year cat">{{ c.category }}/</span>
            <div class="bar-wrap">
              <div class="bar" :style="{ width: (c.count / maxCat * 100) + '%' }"></div>
            </div>
            <span class="count">{{ c.count }}</span>
          </div>
        </div>
      </div>

      <div class="years" v-if="stats.byYear?.length">
        <div class="years-head">
          <BarChart3 :size="13" />
          <span>Per anno</span>
        </div>
        <div class="years-list" v-auto-animate>
          <div v-for="y in stats.byYear" :key="y.year" class="year-row">
            <span class="year">{{ y.year }}</span>
            <div class="bar-wrap">
              <div class="bar" :style="{ width: (y.count / maxYear * 100) + '%' }"></div>
            </div>
            <span class="count">{{ y.count }}</span>
          </div>
        </div>
      </div>
    </template>

    <!-- Storico esecuzioni -->
    <div class="history" v-if="history.length">
      <div class="years-head">
        <History :size="13" />
        <span>Ultime esecuzioni</span>
      </div>
      <div class="history-list">
        <div v-for="(run, i) in history.slice(0, 15)" :key="i" class="history-row">
          <span class="history-date">{{ formatRunDate(run.runAt) }}</span>
          <span class="history-dir" :title="run.inputDir">{{ basename(run.inputDir) }}</span>
          <span class="history-count">{{ run.moved }} foto</span>
        </div>
      </div>
    </div>
  </div>

  <div v-else class="empty">
    <BarChart3 :size="32" />
    <div class="empty-title">Nessun risultato</div>
    <div class="empty-desc">I risultati dell'organizzazione compariranno qui.</div>
  </div>
</template>

<style scoped>
.results-tab {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 22px;
  padding: 24px;
  max-width: 760px;
  width: 100%;
  margin: 0 auto;
  overflow-y: auto;
}
.head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.head h2 {
  font-size: 16px;
  font-weight: 600;
  color: hsl(var(--text));
  margin: 0;
}
.totals {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: 10px;
}
.total-card {
  background: hsl(var(--surface));
  border: 1px solid hsl(var(--border));
  border-radius: 10px;
  padding: 12px 14px;
  display: flex;
  flex-direction: column;
  gap: 2px;
  position: relative;
}
.total-icon {
  color: hsl(var(--muted));
  margin-bottom: 4px;
}
.total-num {
  font-size: 22px;
  font-weight: 600;
  color: hsl(var(--text));
  font-variant-numeric: tabular-nums;
  line-height: 1.1;
}
.total-label {
  font-size: 11px;
  color: hsl(var(--muted));
}
.total-card[data-color="accent"] .total-icon,
.total-card[data-color="accent"] .total-num { color: hsl(var(--accent)); }
.total-card[data-color="warning"] .total-icon,
.total-card[data-color="warning"] .total-num { color: hsl(var(--warning)); }
.total-card[data-color="muted"] .total-num { color: hsl(var(--muted)); }

.years {
  background: hsl(var(--surface));
  border: 1px solid hsl(var(--border));
  border-radius: 10px;
  padding: 14px 16px;
}
.years-head {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: .06em;
  color: hsl(var(--muted));
  margin-bottom: 10px;
}
.years-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.year-row {
  display: grid;
  grid-template-columns: 48px 1fr 60px;
  align-items: center;
  gap: 10px;
  font-size: 12.5px;
}
.year {
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  color: hsl(var(--text));
}
.year.cat { font-family: var(--font-mono, monospace); font-size: 11.5px; }
.bar-wrap {
  height: 6px;
  background: hsl(var(--subtle));
  border-radius: 3px;
  overflow: hidden;
}
.bar {
  height: 100%;
  background: linear-gradient(90deg, hsl(var(--accent) / .6), hsl(var(--accent)));
  border-radius: 3px;
  transition: width .35s ease-out;
}
.count {
  text-align: right;
  font-variant-numeric: tabular-nums;
  color: hsl(var(--muted));
  font-size: 12px;
}

/* Storico */
.history {
  background: hsl(var(--surface));
  border: 1px solid hsl(var(--border));
  border-radius: 10px;
  padding: 14px 16px;
}
.history-list {
  display: flex;
  flex-direction: column;
  gap: 5px;
}
.history-row {
  display: grid;
  grid-template-columns: 110px 1fr auto;
  align-items: center;
  gap: 10px;
  font-size: 12px;
  padding: 3px 0;
}
.history-date {
  font-size: 11px;
  color: hsl(var(--muted));
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}
.history-dir {
  color: hsl(var(--text));
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.history-count {
  font-size: 11.5px;
  color: hsl(var(--accent));
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
  font-weight: 600;
}

.empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  color: hsl(var(--muted));
  padding: 40px;
}
.empty-title {
  font-size: 13px;
  font-weight: 500;
  color: hsl(var(--text));
  margin-top: 10px;
}
.empty-desc { font-size: 12px; text-align: center; }
</style>
